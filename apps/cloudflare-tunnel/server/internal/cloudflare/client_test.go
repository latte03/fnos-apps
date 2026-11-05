package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

const (
	testAccount = "0123456789abcdef0123456789abcdef"
	testToken   = "super-secret-token-value"
)

// newTestClient wires a client against an httptest server.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		RetryDelay: 5 * time.Millisecond,
		ZoneTTL:    time.Minute,
	})
}

// reply writes a successful envelope.
func reply(w http.ResponseWriter, status int, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": status < 400,
		"errors":  []any{},
		"result":  result,
	})
}

// replyError writes a Cloudflare style error envelope.
func replyError(w http.ResponseWriter, status, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"errors":  []map[string]any{{"code": code, "message": message}},
		"result":  nil,
	})
}

func TestVerifyToken(t *testing.T) {
	var mu sync.Mutex
	seen := []string{}
	mux := http.NewServeMux()
	mux.HandleFunc("/user/tokens/verify", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path)
		mu.Unlock()
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("missing bearer header, got %q", got)
		}
		reply(w, http.StatusOK, map[string]any{"id": "tok", "status": "active"})
	})
	mux.HandleFunc("/accounts/"+testAccount+"/cfd_tunnel", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+"?"+r.URL.RawQuery)
		mu.Unlock()
		reply(w, http.StatusOK, []any{})
	})
	mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+"?"+r.URL.RawQuery)
		mu.Unlock()
		reply(w, http.StatusOK, []any{})
	})

	client := newTestClient(t, mux)
	res, err := client.VerifyToken(context.Background(), testAccount, testToken)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if !res.OK || res.TokenStatus != "active" || !res.TunnelAccess || !res.ZoneAccess {
		t.Fatalf("unexpected verify result: %+v", res)
	}
	if res.Error != "" {
		t.Fatalf("expected no error text, got %q", res.Error)
	}
	joined := strings.Join(seen, " ")
	for _, want := range []string{"/user/tokens/verify", "per_page=1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected request %q in %v", want, seen)
		}
	}
	if strings.Contains(res.Error, testToken) {
		t.Fatal("token leaked into the verify result")
	}
}

// Account-owned API tokens are valid but return 401 from the user-level verify
// endpoint. Verification must succeed through the account-scoped endpoint.
func TestVerifyTokenAccountScoped(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/"+testAccount+"/tokens/verify", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, map[string]any{"id": "tok", "status": "active"})
	})
	mux.HandleFunc("/user/tokens/verify", func(w http.ResponseWriter, r *http.Request) {
		t.Error("user-level verify should not be called when the account-scoped check succeeds")
		replyError(w, http.StatusUnauthorized, 1000, "Invalid API Token")
	})
	mux.HandleFunc("/accounts/"+testAccount+"/cfd_tunnel", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, []any{})
	})
	mux.HandleFunc("/zones", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, []any{})
	})

	client := newTestClient(t, mux)
	res, err := client.VerifyToken(context.Background(), testAccount, testToken)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if !res.OK || res.TokenStatus != "active" || !res.TunnelAccess || !res.ZoneAccess {
		t.Fatalf("unexpected verify result: %+v", res)
	}
	if res.Error != "" {
		t.Fatalf("expected no error text, got %q", res.Error)
	}
}

func TestVerifyTokenMissingToken(t *testing.T) {
	client := newTestClient(t, http.NewServeMux())
	res, err := client.VerifyToken(context.Background(), testAccount, "")
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if res.OK || res.Error == "" {
		t.Fatalf("expected a failing result, got %+v", res)
	}
}

