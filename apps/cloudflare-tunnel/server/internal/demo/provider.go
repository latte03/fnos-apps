// Package demo provides a fully offline stand-in for the Cloudflare API, the
// cloudflared process and its metrics endpoint. Nothing here touches the
// network or the real data share.
package demo

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// Seeded identities. These are obviously fake and never used against the API.
const (
	AccountID = "a1b2c3d4e5f60718293a4b5c6d7e8f90"
	APIToken  = "cf_demo_9f3a2b7c4e1d8a6f0b5c"
)

// Tunnel ids of the seeded account.
const (
	TunnelMainID   = "5b1f7c2a-9d4e-4a51-8b3c-1e0d2f3a4b5c"
	TunnelLabID    = "7c2e8d3b-1f5a-4b62-9c4d-2f1e3a5b6c7d"
	TunnelBackupID = "9d3f9e4c-2a6b-4c73-ad5e-3a2f4b6c7d8e"
)

// Provider is an in-memory Cloudflare API.
type Provider struct {
	mu       sync.Mutex
	tunnels  []model.Tunnel
	routes   map[string][]model.Route
	catchAll map[string]string
	zones    []model.Zone
	dns      map[string]model.DNSResult
	seq      int
}

// NewProvider returns the seeded demo account.
func NewProvider() *Provider {
	p := &Provider{
		tunnels: []model.Tunnel{
			{ID: TunnelMainID, Name: "fnos-main", Status: model.TunnelHealthy, CreatedAt: "2026-09-18T09:12:40Z", Connections: 2, ConfigSrc: "cloudflare"},
			{ID: TunnelLabID, Name: "home-lab", Status: model.TunnelInactive, CreatedAt: "2026-10-02T13:41:06Z", Connections: 0, ConfigSrc: "cloudflare"},
			{ID: TunnelBackupID, Name: "edge-backup", Status: model.TunnelError, CreatedAt: "2026-10-21T05:03:55Z", Connections: 0, ConfigSrc: "cloudflare"},
		},
		routes: map[string][]model.Route{
			TunnelMainID: {
				{Hostname: "nas.example.com", Path: "/", Service: "http://127.0.0.1:5666"},
				{Hostname: "ha.example.com", Path: "/", Service: "http://192.168.0.42:8123"},
				{Hostname: "ai.example.com", Path: "/", Service: "http://127.0.0.1:3000"},
				{Hostname: "pa.example.com", Path: "/", Service: "http://127.0.0.1:19091"},
			},
			TunnelLabID: {
				{Hostname: "home.example.com", Path: "/", Service: "http://192.168.0.42:8123"},
				{Hostname: "files.example.com", Path: "/", Service: "http://192.168.0.42:8080"},
			},
			TunnelBackupID: {
				{Hostname: "backup.example.com", Path: "/", Service: "http://127.0.0.1:9000"},
			},
		},
		catchAll: map[string]string{
			TunnelMainID:   model.DefaultCatchAll,
			TunnelLabID:    model.DefaultCatchAll,
			TunnelBackupID: model.DefaultCatchAll,
		},
		zones: []model.Zone{
			{ID: "023e105f4ecef8ad9ca31a8372d0c353", Name: "example.com"},
			{ID: "1f2e3d4c5b6a79880f1e2d3c4b5a6978", Name: "example.net"},
		},
		dns: map[string]model.DNSResult{},
	}
	return p
}

// VerifyToken always reports a healthy, fully scoped demo token.
func (p *Provider) VerifyToken(_ context.Context, accountID, token string) (model.VerifyResult, error) {
	res := model.VerifyResult{
		OK:           true,
		AccountID:    accountID,
		TokenStatus:  "active",
		TunnelAccess: true,
		ZoneAccess:   true,
	}
	if strings.TrimSpace(token) == "" {
		res.OK = false
		res.TokenStatus = "unknown"
		res.Error = "API token is not configured"
	}
	return res, nil
}

// AccountName returns the demo account label.
func (p *Provider) AccountName(_ context.Context, accountID, _ string) string {
	if accountID == "" {
		return ""
	}
	return "Demo Account"
}

// ListTunnels copies the seeded tunnel list.
func (p *Provider) ListTunnels(_ context.Context, _, _ string) ([]model.Tunnel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]model.Tunnel, len(p.tunnels))
	copy(out, p.tunnels)
	return out, nil
}

