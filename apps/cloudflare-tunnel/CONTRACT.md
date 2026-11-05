# Cloudflare Tunnel for fnOS — MVP contract

This file is the single source of truth for parallel work. Do not change it without
coordinating with the integrator. Product decisions are frozen here.

## 1. Product scope (MVP)

Native fnOS app (`cloudflare-tunnel`) that supervises one **token-based, remotely
managed** cloudflared tunnel and manages it through the Cloudflare API.

In scope:

- account settings (Account ID + API Token), token verification
- remote tunnel list / create / delete / token
- ingress routes per tunnel (read, edit, reorder, save) + automatic CNAME creation
- connect / disconnect the selected tunnel, live status, auto-reconnect
- live cloudflared logs (SSE), filters, download, clear
- API keys for future HTTP API / MCP clients (hashed at rest)
- demo mode for UI work and browser E2E without a real Cloudflare account
- fnOS light/dark theme, responsive 620–1440 px

Out of scope for MVP (interface must not block them):

- locally managed tunnels / credentials files
- more than one simultaneously running tunnel
- WARP routing, Access applications, multi-account switching
- MCP server implementation (HTTP API + DTOs are the foundation)

## 2. Directory ownership

```
apps/cloudflare-tunnel/
├── CONTRACT.md          integrator
├── prototype/           read-only visual reference (do not edit)
├── server/              backend agent only
├── web/                 frontend agent only
└── fpk/                 packaging agent only
```

No agent writes outside its directory. Root-level integration scripts are created by
the integrator after the three streams land.

## 3. Runtime layout and names

- appname: `cloudflare-tunnel`
- service port: `18960` (`manifest service_port`; `$TRIM_SERVICE_PORT` wins at runtime)
- binary: `cloudflare-tunnel-server`
- embedded cloudflared: `bin/cloudflared`
- share name: `cloudflare-tunnel`
- share root at runtime: `/var/apps/cloudflare-tunnel/shares/cloudflare-tunnel`
- data (persistent, visible in File Manager):
  - `<share>/config/settings.json` (0600) — account id, api token, prefs
  - `<share>/config/apikeys.json` (0600) — key metadata + SHA-256 hashes
  - `<share>/data/state.json` — last tunnel / last state for boot reconnect
  - `<share>/data/activity.json` — last 100 activity items
- @appdata (`$TRIM_PKGVAR`, ephemeral):
  - `logs/app.log` — backend stdout/stderr
  - `logs/cloudflared.log` — cloudflared log mirror (JSON lines)
  - `app.pid`
- env passed by `cmd/main`:
  - `CT_LISTEN=0.0.0.0:$TRIM_SERVICE_PORT`
  - `CT_CONFIG_DIR=<share>/config`
  - `CT_DATA_DIR=<share>/data`
  - `CT_LOG_DIR=$TRIM_PKGVAR/logs`
  - `CT_CLOUDFLARED=$APP_DIR/bin/cloudflared`
  - `CT_METRICS_PORT=20241` (loopback only)
- backend flags mirror the env (`-listen`, `-config-dir`, `-data-dir`, `-log-dir`,
  `-cloudflared`, `-metrics-port`, `-demo`). Flags win over env.

Secrets never go to logs, SSE, activity, or error messages. `settings.json` and
`apikeys.json` are written with mode 0600. The API token is never returned by any
endpoint; only `apiTokenSet` + last 4 characters.

## 4. cloudflared process model

One child process at a time. The token goes through the environment variable
`TUNNEL_TOKEN` (never argv, so it is not visible in `ps`).

```
cloudflared tunnel --no-autoupdate run \
  --metrics 127.0.0.1:<metricsPort> \
  --protocol <quic|http2|auto> \
  --edge-ip-version <auto|4|6> \
  --loglevel info --logformat json
```

- `--protocol auto` is represented by omitting `--protocol`.
- `--edge-ip-version auto` is represented by omitting the flag.
- stdout+stderr are scanned line by line; JSON lines are parsed for `level`,
  `time`, `message` and remaining fields. Non-JSON lines are kept verbatim.
- readiness: poll `http://127.0.0.1:<metricsPort>/ready` every 1 s; HTTP 200 means
  `connected`. On process start state is `connecting`.