func TestListCreateDeleteTunnel(t *testing.T) {
	var (
		mu      sync.Mutex
		created string
		deleted string
	)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testAccount+"/cfd_tunnel", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("is_deleted"); got != "false" {
			t.Errorf("is_deleted query = %q, want false", got)
		}
		reply(w, http.StatusOK, []map[string]any{
			{
				"id": "tun-1", "name": "fnos-main", "status": "healthy",
				"created_at": "2026-09-18T09:12:40Z", "remote_config": true,
				"connections": []map[string]any{{"colo_name": "HKG"}, {"colo_name": "HKG"}},
			},
			{
				"id": "tun-2", "name": "edge-backup", "status": "down",
				"created_at": "2026-10-21T05:03:55Z", "remote_config": false,
			},
		})
	})
	mux.HandleFunc("POST /accounts/"+testAccount+"/cfd_tunnel", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		created, _ = body["name"].(string)
		mu.Unlock()
		if body["config_src"] != "cloudflare" {
			t.Errorf("config_src = %v, want cloudflare", body["config_src"])
		}
		reply(w, http.StatusOK, map[string]any{"id": "tun-3", "name": body["name"], "remote_config": true})
	})
	mux.HandleFunc("DELETE /accounts/"+testAccount+"/cfd_tunnel/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = r.PathValue("id")
		mu.Unlock()
		reply(w, http.StatusOK, nil)
	})

	client := newTestClient(t, mux)
	tunnels, err := client.ListTunnels(context.Background(), testAccount, testToken)
	if err != nil {
		t.Fatalf("ListTunnels: %v", err)
	}
	if len(tunnels) != 2 {
		t.Fatalf("got %d tunnels, want 2", len(tunnels))
	}
	if tunnels[0].Status != model.TunnelHealthy || tunnels[0].Connections != 2 || tunnels[0].ConfigSrc != "cloudflare" {
		t.Fatalf("unexpected first tunnel: %+v", tunnels[0])
	}
	if tunnels[1].Status != model.TunnelError || tunnels[1].ConfigSrc != "local" {
		t.Fatalf("unexpected second tunnel: %+v", tunnels[1])
	}

	createdTunnel, err := client.CreateTunnel(context.Background(), testAccount, testToken, "new-one")
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if createdTunnel.ID != "tun-3" || createdTunnel.ConfigSrc != "cloudflare" || createdTunnel.Status != model.TunnelInactive {
		t.Fatalf("unexpected created tunnel: %+v", createdTunnel)
	}
	mu.Lock()
	if created != "new-one" {
		t.Errorf("created name = %q", created)
	}
	mu.Unlock()

	if err := client.DeleteTunnel(context.Background(), testAccount, testToken, "tun-3"); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if deleted != "tun-3" {
		t.Fatalf("deleted = %q, want tun-3", deleted)
	}
}

func TestTunnelToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/"+testAccount+"/cfd_tunnel/tun-1/token", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, "eyJhIjoiYWNjb3VudCIsInQiOiJ0dW5uZWwifQ==")
	})
	client := newTestClient(t, mux)
	token, err := client.TunnelToken(context.Background(), testAccount, testToken, "tun-1")
	if err != nil {
		t.Fatalf("TunnelToken: %v", err)
	}
	if !strings.HasPrefix(token, "eyJ") {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestIngressRoundTripPreservesKeys(t *testing.T) {
	var putBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testAccount+"/cfd_tunnel/tun-1/configurations", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, map[string]any{
			"config": map[string]any{
				"warp-routing": map[string]any{"enabled": true},
				"originRequest": map[string]any{
					"connectTimeout": "30s",
				},
				"ingress": []any{
					map[string]any{"hostname": "ai.example.com", "path": "/", "service": "http://127.0.0.1:3000"},
					map[string]any{"service": "http_status:404"},
				},
			},
		})
	})
	mux.HandleFunc("PUT /accounts/"+testAccount+"/cfd_tunnel/tun-1/configurations", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&putBody)
		reply(w, http.StatusOK, map[string]any{"config": putBody["config"]})
	})

	client := newTestClient(t, mux)
	cfg, err := client.GetConfig(context.Background(), testAccount, testToken, "tun-1")
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	routes, catchAll, raw := model.IngressRoutes(cfg)
	if len(routes) != 1 || routes[0].Hostname != "ai.example.com" || catchAll != "http_status:404" || raw != 2 {
		t.Fatalf("unexpected ingress parse: %+v %q %d", routes, catchAll, raw)
	}

	merged := model.WithIngress(cfg, []model.Route{
		{Hostname: "nas.example.com", Path: "/", Service: "http://127.0.0.1:5666"},
		{Hostname: "ai.example.com", Path: "/", Service: "http://127.0.0.1:3001"},
	}, "http_status:410")
	if err := client.PutConfig(context.Background(), testAccount, testToken, "tun-1", merged); err != nil {
		t.Fatalf("PutConfig: %v", err)
	}
	config, ok := putBody["config"].(map[string]any)
	if !ok {
		t.Fatalf("missing config in PUT body: %#v", putBody)
	}
	if _, ok := config["warp-routing"]; !ok {
		t.Fatal("warp-routing was dropped by the PUT")
	}
	origin, ok := config["originRequest"].(map[string]any)
	if !ok || origin["connectTimeout"] != "30s" {
		t.Fatalf("originRequest was dropped: %#v", config["originRequest"])
	}
	ingress, ok := config["ingress"].([]any)
	if !ok || len(ingress) != 3 {
		t.Fatalf("unexpected ingress: %#v", config["ingress"])
	}
	last, _ := ingress[2].(map[string]any)
	if last["service"] != "http_status:410" || last["hostname"] != nil {
		t.Fatalf("catch-all not appended last: %#v", last)
	}
}

