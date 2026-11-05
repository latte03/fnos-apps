package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// ListZones returns the zones the token can read, cached for ZoneTTL.
func (c *Client) ListZones(ctx context.Context, token string) ([]model.Zone, error) {
	if zones := c.cachedZones(); zones != nil {
		return zones, nil
	}
	raw, err := c.do(ctx, token, http.MethodGet, "/zones?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, model.NewError(http.StatusBadGateway, "upstream_error", "cloudflare returned an unreadable zone list")
		}
	}
	zones := make([]model.Zone, 0, len(rows))
	for _, row := range rows {
		name := strings.ToLower(strings.TrimSpace(row.Name))
		if row.ID == "" || name == "" {
			continue
		}
		zones = append(zones, model.Zone{ID: row.ID, Name: name})
	}
	c.storeZones(zones)
	return zones, nil
}

// InvalidateZones drops the cached zone list.
func (c *Client) InvalidateZones() {
	c.mu.Lock()
	c.zones = nil
	c.zonesAt = time.Time{}
	c.mu.Unlock()
}

func (c *Client) cachedZones() []model.Zone {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.zones == nil || time.Since(c.zonesAt) > c.zoneTTL {
		return nil
	}
	out := make([]model.Zone, len(c.zones))
	copy(out, c.zones)
	return out
}

func (c *Client) storeZones(zones []model.Zone) {
	c.mu.Lock()
	c.zones = zones
	c.zonesAt = time.Now()
	c.mu.Unlock()
}

// MatchZone picks the longest zone suffix that covers hostname.
func MatchZone(zones []model.Zone, hostname string) (model.Zone, bool) {
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	best := model.Zone{}
	found := false
	for _, zone := range zones {
		name := strings.ToLower(zone.Name)
		if host == name || strings.HasSuffix(host, "."+name) {
			if !found || len(name) > len(best.Name) {
				best = zone
				found = true
			}
		}
	}
	return best, found
}
