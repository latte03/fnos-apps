package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// routesRequest is the PUT /tunnels/{id}/routes body.
type routesRequest struct {
	Routes          []model.Route `json:"routes"`
	CatchAllService string        `json:"catchAllService"`
}

func (s *Server) handleRoutesGet(w http.ResponseWriter, r *http.Request) {
	accountID, token, err := s.credentials()
	if err != nil {
		writeAPIError(w, err)
		return
	}
	tunnelID := r.PathValue("id")
	if tunnelID == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "tunnel id is required"))
		return
	}
	cfg, err := s.cloud.GetConfig(r.Context(), accountID, token, tunnelID)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	routes, catchAll, raw := model.IngressRoutes(cfg)
	writeJSON(w, http.StatusOK, model.Routes{
		Routes:          routes,
		CatchAllService: catchAll,
		RawIngressCount: raw,
	})
}

func (s *Server) handleRoutesPut(w http.ResponseWriter, r *http.Request) {
	accountID, token, err := s.credentials()
	if err != nil {
		writeAPIError(w, err)
		return
	}
	tunnelID := r.PathValue("id")
	if tunnelID == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "tunnel id is required"))
		return
	}
	var body routesRequest
	if !decodeJSON(w, r, &body, false) {
		return
	}
	routes, catchAll, err := normaliseRoutes(body.Routes, body.CatchAllService)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	// Read first: every non-ingress key must survive the PUT.
	cfg, err := s.cloud.GetConfig(r.Context(), accountID, token, tunnelID)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	merged := model.WithIngress(cfg, routes, catchAll)
	if err := s.cloud.PutConfig(r.Context(), accountID, token, tunnelID, merged); err != nil {
		writeAPIError(w, err)
		return
	}
	s.addActivity("route", fmt.Sprintf("路由已保存（%d 条）", len(routes)), "blue")
	writeJSON(w, http.StatusOK, model.Routes{
		Routes:          routes,
		CatchAllService: catchAll,
		RawIngressCount: len(routes) + 1,
	})
}

type dnsRequest struct {
	TunnelID string `json:"tunnelId"`
	Hostname string `json:"hostname"`
}

func (s *Server) handleDNS(w http.ResponseWriter, r *http.Request) {
	_, token, err := s.credentials()
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var body dnsRequest
	if !decodeJSON(w, r, &body, false) {
		return
	}
	if body.TunnelID == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "tunnelId is required"))
		return
	}
	if !hostnamePattern.MatchString(normaliseHostname(body.Hostname)) {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "hostname is invalid"))
		return
	}
	result, err := s.cloud.UpsertCNAME(r.Context(), token, body.TunnelID, normaliseHostname(body.Hostname))
	if err != nil {
		writeAPIError(w, err)
		return
	}
	switch result.Action {
	case "created":
		s.addActivity("dns", "CNAME "+result.Name+" 已创建", "teal")
	case "updated":
		s.addActivity("dns", "CNAME "+result.Name+" 已更新", "teal")
	}
	writeJSON(w, http.StatusOK, result)
}

func normaliseHostname(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}