func TestUpsertCNAME(t *testing.T) {
	cases := []struct {
		name       string
		host       string
		records    []map[string]any
		wantAction string
		wantMethod string
		wantStatus int
	}{
		{
			name:       "create",
			host:       "nas.sub.example.com",
			records:    nil,
			wantAction: "created",
			wantMethod: http.MethodPost,
		},
		{
			name:       "unchanged",
			host:       "nas.sub.example.com",
			records:    []map[string]any{{"id": "rec-1", "type": "CNAME", "name": "nas.sub.example.com", "content": "tun-1.cfargotunnel.com", "proxied": true}},
			wantAction: "unchanged",
		},
		{
			name:       "update",
			host:       "nas.sub.example.com",
			records:    []map[string]any{{"id": "rec-1", "type": "CNAME", "name": "nas.sub.example.com", "content": "old.cfargotunnel.com", "proxied": true}},
			wantAction: "updated",
			wantMethod: http.MethodPut,
		},
		{
			name:       "conflict",
			host:       "nas.sub.example.com",
			records:    []map[string]any{{"id": "rec-1", "type": "A", "name": "nas.sub.example.com", "content": "192.0.2.10"}},
			wantStatus: http.StatusConflict,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu       sync.Mutex
				lastVerb string
			)
			mux := http.NewServeMux()
			mux.HandleFunc("GET /zones", func(w http.ResponseWriter, r *http.Request) {
				reply(w, http.StatusOK, []map[string]any{
					{"id": "zone-root", "name": "example.com"},
					{"id": "zone-sub", "name": "sub.example.com"},
				})
			})
			mux.HandleFunc("GET /zones/{zone}/dns_records", func(w http.ResponseWriter, r *http.Request) {
				if r.PathValue("zone") != "zone-sub" {
					t.Errorf("expected the longest zone suffix, got %q", r.PathValue("zone"))
				}
				reply(w, http.StatusOK, tc.records)
			})
			mux.HandleFunc("POST /zones/{zone}/dns_records", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				lastVerb = http.MethodPost
				mu.Unlock()
				reply(w, http.StatusOK, map[string]any{"id": "rec-new", "type": "CNAME", "name": tc.host, "content": "tun-1.cfargotunnel.com", "proxied": true})
			})
			mux.HandleFunc("PUT /zones/{zone}/dns_records/{id}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				lastVerb = http.MethodPut
				mu.Unlock()
				reply(w, http.StatusOK, map[string]any{"id": r.PathValue("id")})
			})

			client := newTestClient(t, mux)
			res, err := client.UpsertCNAME(context.Background(), testToken, "tun-1", tc.host)
			if tc.wantStatus != 0 {
				var apiErr *model.APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected APIError, got %v", err)
				}
				if apiErr.Status != tc.wantStatus || apiErr.Code != "conflict" {
					t.Fatalf("unexpected error: %+v", apiErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpsertCNAME: %v", err)
			}
			if res.Action != tc.wantAction {
				t.Fatalf("action = %q, want %q", res.Action, tc.wantAction)
			}
			if res.Content != "tun-1.cfargotunnel.com" || !res.Proxied || res.ZoneName != "sub.example.com" {
				t.Fatalf("unexpected result: %+v", res)
			}
			mu.Lock()
			defer mu.Unlock()
			if lastVerb != tc.wantMethod {
				t.Fatalf("verb = %q, want %q", lastVerb, tc.wantMethod)
			}
		})
	}
}

