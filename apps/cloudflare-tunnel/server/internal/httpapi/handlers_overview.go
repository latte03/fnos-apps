package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// routeCountTTL caches the ingress fan-out used by the overview page.
const routeCountTTL = 30 * time.Second

// routeCountTimeout bounds the fan-out so a slow API cannot stall the overview.
const routeCountTimeout = 6 * time.Second

type overviewResponse struct {
	Runtime            model.Runtime      `json:"runtime"`
	TunnelCount        int                `json:"tunnelCount"`
	HealthyCount       int                `json:"healthyCount"`
	RouteCount         int                `json:"routeCount"`
	ZoneCount          int                `json:"zoneCount"`
	Metrics            model.MetricValues `json:"metrics"`
	CloudflaredVersion string             `json:"cloudflaredVersion"`
	Activity           []model.Activity   `json:"activity"`
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	resp := overviewResponse{
		Runtime:            s.engineState(),
		Metrics:            s.engineMetrics().Values(),
		CloudflaredVersion: s.opts.CloudflaredVersion,
		Activity:           s.recentActivity(10),
	}
	current := s.store.Settings()
	if current.Configured() {
		if tunnels, err := s.cloud.ListTunnels(r.Context(), current.AccountID, current.APIToken); err == nil {
			resp.TunnelCount = len(tunnels)
			for _, tunnel := range tunnels {
				if tunnel.Status == model.TunnelHealthy {
					resp.HealthyCount++
				}
			}
			resp.RouteCount = s.countRoutes(r, current.AccountID, current.APIToken, tunnels)
		}
		if zones, err := s.cloud.ListZones(r.Context(), current.APIToken); err == nil {
			resp.ZoneCount = len(zones)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) recentActivity(limit int) []model.Activity {
	items := s.store.Activity()
	if len(items) > limit {
		items = items[:limit]
	}
	if items == nil {
		items = []model.Activity{}
	}
	return items
}

// countRoutes sums the ingress rules of every tunnel, cached for a short while.
func (s *Server) countRoutes(r *http.Request, accountID, token string, tunnels []model.Tunnel) int {
	s.mu.Lock()
	if s.routeCountKey == accountID && time.Since(s.routeCountAt) < routeCountTTL {
		cached := s.routeCount
		s.mu.Unlock()
		return cached
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), routeCountTimeout)
	defer cancel()
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		total int
	)
	for _, tunnel := range tunnels {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			cfg, err := s.cloud.GetConfig(ctx, accountID, token, id)
			if err != nil {
				return
			}
			routes, _, _ := model.IngressRoutes(cfg)
			mu.Lock()
			total += len(routes)
			mu.Unlock()
		}(tunnel.ID)
	}
	wg.Wait()

	s.mu.Lock()
	s.routeCount = total
	s.routeCountAt = time.Now()
	s.routeCountKey = accountID
	s.mu.Unlock()
	return total
}
