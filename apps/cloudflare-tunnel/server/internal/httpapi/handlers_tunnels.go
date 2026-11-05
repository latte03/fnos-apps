package httpapi

import (
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

type tunnelsResponse struct {
	Configured bool           `json:"configured"`
	Tunnels    []model.Tunnel `json:"tunnels"`
}

func (s *Server) handleTunnelsList(w http.ResponseWriter, r *http.Request) {
	current := s.store.Settings()
	if !current.Configured() {
		writeJSON(w, http.StatusOK, tunnelsResponse{Configured: false, Tunnels: []model.Tunnel{}})
		return
	}
	tunnels, err := s.cloud.ListTunnels(r.Context(), current.AccountID, current.APIToken)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if tunnels == nil {
		tunnels = []model.Tunnel{}
	}
	for i := range tunnels {
		tunnels[i].LocalActive = s.localActive(tunnels[i].ID)
	}
	writeJSON(w, http.StatusOK, tunnelsResponse{Configured: true, Tunnels: tunnels})
}

type createTunnelRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleTunnelsCreate(w http.ResponseWriter, r *http.Request) {
	accountID, token, err := s.credentials()
	if err != nil {
		writeAPIError(w, err)
		return
	}
	var body createTunnelRequest
	if !decodeJSON(w, r, &body, false) {
		return
	}
	name := strings.ToLower(strings.TrimSpace(body.Name))
	if !tunnelNamePattern.MatchString(name) {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed",
			"name must be lowercase letters, digits and dashes, starting with a letter or digit"))
		return
	}
	tunnel, err := s.cloud.CreateTunnel(r.Context(), accountID, token, name)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	tunnel.LocalActive = s.localActive(tunnel.ID)
	s.addActivity("tunnel", "隧道 "+tunnel.Name+" 已创建", "blue")
	writeJSON(w, http.StatusCreated, tunnel)
}

func (s *Server) handleTunnelsDelete(w http.ResponseWriter, r *http.Request) {
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
	name := s.tunnelNameHint(r, accountID, token, tunnelID)
	if s.localActive(tunnelID) {
		if err := s.engine.Disconnect(); err != nil {
			writeAPIError(w, err)
			return
		}
	}
	if err := s.cloud.DeleteTunnel(r.Context(), accountID, token, tunnelID); err != nil {
		writeAPIError(w, err)
		return
	}
	if saved := s.store.State(); saved.TunnelID == tunnelID {
		clearState := saved
		clearState.TunnelID = ""
		clearState.TunnelName = ""
		clearState.LastState = model.StateDisconnected
		_ = s.store.SaveState(clearState)
	}
	s.addActivity("tunnel", "隧道 "+name+" 已删除", "danger")
	writeNoContent(w)
}

// tunnelNameHint looks up a tunnel name without failing the request.
func (s *Server) tunnelNameHint(r *http.Request, accountID, token, tunnelID string) string {
	if current := s.engineState(); current.TunnelID == tunnelID && current.TunnelName != "" {
		return current.TunnelName
	}
	tunnels, err := s.cloud.ListTunnels(r.Context(), accountID, token)
	if err != nil {
		return tunnelID
	}
	for _, tunnel := range tunnels {
		if tunnel.ID == tunnelID {
			return tunnel.Name
		}
	}
	return tunnelID
}

func (s *Server) handleTunnelToken(w http.ResponseWriter, r *http.Request) {
	// The connector token is only ever handed to a browser session, never to a
	// bearer key, and it is never logged.
	if authKind(r) != "session" {
		writeAPIError(w, model.NewError(http.StatusUnauthorized, "unauthorized", "a browser session is required for this endpoint"))
		return
	}
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
	value, err := s.cloud.TunnelToken(r.Context(), accountID, token, tunnelID)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": value})
}

// rememberTunnel persists the boot reconnect hint.
func (s *Server) rememberTunnel(tunnelID, tunnelName, state string) {
	next := store.State{
		TunnelID:    tunnelID,
		TunnelName:  tunnelName,
		AutoConnect: s.store.Settings().AutoConnect,
		LastState:   state,
	}
	_ = s.store.SaveState(next)
}