func TestRetryOnceOn429(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/user/tokens/verify", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		current := calls
		mu.Unlock()
		if current == 1 {
			replyError(w, http.StatusTooManyRequests, 10000, "rate limited")
			return
		}
		reply(w, http.StatusOK, map[string]any{"status": "active"})
	})
	client := newTestClient(t, mux)
	raw, err := client.do(context.Background(), testToken, http.MethodGet, "/user/tokens/verify", nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if !strings.Contains(string(raw), "active") {
		t.Fatalf("unexpected result %s", raw)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestRetryGivesUp(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/x", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		replyError(w, http.StatusBadGateway, 1000, "bad gateway")
	})
	client := newTestClient(t, mux)
	_, err := client.do(context.Background(), testToken, http.MethodGet, "/accounts/x", nil)
	var apiErr *model.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Status != http.StatusBadGateway || apiErr.Message != "bad gateway" {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestErrorEnvelopeMapping(t *testing.T) {
	cases := []struct {
		upstream int
		status   int
		code     string
		message  string
	}{
		{http.StatusBadRequest, http.StatusBadRequest, "validation_failed", "Invalid tunnel name"},
		{http.StatusUnauthorized, http.StatusBadRequest, "cloudflare_auth", "Invalid API Token"},
		{http.StatusForbidden, http.StatusBadRequest, "cloudflare_forbidden", "Authentication error"},
		{http.StatusNotFound, http.StatusNotFound, "not_found", "Tunnel not found"},
		{http.StatusConflict, http.StatusConflict, "conflict", "Duplicate record"},
	}
	for _, tc := range cases {
		mux := http.NewServeMux()
		mux.HandleFunc("/accounts/x", func(w http.ResponseWriter, r *http.Request) {
			replyError(w, tc.upstream, 9999, tc.message)
		})
		client := newTestClient(t, mux)
		_, err := client.do(context.Background(), testToken, http.MethodGet, "/accounts/x", nil)
		var apiErr *model.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError for %d, got %v", tc.upstream, err)
		}
		if apiErr.Status != tc.status || apiErr.Code != tc.code || apiErr.Message != tc.message {
			t.Fatalf("upstream %d mapped to %+v", tc.upstream, apiErr)
		}
		if strings.Contains(apiErr.Error(), testToken) {
			t.Fatal("token leaked into the error")
		}
	}
}

func TestTimeoutMapping(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/x", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		reply(w, http.StatusOK, map[string]any{})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := New(Options{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 25 * time.Millisecond},
		RetryDelay: time.Millisecond,
	})
	_, err := client.do(context.Background(), testToken, http.MethodGet, "/accounts/x", nil)
	var apiErr *model.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Status != http.StatusGatewayTimeout || apiErr.Code != "upstream_timeout" {
		t.Fatalf("unexpected timeout mapping: %+v", apiErr)
	}
}

func TestMatchZone(t *testing.T) {
	zones := []model.Zone{{ID: "1", Name: "example.com"}, {ID: "2", Name: "sub.example.com"}}
	zone, ok := MatchZone(zones, "nas.sub.example.com")
	if !ok || zone.ID != "2" {
		t.Fatalf("longest suffix not picked: %+v", zone)
	}
	if _, ok := MatchZone(zones, "other.test"); ok {
		t.Fatal("unexpected zone match")
	}
}

func TestErrorBodyIsBounded(t *testing.T) {
	long := strings.Repeat("x", 5000)
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/x", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, long)
	})
	client := newTestClient(t, mux)
	_, err := client.do(context.Background(), testToken, http.MethodGet, "/accounts/x", nil)
	var apiErr *model.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if len(apiErr.Message) > 260 {
		t.Fatalf("message not truncated: %d", len(apiErr.Message))
	}
}
