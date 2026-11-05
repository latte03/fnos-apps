package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

const testAPIToken = "cf-test-token-abcd1234"

type fakeCloud struct {
	tunnels []model.Tunnel
	configs map[string]map[string]any
	zones   []model.Zone
}

func newFakeCloud() *fakeCloud {
	return &fakeCloud{
		tunnels: []model.Tunnel{
			{ID: "tun-1", Name: "fnos-main", Status: model.TunnelHealthy, CreatedAt: "2026-09-18T09:12:40Z", Connections: 2, ConfigSrc: "cloudflare"},
			{ID: "tun-2", Name: "home-lab", Status: model.TunnelInactive, CreatedAt: "2026-10-02T13:41:06Z", ConfigSrc: "cloudflare"},
		},
		configs: map[string]map[string]any{
			"tun-1": {
				"warp-routing": map[string]any{"enabled": true},
				"ingress": []any{
					map[string]any{"hostname": "ai.example.com", "path": "/", "service": "http://127.0.0.1:3000"},
					map[string]any{"service": "http_status:404"},
				},
			},
		},
		zones: []model.Zone{{ID: "zone-1", Name: "example.com"}},
	}
}

func (f *fakeCloud) VerifyToken(_ context.Context, accountID, token string) (model.VerifyResult, error) {
	return model.VerifyResult{OK: true, AccountID: accountID, TokenStatus: "active", TunnelAccess: true, ZoneAccess: true}, nil
}

func (f *fakeCloud) AccountName(_ context.Context, _, _ string) string { return "Test Account" }

func (f *fakeCloud) ListTunnels(_ context.Context, _, _ string) ([]model.Tunnel, error) {
	return append([]model.Tunnel(nil), f.tunnels...), nil
}

func (f *fakeCloud) CreateTunnel(_ context.Context, _, _, name string) (model.Tunnel, error) {
	tunnel := model.Tunnel{ID: "tun-new", Name: name, Status: model.TunnelInactive, ConfigSrc: "cloudflare"}
	f.tunnels = append(f.tunnels, tunnel)
	return tunnel, nil
}

func (f *fakeCloud) DeleteTunnel(_ context.Context, _, _, tunnelID string) error {
	for i, tunnel := range f.tunnels {
		if tunnel.ID == tunnelID {
			f.tunnels = append(f.tunnels[:i], f.tunnels[i+1:]...)
			return nil
		}
	}
	return model.NewError(http.StatusNotFound, "not_found", "tunnel not found")
}

func (f *fakeCloud) TunnelToken(_ context.Context, _, _, _ string) (string, error) {
	return "connector-token-value", nil
}

func (f *fakeCloud) GetConfig(_ context.Context, _, _, tunnelID string) (map[string]any, error) {
	cfg, ok := f.configs[tunnelID]
	if !ok {
		return map[string]any{}, nil
	}
	return cfg, nil
}

func (f *fakeCloud) PutConfig(_ context.Context, _, _, tunnelID string, cfg map[string]any) error {
	f.configs[tunnelID] = cfg
	return nil
}

func (f *fakeCloud) ListZones(_ context.Context, _ string) ([]model.Zone, error) {
	return f.zones, nil
}

func (f *fakeCloud) UpsertCNAME(_ context.Context, _, tunnelID, hostname string) (model.DNSResult, error) {
	return model.DNSResult{
		RecordID: "rec-1", ZoneID: "zone-1", ZoneName: "example.com",
		Name: hostname, Content: tunnelID + ".cfargotunnel.com", Proxied: true, Action: "created",
	}, nil
}

type fakeEngine struct {
	state   model.Runtime
	metrics model.RuntimeMetrics
	logs    []model.LogLine
	cleared bool
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{
		state: model.Runtime{State: model.StateDisconnected},
		logs: []model.LogLine{
			{Seq: 1, TS: "2026-11-05T12:00:00.000Z", Level: "info", Message: "Registered tunnel connection", Fields: map[string]any{"location": "HKG"}},
			{Seq: 2, TS: "2026-11-05T12:00:01.000Z", Level: "warn", Message: "Connection terminated", Fields: map[string]any{}},
		},
	}
}

func (e *fakeEngine) State() model.Runtime { return e.state }

func (e *fakeEngine) Metrics() model.RuntimeMetrics { return e.metrics }

func (e *fakeEngine) Connect(tunnelID, tunnelName, _ string) error {
	e.state = model.Runtime{State: model.StateConnecting, TunnelID: tunnelID, TunnelName: tunnelName}
	return nil
}

func (e *fakeEngine) Disconnect() error {
	e.state = model.Runtime{State: model.StateDisconnected}
	return nil
}

func (e *fakeEngine) Logs(q runtime.Query) model.LogPage {
	return model.LogPage{Lines: e.logs, NextSeq: 2}
}

func (e *fakeEngine) ClearLogs() { e.logs = nil; e.cleared = true }

