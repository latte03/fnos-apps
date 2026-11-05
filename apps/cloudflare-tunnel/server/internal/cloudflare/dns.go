package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// TunnelCNAME is the DNS target every remotely managed tunnel uses.
func TunnelCNAME(tunnelID string) string {
	return strings.ToLower(strings.TrimSpace(tunnelID)) + ".cfargotunnel.com"
}

type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

// UpsertCNAME makes sure hostname points at the tunnel. It creates the record,
// updates a stale CNAME, reports "unchanged" for an identical record, and
// refuses to clobber a non-CNAME record.
func (c *Client) UpsertCNAME(ctx context.Context, token, tunnelID, hostname string) (model.DNSResult, error) {
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	if host == "" {
		return model.DNSResult{}, model.NewError(http.StatusBadRequest, "validation_failed", "hostname is required")
	}
	zones, err := c.ListZones(ctx, token)
	if err != nil {
		return model.DNSResult{}, err
	}
	zone, ok := MatchZone(zones, host)
	if !ok {
		return model.DNSResult{}, model.Errorf(http.StatusBadRequest, "validation_failed", "no zone in this account covers %s", host)
	}
	content := TunnelCNAME(tunnelID)
	result := model.DNSResult{
		ZoneID:   zone.ID,
		ZoneName: zone.Name,
		Name:     host,
		Content:  content,
		Proxied:  true,
	}

	raw, err := c.do(ctx, token, http.MethodGet, "/zones/"+esc(zone.ID)+"/dns_records?name="+queryEscape(host)+"&per_page=100", nil)
	if err != nil {
		return model.DNSResult{}, err
	}
	var records []dnsRecord
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &records); err != nil {
			return model.DNSResult{}, model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable DNS answer")
		}
	}
	var existing *dnsRecord
	for i := range records {
		if strings.EqualFold(strings.TrimSuffix(records[i].Name, "."), host) {
			existing = &records[i]
			break
		}
	}
	if existing == nil {
		body := map[string]any{"type": "CNAME", "name": host, "content": content, "proxied": true}
		raw, err := c.do(ctx, token, http.MethodPost, "/zones/"+esc(zone.ID)+"/dns_records", body)
		if err != nil {
			return model.DNSResult{}, err
		}
		created := model.DNSResult{Action: "created"}
		if len(raw) > 0 {
			var rec dnsRecord
			if err := json.Unmarshal(raw, &rec); err == nil {
				created.RecordID = rec.ID
			}
		}
		result.RecordID = created.RecordID
		result.Action = "created"
		return result, nil
	}

	result.RecordID = existing.ID
	if !strings.EqualFold(strings.TrimSpace(existing.Type), "CNAME") {
		return model.DNSResult{}, model.Errorf(http.StatusConflict, "conflict",
			"a %s record named %s already exists", strings.ToUpper(existing.Type), host)
	}
	if strings.EqualFold(strings.TrimSuffix(strings.TrimSpace(existing.Content), "."), content) && existing.Proxied {
		result.Action = "unchanged"
		return result, nil
	}
	body := map[string]any{"type": "CNAME", "name": host, "content": content, "proxied": true}
	if _, err := c.do(ctx, token, http.MethodPut, "/zones/"+esc(zone.ID)+"/dns_records/"+esc(existing.ID), body); err != nil {
		return model.DNSResult{}, err
	}
	result.Action = "updated"
	return result, nil
}
