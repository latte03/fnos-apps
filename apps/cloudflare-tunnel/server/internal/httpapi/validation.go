package httpapi

import (
	"regexp"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

var (
	accountIDPattern   = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	tunnelNamePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	hostnamePattern    = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
	httpStatusPattern  = regexp.MustCompile(`^http_status:[1-5][0-9]{2}$`)
	tokenWhitespaceRe  = regexp.MustCompile(`\s`)
	maxRoutesPerTunnel = 200
)

// validService accepts the cloudflared origin forms this app exposes.
func validService(service string) bool {
	value := strings.TrimSpace(service)
	if value == "" || tokenWhitespaceRe.MatchString(value) {
		return false
	}
	if httpStatusPattern.MatchString(value) {
		return true
	}
	for _, prefix := range []string{"http://", "https://", "tcp://", "unix:"} {
		if strings.HasPrefix(value, prefix) && len(value) > len(prefix) {
			return true
		}
	}
	return false
}

// normaliseRoutes validates and normalises the editable ingress rules.
func normaliseRoutes(routes []model.Route, catchAll string) ([]model.Route, string, error) {
	if len(routes) > maxRoutesPerTunnel {
		return nil, "", model.Errorf(400, "validation_failed", "at most %d routes are allowed", maxRoutesPerTunnel)
	}
	out := make([]model.Route, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		host := strings.ToLower(strings.TrimSpace(route.Hostname))
		if host == "" {
			return nil, "", model.NewError(400, "validation_failed", "route hostname is required")
		}
		if !hostnamePattern.MatchString(host) {
			return nil, "", model.Errorf(400, "validation_failed", "hostname is invalid: %s", host)
		}
		service := strings.TrimSpace(route.Service)
		if !validService(service) {
			return nil, "", model.Errorf(400, "validation_failed", "service is not a supported origin for %s", host)
		}
		path := strings.TrimSpace(route.Path)
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") || tokenWhitespaceRe.MatchString(path) {
			return nil, "", model.Errorf(400, "validation_failed", "path must start with / for %s", host)
		}
		key := host + " " + path
		if _, dup := seen[key]; dup {
			return nil, "", model.Errorf(400, "validation_failed", "duplicate route for %s%s", host, path)
		}
		seen[key] = struct{}{}
		clean := model.Route{Hostname: host, Path: path, Service: service}
		if len(route.OriginRequest) > 0 {
			clean.OriginRequest = route.OriginRequest
		}
		out = append(out, clean)
	}
	catch := strings.TrimSpace(catchAll)
	if catch == "" {
		catch = model.DefaultCatchAll
	}
	if !validService(catch) {
		return nil, "", model.NewError(400, "validation_failed", "catchAllService is not a supported origin")
	}
	return out, catch, nil
}