- metrics: scrape `http://127.0.0.1:<metricsPort>/metrics` every 5 s and parse
  `cloudflared_tunnel_ha_connections`,
  `cloudflared_tunnel_total_requests` (or `cloudflared_tunnel_request_total`),
  `cloudflared_tunnel_request_errors`, and `cloudflared_tunnel_rtt` if present.
- log line `Registered tunnel connection` carries `location` (edge colo) and
  `connIndex`; use `location` for the UI edge label.
- stop: SIGTERM, wait 5 s, SIGKILL. Clear PID/state.
- unexpected exit: state `error`, then if `autoReconnect` is on, retry with backoff
  5 s, 10 s, 20 s, 40 s, 60 s (cap), up to `maxReconnect` attempts.
- boot: if `autoConnect` and `state.json` has a tunnel id, connect after the HTTP
  server is listening.

## 5. Cloudflare API usage (v4)

Base URL `https://api.cloudflare.com/client/v4`, override with `CF_API_BASE` for
tests. Header `Authorization: Bearer <token>`. Envelope
`{success, errors:[{code,message}], result, result_info}`.

| Purpose | Call |
| --- | --- |
| verify token | `GET /user/tokens/verify` |
| list tunnels | `GET /accounts/{account}/cfd_tunnel?is_deleted=false&per_page=100` |
| create tunnel | `POST /accounts/{account}/cfd_tunnel` `{name, config_src:"cloudflare"}` |
| delete tunnel | `DELETE /accounts/{account}/cfd_tunnel/{id}` |
| tunnel token | `GET /accounts/{account}/cfd_tunnel/{id}/token` |
| read ingress | `GET /accounts/{account}/cfd_tunnel/{id}/configurations` |
| write ingress | `PUT /accounts/{account}/cfd_tunnel/{id}/configurations` `{config:{ingress:[...]}}` |
| list zones | `GET /zones?per_page=50` |
| create CNAME | `POST /zones/{zone}/dns_records` `{type:"CNAME",name,content:"<tunnel>.cfargotunnel.com",proxied:true}` |
| update CNAME | `PUT /zones/{zone}/dns_records/{record}` |

- zones are cached for 5 minutes; pick the longest matching zone suffix.
- existing CNAME with the same content + proxied=true is `unchanged`; a different
  CNAME is updated; a non-CNAME record at the name is a 409 conflict.
- ingress PUT must preserve every non-`ingress` key returned by GET.
- required token permissions, surfaced in the UI help text:
  Account → Cloudflare Tunnel → Edit; Zone → DNS → Edit; Zone → Zone → Read.
- retry once on 429/5xx after 1 s. Timeout 20 s. Surface Cloudflare error messages
  verbatim inside our error envelope.

## 6. HTTP API

Prefix `/api/v1`. JSON only, UTF-8. Errors:

```json
{"error":{"code":"validation_failed","message":"hostname is invalid","details":{}}}
```

Status codes: 200, 201, 202, 204, 400, 401, 404, 409, 412, 502, 504.

### Auth

- `GET /api/v1/health` is public.
- Every other endpoint requires either:
  - the session cookie `ct_session` (set by the server when it serves `/`), or
  - `Authorization: Bearer ct_<key>` for external clients.
- The UI always uses the cookie. SSE uses the cookie. Keys are created in the UI.
- Session ids are random 32-byte hex values held in memory; restart invalidates
  browser sessions (the next page load issues a new one).

### Endpoints and DTOs

`GET /health` →
```json
{"status":"ok","version":"2026.10.7-3","demo":false,
 "startedAt":"2026-11-05T12:00:00Z","cloudflared":{"installed":true,"version":"2026.10.7"}}
```

`GET /meta` →
```json
{"version":"...","demo":false,"dataDir":"...","configDir":"...","logDir":"...",
 "listen":"0.0.0.0:18960","metricsPort":20241,"cloudflaredVersion":"..."}
```

`GET /settings` →
```json
{"accountId":"a1b2...","apiTokenSet":true,"apiTokenPreview":"••••1d8a",
 "protocol":"quic","edgeIpVersion":"auto","autoConnect":true,
 "autoReconnect":true,"maxReconnect":10,"logRetentionDays":30,"maxLogLines":2000}
```
`PUT /settings` accepts the same fields plus optional `apiToken` (empty/absent keeps
the current token). Validation: accountId 32 hex chars (or empty when unconfigured),
protocol in `auto|quic|http2`, edgeIpVersion in `auto|4|6`, maxReconnect 0–50,
logRetentionDays 1–365, maxLogLines 200–10000. Returns the new settings DTO.

