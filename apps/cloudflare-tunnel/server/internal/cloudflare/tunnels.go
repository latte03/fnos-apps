package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// cfTunnel mirrors the subset of the Cloudflare tunnel object we use.
type cfTunnel struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	CreatedAt    string            `json:"created_at"`
	Connections  []json.RawMessage `json:"connections"`
	ConfigSrc    string            `json:"config_src"`
	RemoteConfig bool              `json:"remote_config"`
}

func (t cfTunnel) model() model.Tunnel {
	status := strings.ToLower(strings.TrimSpace(t.Status))
	switch status {
	case model.TunnelHealthy, model.TunnelDegraded, model.TunnelInactive:
	case "down":
		status = model.TunnelError
	default:
		status = model.TunnelError
	}
	src := strings.ToLower(strings.TrimSpace(t.ConfigSrc))
	if src != "cloudflare" && src != "local" {
		src = "local"
		if t.RemoteConfig {
			src = "cloudflare"
		}
	}
	return model.Tunnel{
		ID:          t.ID,
		Name:        t.Name,
		Status:      status,
		CreatedAt:   t.CreatedAt,
		Connections: len(t.Connections),
		ConfigSrc:   src,
	}
}

// VerifyToken checks the token itself plus the account tunnel and zone scopes.
// Transport failures are returned as errors; scope failures are described in
// VerifyResult.Error so the UI can render them inline.
func (c *Client) VerifyToken(ctx context.Context, accountID, token string) (model.VerifyResult, error) {
	res := model.VerifyResult{AccountID: accountID}
	if strings.TrimSpace(token) == "" {
		res.Error = "API token is not configured"
		return res, nil
	}
	verifyErr := c.verifyTokenStatus(ctx, accountID, token, &res)

	var problems []string
	if accountID == "" {
		problems = append(problems, "account id is not configured")
	} else {
		_, terr := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID)+"/cfd_tunnel?per_page=1", nil)
		if terr != nil {
			problems = append(problems, "tunnel access: "+errMessage(terr))
		} else {
			res.TunnelAccess = true
		}
	}
	if _, zerr := c.do(ctx, token, http.MethodGet, "/zones?per_page=1", nil); zerr != nil {
		problems = append(problems, "zone access: "+errMessage(zerr))
	} else {
		res.ZoneAccess = true
	}
	if res.TokenStatus != "active" {
		switch {
		case verifyErr != nil:
			problems = append(problems, "token: "+errMessage(verifyErr))
		case res.TokenStatus != "unknown":
			problems = append(problems, "token status: "+res.TokenStatus)
		}
	}
	res.OK = res.TokenStatus == "active" && res.TunnelAccess && res.ZoneAccess
	res.Error = strings.Join(problems, "; ")
	return res, nil
}

// verifyTokenStatus accepts both user tokens and account-owned tokens. The
// /user/tokens/verify endpoint rejects account-owned tokens with 401, so it is
// only a fallback once the account-scoped endpoint has failed.
func (c *Client) verifyTokenStatus(ctx context.Context, accountID, token string, res *model.VerifyResult) error {
	if accountID != "" {
		raw, err := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID)+"/tokens/verify", nil)
		if err == nil {
			res.TokenStatus = tokenStatus(raw)
			return nil
		}
	}
	raw, err := c.do(ctx, token, http.MethodGet, "/user/tokens/verify", nil)
	if err == nil {
		res.TokenStatus = tokenStatus(raw)
		return nil
	}
	res.TokenStatus = "unknown"
	return err
}

func tokenStatus(raw []byte) string {
	var verify struct {
		Status string `json:"status"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &verify)
	}
	status := strings.TrimSpace(verify.Status)
	if status == "" {
		return "unknown"
	}
	return status
}

// AccountName reads the account display name. It is best effort: a token that
// cannot read the account is not an error for the UI.
func (c *Client) AccountName(ctx context.Context, accountID, token string) string {
	if accountID == "" || token == "" {
		return ""
	}
	raw, err := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID), nil)
	if err != nil || len(raw) == 0 {
		return ""
	}
	var account struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &account); err != nil {
		return ""
	}
	return strings.TrimSpace(account.Name)
}

// ListTunnels returns the non-deleted remote tunnels of the account.
func (c *Client) ListTunnels(ctx context.Context, accountID, token string) ([]model.Tunnel, error) {
	if accountID == "" {
		return nil, model.NewError(http.StatusPreconditionFailed, "precondition_failed", "account id is not configured")
	}
	raw, err := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID)+"/cfd_tunnel?is_deleted=false&per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var rows []cfTunnel
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable tunnel list")
		}
	}
	out := make([]model.Tunnel, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.model())
	}
	return out, nil
}

// CreateTunnel creates a remotely managed tunnel.
func (c *Client) CreateTunnel(ctx context.Context, accountID, token, name string) (model.Tunnel, error) {
	body := map[string]any{"name": name, "config_src": "cloudflare"}
	raw, err := c.do(ctx, token, http.MethodPost, "/accounts/"+esc(accountID)+"/cfd_tunnel", body)
	if err != nil {
		return model.Tunnel{}, err
	}
	var row cfTunnel
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &row); err != nil {
			return model.Tunnel{}, model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable tunnel")
		}
	}
	out := row.model()
	if out.Status == "" || out.Status == model.TunnelError {
		out.Status = model.TunnelInactive
	}
	if out.ConfigSrc == "" {
		out.ConfigSrc = "cloudflare"
	}
	return out, nil
}

// DeleteTunnel removes a tunnel.
func (c *Client) DeleteTunnel(ctx context.Context, accountID, token, tunnelID string) error {
	_, err := c.do(ctx, token, http.MethodDelete, "/accounts/"+esc(accountID)+"/cfd_tunnel/"+esc(tunnelID), nil)
	return err
}

// TunnelToken fetches the connector token for a tunnel.
func (c *Client) TunnelToken(ctx context.Context, accountID, token, tunnelID string) (string, error) {
	raw, err := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID)+"/cfd_tunnel/"+esc(tunnelID)+"/token", nil)
	if err != nil {
		return "", err
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable tunnel token")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an empty tunnel token")
	}
	return value, nil
}

// GetConfig returns the full stored configuration of a tunnel.
func (c *Client) GetConfig(ctx context.Context, accountID, token, tunnelID string) (map[string]any, error) {
	raw, err := c.do(ctx, token, http.MethodGet, "/accounts/"+esc(accountID)+"/cfd_tunnel/"+esc(tunnelID)+"/configurations", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Config map[string]any `json:"config"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable tunnel configuration")
		}
	}
	if payload.Config == nil {
		payload.Config = map[string]any{}
	}
	return payload.Config, nil
}

// PutConfig replaces the tunnel configuration. Callers must merge first so no
// key outside "ingress" is lost.
func (c *Client) PutConfig(ctx context.Context, accountID, token, tunnelID string, cfg map[string]any) error {
	body := map[string]any{"config": cfg}
	_, err := c.do(ctx, token, http.MethodPut, "/accounts/"+esc(accountID)+"/cfd_tunnel/"+esc(tunnelID)+"/configurations", body)
	return err
}

// errMessage renders an error for VerifyResult without leaking secrets.
func errMessage(err error) string {
	if err == nil {
		return ""
	}
	if apiErr, ok := err.(*model.APIError); ok && apiErr.Message != "" {
		return apiErr.Message
	}
	return err.Error()
}
