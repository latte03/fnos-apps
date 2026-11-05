package model

// IngressRoutes splits a stored cloudflared configuration into the editable
// routes plus the trailing catch-all service. Non-hostname rules that are not
// last are dropped silently; the last service-only rule is the catch-all.
func IngressRoutes(cfg map[string]any) ([]Route, string, int) {
	routes := make([]Route, 0, 4)
	catchAll := ""
	raw := 0
	if cfg == nil {
		return routes, DefaultCatchAll, 0
	}
	ing, _ := cfg["ingress"].([]any)
	raw = len(ing)
	for i, item := range ing {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		service, _ := m["service"].(string)
		host, _ := m["hostname"].(string)
		path, _ := m["path"].(string)
		if host == "" {
			if service != "" && i == len(ing)-1 {
				catchAll = service
			}
			continue
		}
		if path == "" {
			path = "/"
		}
		r := Route{Hostname: host, Path: path, Service: service}
		if or, ok := m["origin_request"].(map[string]any); ok && len(or) > 0 {
			r.OriginRequest = or
		}
		routes = append(routes, r)
	}
	if catchAll == "" {
		catchAll = DefaultCatchAll
	}
	return routes, catchAll, raw
}

// DefaultCatchAll is used when the stored configuration has no catch-all rule.
const DefaultCatchAll = "http_status:404"

// WithIngress returns a copy of cfg with ingress replaced. Every other key of
// cfg (warp-routing, originRequest, ...) is preserved untouched.
func WithIngress(cfg map[string]any, routes []Route, catchAll string) map[string]any {
	out := make(map[string]any, len(cfg)+1)
	for k, v := range cfg {
		out[k] = v
	}
	ing := make([]any, 0, len(routes)+1)
	for _, r := range routes {
		m := map[string]any{"service": r.Service}
		if r.Hostname != "" {
			m["hostname"] = r.Hostname
		}
		if r.Path != "" && r.Path != "/" {
			m["path"] = r.Path
		}
		if len(r.OriginRequest) > 0 {
			m["origin_request"] = r.OriginRequest
		}
		ing = append(ing, m)
	}
	ing = append(ing, map[string]any{"service": catchAll})
	out["ingress"] = ing
	return out
}