`POST /settings/verify` body `{accountId?,apiToken?}` (absent fields use saved
values) →
```json
{"ok":true,"accountId":"...","tokenStatus":"active",
 "tunnelAccess":true,"zoneAccess":true,"error":""}
```

`GET /account` → `{"configured":true,"accountId":"...","accountName":"","zones":3}`
(accountName may be empty; do not fail if the token cannot read the account).

`GET /tunnels` → `{"configured":true,"tunnels":[Tunnel]}`
```json
{"id":"...","name":"fnos-main","status":"healthy","createdAt":"...",
 "connections":2,"configSrc":"cloudflare","localActive":true}
```
`POST /tunnels` `{name}` → 201 `Tunnel`; name `^[a-z0-9][a-z0-9-]{0,62}$`.
`DELETE /tunnels/{id}` → 204. If it is the locally running tunnel, stop first.
`GET /tunnels/{id}/token` → `{"token":"..."}` (session only; never logged).

`GET /tunnels/{id}/routes` →
```json
{"routes":[{"hostname":"ai.example.com","path":"/","service":"http://192.168.0.10:8080"}],
 "catchAllService":"http_status:404","rawIngressCount":3}
```
`PUT /tunnels/{id}/routes` body `{routes, catchAllService}` → same shape. The server
appends the catch-all rule (`{service: catchAllService}`) as the last ingress entry.
Validation is server-side; the UI mirrors it but never replaces it.

`POST /dns` `{tunnelId,hostname}` →
```json
{"recordId":"...","zoneId":"...","zoneName":"example.com","name":"ai.example.com",
 "content":"<tunnel-id>.cfargotunnel.com","proxied":true,"action":"created"}
```
`action` is `created|updated|unchanged`.

`GET /runtime` → Runtime
```json
{"state":"connected","tunnelId":"...","tunnelName":"fnos-main",
 "connectedAt":"...","uptimeSec":3421,"protocol":"quic",
 "edgeLocation":"HKG","connections":2,"reconnectAttempt":0,
 "lastError":"","cloudflaredVersion":"2026.10.7"}
```
`state` ∈ `disconnected|connecting|connected|reconnecting|error|stopping`.
`POST /runtime/connect` `{tunnelId}` → 202 `{"status":Runtime}`.
`POST /runtime/disconnect` → 200 `{"status":Runtime}`.
`GET /runtime/metrics` →
```json
{"requests":12400,"errors":3,"connections":2,"rttMs":38,"sampledAt":"..."}
```
Missing metric values are `null`, never invented.

`GET /logs?limit=500&afterSeq=0&level=all&q=` →
```json
{"lines":[{"seq":42,"ts":"...","level":"info","message":"...","fields":{}}],
 "nextSeq":42,"dropped":0}
```
`level` ∈ `all|debug|info|warn|error`; `q` is a case-insensitive substring filter.
`POST /logs/clear` → 204. `GET /logs/download` → `text/plain` attachment.
`GET /events` → SSE. Events:
- `hello` `{status:Runtime, recent:[LogLine,...]}` (last 50)
- `status` Runtime
- `log` LogLine
- `activity` Activity
- `metrics` RuntimeMetrics
- comment ping every 15 s.

`GET /activity` → `{"items":[{"id":"...","ts":"...","kind":"connect","title":"...","tone":"green"}]}`
(kind/tone are presentation hints; keep the same vocabulary as the prototype:
green/blue/teal/neutral/danger).

`GET /keys` → `{"keys":[{"id":"...","name":"Home Assistant","prefix":"ct_ab12",
"createdAt":"...","lastUsedAt":""}]}`
`POST /keys` `{name}` → 201 `{"key":Key,"token":"ct_<64 hex>"}` (token shown once).
`DELETE /keys/{id}` → 204.

`GET /overview` → aggregate for the overview page:
```json
{"runtime":Runtime,"tunnelCount":3,"healthyCount":1,"routeCount":8,
 "zoneCount":2,"metrics":{"requests":12400,"errors":3,"connections":2,"rttMs":38},
 "cloudflaredVersion":"2026.10.7",
 "activity":[Activity]}
```

## 7. Demo mode

