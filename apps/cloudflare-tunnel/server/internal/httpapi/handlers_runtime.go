package httpapi

import (
	"net/http"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

func (s *Server) handleRuntimeGet(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.engineState())
}

func (s *Server) handleRuntimeMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.engineMetrics())
}

type connectRequest struct {
	TunnelID string `json:"tunnelId"`
}

type runtimeStatusResponse struct {
	Status model.Runtime `json:"status"`
}

func (s *Server) handleRuntimeConnect(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "runtime is not available"))
		return
	}
	accountID, token, err := s.credentials()
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var body connectRequest
	if !decodeJSON(w, r, &body, false) {
		return
	}
	tunnelID := body.TunnelID
	if tunnelID == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "tunnelId is required"))
		return
	}
	tunnels, err := s.cloud.ListTunnels(r.Context(), accountID, token)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	name := ""
	for _, tunnel := range tunnels {
		if tunnel.ID == tunnelID {
			name = tunnel.Name
			break
		}
	}
	if name == "" {
		writeAPIError(w, model.NewError(http.StatusNotFound, "not_found", "tunnel not found"))
		return
	}
	connectorToken, err := s.cloud.TunnelToken(r.Context(), accountID, token, tunnelID)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if err := s.engine.Connect(tunnelID, name, connectorToken); err != nil {
		writeAPIError(w, err)
		return
	}
	s.rememberTunnel(tunnelID, name, model.StateConnecting)
	s.addActivity("connect", "隧道 "+name+" 已连接", "green")
	writeJSON(w, http.StatusAccepted, runtimeStatusResponse{Status: s.engine.State()})
}

func (s *Server) handleRuntimeDisconnect(w http.ResponseWriter, _ *http.Request) {
	if s.engine == nil {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "runtime is not available"))
		return
	}
	name := s.engine.State().TunnelName
	if err := s.engine.Disconnect(); err != nil {
		writeAPIError(w, err)
		return
	}
	if name == "" {
		name = s.store.State().TunnelName
	}
	s.rememberTunnel(s.store.State().TunnelID, name, model.StateDisconnected)
	s.addActivity("disconnect", "隧道 "+name+" 已断开", "neutral")
	writeJSON(w, http.StatusOK, runtimeStatusResponse{Status: s.engine.State()})
}