// CreateTunnel appends a tunnel to the demo account.
func (p *Provider) CreateTunnel(_ context.Context, _, _, name string) (model.Tunnel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.tunnels {
		if strings.EqualFold(t.Name, name) {
			return model.Tunnel{}, model.Errorf(409, "conflict", "a tunnel named %s already exists", name)
		}
	}
	p.seq++
	tunnel := model.Tunnel{
		ID:        fmt.Sprintf("demo%02d-4b62-9c4d-2f1e3a5b6c7d", p.seq),
		Name:      name,
		Status:    model.TunnelInactive,
		CreatedAt: model.FormatTS(time.Now()),
		ConfigSrc: "cloudflare",
	}
	p.tunnels = append(p.tunnels, tunnel)
	p.catchAll[tunnel.ID] = model.DefaultCatchAll
	return tunnel, nil
}

// DeleteTunnel removes a tunnel and its routes.
func (p *Provider) DeleteTunnel(_ context.Context, _, _, tunnelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, t := range p.tunnels {
		if t.ID == tunnelID {
			p.tunnels = append(p.tunnels[:i], p.tunnels[i+1:]...)
			delete(p.routes, tunnelID)
			delete(p.catchAll, tunnelID)
			return nil
		}
	}
	return model.NewError(404, "not_found", "tunnel not found")
}

// TunnelToken returns a fake connector token.
func (p *Provider) TunnelToken(_ context.Context, _, _, tunnelID string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.exists(tunnelID) {
		return "", model.NewError(404, "not_found", "tunnel not found")
	}
	return "eyJhIjoiZGVtbyIsInQiOiJkZW1vLXR1bm5lbCIsInMiOiJkZW1vIn0", nil
}

// GetConfig renders the stored ingress.
func (p *Provider) GetConfig(_ context.Context, _, _, tunnelID string) (map[string]any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.exists(tunnelID) {
		return nil, model.NewError(404, "not_found", "tunnel not found")
	}
	catchAll := p.catchAll[tunnelID]
	if catchAll == "" {
		catchAll = model.DefaultCatchAll
	}
	cfg := map[string]any{"warp-routing": map[string]any{"enabled": false}}
	return model.WithIngress(cfg, p.routes[tunnelID], catchAll), nil
}

// PutConfig stores the ingress part of cfg.
func (p *Provider) PutConfig(_ context.Context, _, _, tunnelID string, cfg map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.exists(tunnelID) {
		return model.NewError(404, "not_found", "tunnel not found")
	}
	routes, catchAll, _ := model.IngressRoutes(cfg)
	p.routes[tunnelID] = routes
	p.catchAll[tunnelID] = catchAll
	return nil
}

// ListZones copies the demo zones.
func (p *Provider) ListZones(_ context.Context, _ string) ([]model.Zone, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]model.Zone, len(p.zones))
	copy(out, p.zones)
	return out, nil
}

// UpsertCNAME simulates the DNS bookkeeping.
func (p *Provider) UpsertCNAME(_ context.Context, _, tunnelID, hostname string) (model.DNSResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	host := strings.ToLower(strings.TrimSpace(hostname))
	zone := model.Zone{}
	for _, z := range p.zones {
		if host == z.Name || strings.HasSuffix(host, "."+z.Name) {
			if zone.Name == "" || len(z.Name) > len(zone.Name) {
				zone = z
			}
		}
	}
	if zone.Name == "" {
		return model.DNSResult{}, model.Errorf(400, "validation_failed", "no zone in this account covers %s", host)
	}
	content := tunnelID + ".cfargotunnel.com"
	prev, seen := p.dns[host]
	result := model.DNSResult{
		ZoneID:   zone.ID,
		ZoneName: zone.Name,
		Name:     host,
		Content:  content,
		Proxied:  true,
	}
	p.seq++
	result.RecordID = fmt.Sprintf("demo-record-%03d", p.seq)
	switch {
	case !seen:
		result.Action = "created"
	case prev.Content == content:
		result.RecordID = prev.RecordID
		result.Action = "unchanged"
	default:
		result.Action = "updated"
	}
	p.dns[host] = result
	return result, nil
}

func (p *Provider) exists(tunnelID string) bool {
	for _, t := range p.tunnels {
		if t.ID == tunnelID {
			return true
		}
	}
	return false
}
