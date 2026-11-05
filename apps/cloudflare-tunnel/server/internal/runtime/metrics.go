package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// Metric names emitted by cloudflared's Prometheus endpoint.
const (
	metricHaConnections = "cloudflared_tunnel_ha_connections"
	metricRequests      = "cloudflared_tunnel_total_requests"
	metricRequestsAlt   = "cloudflared_tunnel_request_total"
	metricErrors        = "cloudflared_tunnel_request_errors"
	metricRTT           = "cloudflared_tunnel_rtt"
	metricRTTAlt        = "cloudflared_tunnel_rtt_seconds"
)

// ParseMetrics reads the subset of the Prometheus exposition we need. Values
// that are absent stay nil so the UI can render "—" instead of a fake zero.
func ParseMetrics(body string, now time.Time) model.RuntimeMetrics {
	out := model.RuntimeMetrics{SampledAt: model.FormatTS(now)}
	var requests, errorsCount, connections int64
	var haveRequests, haveErrors, haveConnections bool
	var rtt float64
	var haveRTT bool

	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		if i := strings.IndexByte(name, '{'); i >= 0 {
			name = name[:i]
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch name {
		case metricHaConnections:
			connections += int64(value)
			haveConnections = true
		case metricRequests, metricRequestsAlt:
			requests += int64(value)
			haveRequests = true
		case metricErrors:
			errorsCount += int64(value)
			haveErrors = true
		case metricRTT, metricRTTAlt:
			if value >= 0 && (!haveRTT || value > rtt) {
				rtt = value
				haveRTT = true
			}
		}
	}
	if haveRequests {
		out.Requests = &requests
	}
	if haveErrors {
		out.Errors = &errorsCount
	}
	if haveConnections {
		out.Connections = &connections
	}
	if haveRTT {
		// cloudflared exposes the round trip time in seconds.
		ms := rtt * 1000
		out.RTTMs = &ms
	}
	return out
}

// HTTPProber polls the local cloudflared readiness and metrics endpoints.
type HTTPProber struct {
	Port   int
	Client *http.Client
}

// NewHTTPProber builds a prober for the loopback metrics port.
func NewHTTPProber(port int) *HTTPProber {
	return &HTTPProber{Port: port, Client: &http.Client{Timeout: 2 * time.Second}}
}

func (p *HTTPProber) httpClient() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 2 * time.Second}
}

// Ready reports whether /ready answers 200.
func (p *HTTPProber) Ready(ctx context.Context, _ time.Time) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/ready", p.Port), nil)
	if err != nil {
		return false
	}
	resp, err := p.httpClient().Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// Metrics scrapes /metrics.
func (p *HTTPProber) Metrics(ctx context.Context, _ time.Time) (model.RuntimeMetrics, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/metrics", p.Port), nil)
	if err != nil {
		return model.RuntimeMetrics{}, false
	}
	resp, err := p.httpClient().Do(req)
	if err != nil {
		return model.RuntimeMetrics{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.RuntimeMetrics{}, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return model.RuntimeMetrics{}, false
	}
	parsed := ParseMetrics(string(body), time.Now())
	if parsed.Requests == nil && parsed.Errors == nil && parsed.Connections == nil && parsed.RTTMs == nil {
		return parsed, false
	}
	return parsed, true
}
