package httpapi

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

// CloudAPI is the Cloudflare surface the HTTP layer needs. The token is passed
// per call so settings changes take effect immediately.
type CloudAPI interface {
	VerifyToken(ctx context.Context, accountID, token string) (model.VerifyResult, error)
	AccountName(ctx context.Context, accountID, token string) string
	ListTunnels(ctx context.Context, accountID, token string) ([]model.Tunnel, error)
	CreateTunnel(ctx context.Context, accountID, token, name string) (model.Tunnel, error)
	DeleteTunnel(ctx context.Context, accountID, token, tunnelID string) error
	TunnelToken(ctx context.Context, accountID, token, tunnelID string) (string, error)
	GetConfig(ctx context.Context, accountID, token, tunnelID string) (map[string]any, error)
	PutConfig(ctx context.Context, accountID, token, tunnelID string, cfg map[string]any) error
	ListZones(ctx context.Context, token string) ([]model.Zone, error)
	UpsertCNAME(ctx context.Context, token, tunnelID, hostname string) (model.DNSResult, error)
}

// Engine is the supervisor surface used by the handlers.
type Engine interface {
	State() model.Runtime
	Metrics() model.RuntimeMetrics
	Connect(tunnelID, tunnelName, token string) error
	Disconnect() error
	Logs(runtime.Query) model.LogPage
	ClearLogs()
	Text() string
}

// Options configures the HTTP layer.
type Options struct {
	Version            string
	Demo               bool
	Listen             string
	MetricsPort        int
	ConfigDir          string
	DataDir            string
	LogDir             string
	CloudflaredPath    string
	CloudflaredVersion string
	StartedAt          time.Time

	Store  *store.Store
	Cloud  CloudAPI
	Engine Engine
}

// Server owns the router, the SSE hub and the in-memory sessions.
type Server struct {
	opts      Options
	store     *store.Store
	cloud     CloudAPI
	engine    Engine
	hub       *Hub
	sessions  *sessionStore
	startedAt time.Time

	mu            sync.Mutex
	routeCount    int
	routeCountAt  time.Time
	routeCountKey string
}

// New builds the HTTP layer. Engine may be attached later with SetEngine.
func New(opts Options) *Server {
	if opts.Version == "" {
		opts.Version = "2026.10.7-5"
	}
	if opts.StartedAt.IsZero() {
		opts.StartedAt = time.Now()
	}
	return &Server{
		opts:      opts,
		store:     opts.Store,
		cloud:     opts.Cloud,
		engine:    opts.Engine,
		hub:       NewHub(),
		sessions:  newSessionStore(),
		startedAt: opts.StartedAt,
	}
}

// SetEngine attaches the supervisor (it is created after the server so it can
// publish through the hub).
func (s *Server) SetEngine(e Engine) { s.engine = e }

// Hub exposes the event hub for tests and wiring.
func (s *Server) Hub() *Hub { return s.hub }

type route struct {
	pattern string
	handler http.HandlerFunc
}

// Handler builds the mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	routes := []route{
		{"GET /api/v1/meta", s.handleMeta},
		{"GET /api/v1/settings", s.handleSettingsGet},
		{"PUT /api/v1/settings", s.handleSettingsPut},
		{"POST /api/v1/settings/verify", s.handleSettingsVerify},
		{"GET /api/v1/account", s.handleAccount},
		{"GET /api/v1/tunnels", s.handleTunnelsList},
		{"POST /api/v1/tunnels", s.handleTunnelsCreate},
		{"DELETE /api/v1/tunnels/{id}", s.handleTunnelsDelete},
		{"GET /api/v1/tunnels/{id}/token", s.handleTunnelToken},
		{"GET /api/v1/tunnels/{id}/routes", s.handleRoutesGet},
		{"PUT /api/v1/tunnels/{id}/routes", s.handleRoutesPut},
		{"POST /api/v1/dns", s.handleDNS},
		{"GET /api/v1/runtime", s.handleRuntimeGet},
		{"POST /api/v1/runtime/connect", s.handleRuntimeConnect},
		{"POST /api/v1/runtime/disconnect", s.handleRuntimeDisconnect},
		{"GET /api/v1/runtime/metrics", s.handleRuntimeMetrics},
		{"GET /api/v1/logs", s.handleLogs},
		{"POST /api/v1/logs/clear", s.handleLogsClear},
		{"GET /api/v1/logs/download", s.handleLogsDownload},
		{"GET /api/v1/events", s.handleEvents},
		{"GET /api/v1/activity", s.handleActivity},
		{"GET /api/v1/keys", s.handleKeysList},
		{"POST /api/v1/keys", s.handleKeysCreate},
		{"DELETE /api/v1/keys/{id}", s.handleKeysDelete},
		{"GET /api/v1/overview", s.handleOverview},
	}
	for _, r := range routes {
		mux.HandleFunc(r.pattern, s.auth(r.handler))
	}

	// Unknown API paths must still answer with the JSON error envelope.
	mux.HandleFunc("/api/v1/", s.handleAPINotFound)
	mux.HandleFunc("/api/", s.handleAPINotFound)
	mux.HandleFunc("/", s.handleStatic)
	return mux
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, model.NewError(http.StatusNotFound, "not_found", "unknown endpoint "+r.Method+" "+r.URL.Path))
}

// settings is a shorthand for the saved configuration.
func (s *Server) settings() store.Settings { return s.store.Settings() }

// credentials returns the account id and token or a 412 precondition error.
func (s *Server) credentials() (string, string, error) {
	current := s.store.Settings()
	if !current.Configured() {
		return current.AccountID, current.APIToken,
			model.NewError(http.StatusPreconditionFailed, "precondition_failed", "account id and API token must be configured first")
	}
	return current.AccountID, current.APIToken, nil
}

// addActivity records an entry and streams it.
func (s *Server) addActivity(kind, title, tone string) {
	item, err := s.store.AddActivity(kind, title, tone)
	if err != nil {
		return
	}
	s.hub.Broadcast("activity", item)
}

// localActive reports whether tunnelID is the tunnel the supervisor runs.
func (s *Server) localActive(tunnelID string) bool {
	if s.engine == nil {
		return false
	}
	current := s.engine.State()
	return current.TunnelID == tunnelID && model.IsLiveState(current.State)
}

// cloudflaredInstalled reports whether the binary exists on disk. Demo mode
// always reports true.
func (s *Server) cloudflaredInstalled() bool {
	if s.opts.Demo {
		return true
	}
	info, err := os.Stat(s.opts.CloudflaredPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ensureDemoStarted makes sure demo mode has an engine even when main wires it
// later; it is a no-op in production.
func (s *Server) engineState() model.Runtime {
	if s.engine == nil {
		return model.Runtime{State: model.StateDisconnected}
	}
	return s.engine.State()
}

func (s *Server) engineMetrics() model.RuntimeMetrics {
	if s.engine == nil {
		return model.RuntimeMetrics{}
	}
	return s.engine.Metrics()
}
