package httpapi

import (
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

// settingsView is the only shape in which settings leave the process. The API
// token is reduced to a masked preview.
type settingsView struct {
	AccountID        string `json:"accountId"`
	APITokenSet      bool   `json:"apiTokenSet"`
	APITokenPreview  string `json:"apiTokenPreview"`
	Protocol         string `json:"protocol"`
	EdgeIPVersion    string `json:"edgeIpVersion"`
	AutoConnect      bool   `json:"autoConnect"`
	AutoReconnect    bool   `json:"autoReconnect"`
	MaxReconnect     int    `json:"maxReconnect"`
	LogRetentionDays int    `json:"logRetentionDays"`
	MaxLogLines      int    `json:"maxLogLines"`
}

func newSettingsView(current store.Settings) settingsView {
	return settingsView{
		AccountID:        current.AccountID,
		APITokenSet:      current.APIToken != "",
		APITokenPreview:  store.PreviewToken(current.APIToken),
		Protocol:         current.Protocol,
		EdgeIPVersion:    current.EdgeIPVersion,
		AutoConnect:      current.AutoConnect,
		AutoReconnect:    current.AutoReconnect,
		MaxReconnect:     current.MaxReconnect,
		LogRetentionDays: current.LogRetentionDays,
		MaxLogLines:      current.MaxLogLines,
	}
}

// settingsPatch accepts partial updates. Absent fields keep their value; an
// empty apiToken also keeps the current token.
type settingsPatch struct {
	AccountID        *string `json:"accountId"`
	APIToken         *string `json:"apiToken"`
	Protocol         *string `json:"protocol"`
	EdgeIPVersion    *string `json:"edgeIpVersion"`
	AutoConnect      *bool   `json:"autoConnect"`
	AutoReconnect    *bool   `json:"autoReconnect"`
	MaxReconnect     *int    `json:"maxReconnect"`
	LogRetentionDays *int    `json:"logRetentionDays"`
	MaxLogLines      *int    `json:"maxLogLines"`
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, newSettingsView(s.store.Settings()))
}

func (s *Server) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var patch settingsPatch
	if !decodeJSON(w, r, &patch, false) {
		return
	}
	previous := s.store.Settings()
	next := previous
	if patch.AccountID != nil {
		next.AccountID = strings.TrimSpace(*patch.AccountID)
	}
	if patch.APIToken != nil {
		if token := strings.TrimSpace(*patch.APIToken); token != "" {
			next.APIToken = token
		}
	}
	if patch.Protocol != nil {
		next.Protocol = strings.ToLower(strings.TrimSpace(*patch.Protocol))
	}
	if patch.EdgeIPVersion != nil {
		next.EdgeIPVersion = strings.ToLower(strings.TrimSpace(*patch.EdgeIPVersion))
	}
	if patch.AutoConnect != nil {
		next.AutoConnect = *patch.AutoConnect
	}
	if patch.AutoReconnect != nil {
		next.AutoReconnect = *patch.AutoReconnect
	}
	if patch.MaxReconnect != nil {
		next.MaxReconnect = *patch.MaxReconnect
	}
	if patch.LogRetentionDays != nil {
		next.LogRetentionDays = *patch.LogRetentionDays
	}
	if patch.MaxLogLines != nil {
		next.MaxLogLines = *patch.MaxLogLines
	}
	if err := validateSettings(next); err != nil {
		writeAPIError(w, err)
		return
	}
	if err := s.store.SaveSettings(next); err != nil {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "cannot save settings"))
		return
	}
	if next.AccountID != previous.AccountID || next.APIToken != previous.APIToken {
		if inv, ok := s.cloud.(interface{ InvalidateZones() }); ok {
			inv.InvalidateZones()
		}
	}
	s.applySettings(next)
	s.addActivity("settings", "设置已更新", "neutral")
	writeJSON(w, http.StatusOK, newSettingsView(next))
}

type verifyRequest struct {
	AccountID *string `json:"accountId"`
	APIToken  *string `json:"apiToken"`
}

func (s *Server) handleSettingsVerify(w http.ResponseWriter, r *http.Request) {
	var body verifyRequest
	if !decodeJSON(w, r, &body, true) {
		return
	}
	current := s.store.Settings()
	accountID := current.AccountID
	if body.AccountID != nil && strings.TrimSpace(*body.AccountID) != "" {
		accountID = strings.TrimSpace(*body.AccountID)
	}
	if accountID != "" && !accountIDPattern.MatchString(accountID) {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "accountId must be 32 hexadecimal characters"))
		return
	}
	token := current.APIToken
	if body.APIToken != nil && strings.TrimSpace(*body.APIToken) != "" {
		token = strings.TrimSpace(*body.APIToken)
	}
	if token == "" {
		writeJSON(w, http.StatusOK, model.VerifyResult{
			AccountID:   accountID,
			TokenStatus: "unknown",
			Error:       "API token is not configured",
		})
		return
	}
	result, err := s.cloud.VerifyToken(r.Context(), accountID, token)
	if err != nil {
		result = model.VerifyResult{AccountID: accountID, TokenStatus: "unknown", Error: asAPIError(err).Message}
	}
	result.AccountID = accountID
	writeJSON(w, http.StatusOK, result)
}

func validateSettings(next store.Settings) error {
	if next.AccountID != "" && !accountIDPattern.MatchString(next.AccountID) {
		return model.NewError(http.StatusBadRequest, "validation_failed", "accountId must be 32 hexadecimal characters")
	}
	switch next.Protocol {
	case "auto", "quic", "http2":
	default:
		return model.NewError(http.StatusBadRequest, "validation_failed", "protocol must be auto, quic or http2")
	}
	switch next.EdgeIPVersion {
	case "auto", "4", "6":
	default:
		return model.NewError(http.StatusBadRequest, "validation_failed", "edgeIpVersion must be auto, 4 or 6")
	}
	if next.MaxReconnect < 0 || next.MaxReconnect > 50 {
		return model.NewError(http.StatusBadRequest, "validation_failed", "maxReconnect must be between 0 and 50")
	}
	if next.LogRetentionDays < 1 || next.LogRetentionDays > 365 {
		return model.NewError(http.StatusBadRequest, "validation_failed", "logRetentionDays must be between 1 and 365")
	}
	if next.MaxLogLines < 200 || next.MaxLogLines > 10000 {
		return model.NewError(http.StatusBadRequest, "validation_failed", "maxLogLines must be between 200 and 10000")
	}
	return nil
}

// applySettings pushes the runtime relevant parts into the supervisor.
func (s *Server) applySettings(next store.Settings) {
	if s.engine == nil {
		return
	}
	if supervisor, ok := s.engine.(interface {
		SetPolicy(bool, int)
		SetTunnelOptions(string, string)
		SetLogCapacity(int)
	}); ok {
		supervisor.SetPolicy(next.AutoReconnect, next.MaxReconnect)
		supervisor.SetTunnelOptions(next.Protocol, next.EdgeIPVersion)
		supervisor.SetLogCapacity(next.MaxLogLines)
	}
}