`-demo` (or `CT_DEMO=1`) must provide a fully working UI without network or
cloudflared: seeded settings (fake account/token), three tunnels (`fnos-main`
healthy, `home-lab` inactive, `edge-backup` error), routes, simulated connect
transitions, periodic log lines, metrics, activity, API keys. Demo data lives in a
temp directory and never writes to the real share. The UI shows a `演示模式` badge
when `/health.demo` is true.

## 8. Frontend contract

- Vue 3 + TypeScript + Vite + vue-router (hash history). No component library.
- The prototype in `prototype/` is the visual source of truth: copy its CSS
  variables, classes and layout; port `app.js` templates into components.
- Pages: `overview`, `tunnels`, `routes`, `logs`, `api`, `settings`.
- Theme: default follows `prefers-color-scheme`, manual toggle persisted in
  `localStorage`, `data-theme="light|dark"` on `<html>`. fnOS light is the primary
  look. Reduced-motion preference persisted (`localStorage`).
- Live updates: one `EventSource('/api/v1/events')`; reconnect with backoff; when
  disconnected show a subtle banner and retry.
- Loading/empty/error states for every page. No fake data in production; demo badge
  only when the backend says so.
- All mutations: disable the trigger while in flight, show toast on success, keep
  the dialog open with an inline error on failure.
- Never render the API token; show `apiTokenPreview`.
- Accessibility: focus trap + Escape close for modal/drawer, visible focus rings,
  `aria-live` toasts, 44 px minimum touch targets on mobile.
- Responsive: sidebar becomes an overlay ≤920 px; tables become cards or scroll;
  test at 620, 900, 1200, 1440 px.
- Fonts bundled via `@fontsource/manrope` + `@fontsource/ibm-plex-mono` (offline NAS).
- Dev proxy `/api` → `http://127.0.0.1:18960`.

## 9. Build and packaging

Use any Go 1.26+ toolchain on `PATH` (or pass `GO=/path/to/go`).
Cross-compile for fnOS: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`.

```
web:    pnpm install && pnpm build            # outputs web/dist
server: copy web/dist -> server/internal/assets/dist; go build ./cmd/cloudflare-tunnel
fpk:    app/cloudflare-tunnel-server, app/bin/cloudflared, app/ui/config,
        app/ui/images/icon-64.png, icon-256.png, LICENSE, THIRD_PARTY_NOTICES
```

`cmd/main` runs the server from `$TRIM_APPDEST`, data from the share. Nine lifecycle
scripts are mandatory. `manifest service_port = 18960`. `config/privilege` is
`run-as: package`. `config/resource` declares data-share `cloudflare-tunnel`.
`ui/config` is an iframe entry with `protocol: ""` and `port: 18960`.

## 10. Acceptance per stream

Backend:

- `go vet ./...` and `go test ./...` pass.
- `go build ./cmd/cloudflare-tunnel` succeeds; `-demo` serves `/` + API on 18960.
- Fake Cloudflare API tests cover: token verify, tunnel list/create/delete, ingress
  read/write preserve keys, DNS create/update/conflict, error envelope mapping.
- Supervisor tests cover: start → ready → connected, unexpected exit → reconnect,
  stop → disconnected, token passed via env not argv.
- No secret appears in logs/JSON responses (grep the test output).

Frontend:

- `pnpm build` and `vue-tsc --noEmit` pass.
- All six pages render at 620/900/1440, light+dark, with mock data.
- Every prototype interaction exists: create tunnel, connect dialog, add/edit/delete
  route, drag reorder, YAML copy, logs pause/filter/detail/download, settings dirty
  save+verify, key create/revoke, theme, mobile nav, toasts, focus trap.

Packaging:

- `fnpack build` succeeds; `tar tzf` contains `cmd/main`, `ui/config`, `app.tgz`;
  `app.tgz` contains the server binary, `bin/cloudflared`, `ui/config`, icons.
- 9 scripts present and executable; no `.bak`/`.orig`/`~` files.
- `cmd/main` contains `resolve_share_root` and never writes persistent data to
  `@appcenter`/`@appdata`.
- cloudflared is Apache-2.0; `LICENSE` + `THIRD_PARTY_NOTICES` included.

## 11. Rules for every agent

- Do not run `git commit`, `git push`, install packages on the NAS, or touch
  the `nas-client` workspace.
- Do not put credentials, tokens, or machine-specific paths in source files.
- LF line endings only.
- Read `fndepot/app-template/AGENT.md` before touching `fpk/`.
- Report exact commands run and their results; do not claim a test passed without
  the command output.