func (e *fakeEngine) Text() string { return "log line\n" }

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	root := t.TempDir()
	st, err := store.Open(store.Paths{
		ConfigDir: filepath.Join(root, "config"),
		DataDir:   filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	settings := store.DefaultSettings()
	settings.AccountID = "0123456789abcdef0123456789abcdef"
	settings.APIToken = testAPIToken
	if err := st.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	srv := New(Options{
		Version: "test",
		Store:   st,
		Cloud:   newFakeCloud(),
		Engine:  newFakeEngine(),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st
}

func decodeEnvelope(t *testing.T, body io.Reader) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return env
}

func TestAuthRequiredWithoutCredentials(t *testing.T) {
	ts, _ := newTestServer(t)
	res, err := http.Get(ts.URL + "/api/v1/settings")
	if err != nil {
		t.Fatalf("GET settings: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
	env := decodeEnvelope(t, res.Body)
	if env.Error.Code != "unauthorized" || env.Error.Message == "" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
	if env.Error.Details == nil {
		t.Fatal("details must be an object")
	}
}

func TestHealthIsPublic(t *testing.T) {
	ts, _ := newTestServer(t)
	res, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var health healthResponse
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Status != "ok" || health.Version != "test" || health.Demo {
		t.Fatalf("unexpected health: %+v", health)
	}
}

func TestSessionCookieFromRoot(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "Cloudflare Tunnel") {
		t.Fatalf("unexpected index response: %d", res.StatusCode)
	}
	cookies := res.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == sessionCookie && len(cookie.Value) == 64 {
			found = true
		}
	}
	if !found {
		t.Fatalf("ct_session cookie missing: %v", cookies)
	}

	res, err = client.Get(ts.URL + "/api/v1/settings")
	if err != nil {
		t.Fatalf("GET settings: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	raw, _ := io.ReadAll(res.Body)
	if strings.Contains(string(raw), testAPIToken) {
		t.Fatal("settings response leaked the API token")
	}
	var view settingsView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if !view.APITokenSet || view.APITokenPreview != "••••1234" {
		t.Fatalf("unexpected mask: %+v", view)
	}
}

func TestBearerKeyAuth(t *testing.T) {
	ts, st := newTestServer(t)
	_, token, err := st.AddKey("n8n")
	if err != nil {
		t.Fatalf("AddKey: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/tunnels", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET tunnels: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var payload tunnelsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Configured || len(payload.Tunnels) != 2 {
		t.Fatalf("unexpected tunnels: %+v", payload)
	}

	// The connector token endpoint is session only.
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/tunnels/tun-1/token", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET token: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func TestUnknownAPIPathReturnsEnvelope(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	res, err = client.Get(ts.URL + "/api/v1/nope")
	if err != nil {
		t.Fatalf("GET nope: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	env := decodeEnvelope(t, res.Body)
	if env.Error.Code != "not_found" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestSPAFallback(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/some/deep/route")
	if err != nil {
		t.Fatalf("GET deep route: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content type = %q", ct)
	}
}

func TestSSEEmitsHello(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/v1/events", nil)
	stream, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer stream.Body.Close()
	if stream.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", stream.StatusCode)
	}
	if ct := stream.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type = %q", ct)
	}

	reader := bufio.NewReader(stream.Body)
	eventLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read event line: %v", err)
	}
	if strings.TrimSpace(eventLine) != "event: hello" {
		t.Fatalf("first event = %q", eventLine)
	}
	dataLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read data line: %v", err)
	}
	if !strings.HasPrefix(dataLine, "data: ") {
		t.Fatalf("unexpected data line %q", dataLine)
	}
	var payload helloPayload
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(dataLine), "data: ")), &payload); err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	if payload.Status.State != model.StateDisconnected {
		t.Fatalf("unexpected status: %+v", payload.Status)
	}
	if len(payload.Recent) != 2 {
		t.Fatalf("recent = %d, want 2", len(payload.Recent))
	}
	blank, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read blank line: %v", err)
	}
	if strings.TrimSpace(blank) != "" {
		t.Fatalf("expected a blank separator, got %q", blank)
	}
}

func TestRuntimeConnectAndDisconnect(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	res, err = client.Post(ts.URL+"/api/v1/runtime/connect", "application/json", strings.NewReader(`{"tunnelId":"tun-1"}`))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("connect status = %d, want 202", res.StatusCode)
	}
	var status runtimeStatusResponse
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.Status.TunnelID != "tun-1" || status.Status.TunnelName != "fnos-main" {
		t.Fatalf("unexpected runtime: %+v", status.Status)
	}

	res, err = client.Post(ts.URL+"/api/v1/runtime/disconnect", "application/json", nil)
	if err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("disconnect status = %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.Status.State != model.StateDisconnected {
		t.Fatalf("unexpected state: %+v", status.Status)
	}
}

func TestRoutesPutPreservesWarpRouting(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	body := `{"routes":[{"hostname":"nas.example.com","path":"/","service":"http://127.0.0.1:5666"}],"catchAllService":"http_status:410"}`
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/tunnels/tun-1/routes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("PUT routes: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d body=%s", res.StatusCode, raw)
	}
	var routes model.Routes
	if err := json.NewDecoder(res.Body).Decode(&routes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(routes.Routes) != 1 || routes.CatchAllService != "http_status:410" || routes.RawIngressCount != 2 {
		t.Fatalf("unexpected routes: %+v", routes)
	}
}

func TestRoutesPutValidationError(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	body := `{"routes":[{"hostname":"not a host","path":"/","service":"http://127.0.0.1:5666"}],"catchAllService":"http_status:404"}`
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/tunnels/tun-1/routes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("PUT routes: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	env := decodeEnvelope(t, res.Body)
	if env.Error.Code != "validation_failed" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestSettingsPutNeverReturnsToken(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = res.Body.Close()

	body := `{"apiToken":"cf-brand-new-token-9876","protocol":"http2","maxReconnect":7}`
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("PUT settings: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	raw, _ := io.ReadAll(res.Body)
	if strings.Contains(string(raw), "cf-brand-new-token-9876") {
		t.Fatal("PUT settings echoed the token")
	}
	var view settingsView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Protocol != "http2" || view.MaxReconnect != 7 || view.APITokenPreview != "••••9876" {
		t.Fatalf("unexpected settings: %+v", view)
	}

	// An empty token keeps the current one.
	body = `{"apiToken":""}`
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("PUT settings: %v", err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.APITokenPreview != "••••9876" || !view.APITokenSet {
		t.Fatalf("empty token wiped the saved one: %+v", view)
	}
}
