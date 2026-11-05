// Package model holds the DTOs shared by the HTTP API, the Cloudflare client,
// the cloudflared supervisor and demo mode. JSON tags are the wire contract.
package model

import (
	"fmt"
	"strings"
	"time"
)

// Tunnel statuses surfaced by the API.
const (
	TunnelHealthy  = "healthy"
	TunnelDegraded = "degraded"
	TunnelInactive = "inactive"
	TunnelError    = "error"
)

// Runtime states.
const (
	StateDisconnected = "disconnected"
	StateConnecting   = "connecting"
	StateConnected    = "connected"
	StateReconnecting = "reconnecting"
	StateError        = "error"
	StateStopping     = "stopping"
)

// Tunnel is one remote (token based) tunnel.
type Tunnel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	Connections int    `json:"connections"`
	ConfigSrc   string `json:"configSrc"`
	LocalActive bool   `json:"localActive"`
}

// Route is one ingress rule owned by the user (the catch-all lives separately).
type Route struct {
	Hostname      string         `json:"hostname"`
	Path          string         `json:"path"`
	Service       string         `json:"service"`
	OriginRequest map[string]any `json:"originRequest,omitempty"`
}

// Routes is the ingress DTO returned by GET/PUT /tunnels/{id}/routes.
type Routes struct {
	Routes          []Route `json:"routes"`
	CatchAllService string  `json:"catchAllService"`
	RawIngressCount int     `json:"rawIngressCount"`
}

// Zone is a Cloudflare zone the token can see.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DNSResult describes the outcome of a CNAME upsert.
type DNSResult struct {
	RecordID string `json:"recordId"`
	ZoneID   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Proxied  bool   `json:"proxied"`
	Action   string `json:"action"`
}

// Runtime is the supervisor snapshot.
type Runtime struct {
	State              string `json:"state"`
	TunnelID           string `json:"tunnelId"`
	TunnelName         string `json:"tunnelName"`
	ConnectedAt        string `json:"connectedAt"`
	UptimeSec          int64  `json:"uptimeSec"`
	Protocol           string `json:"protocol"`
	EdgeLocation       string `json:"edgeLocation"`
	Connections        int    `json:"connections"`
	ReconnectAttempt   int    `json:"reconnectAttempt"`
	LastError          string `json:"lastError"`
	CloudflaredVersion string `json:"cloudflaredVersion"`
}

// RuntimeMetrics is the /runtime/metrics DTO. Missing values stay nil.
type RuntimeMetrics struct {
	Requests    *int64   `json:"requests"`
	Errors      *int64   `json:"errors"`
	Connections *int64   `json:"connections"`
	RTTMs       *float64 `json:"rttMs"`
	SampledAt   string   `json:"sampledAt"`
}

// MetricValues is the compact metric block embedded in /overview.
type MetricValues struct {
	Requests    *int64   `json:"requests"`
	Errors      *int64   `json:"errors"`
	Connections *int64   `json:"connections"`
	RTTMs       *float64 `json:"rttMs"`
}

// Values strips the sample timestamp for the overview block.
func (m RuntimeMetrics) Values() MetricValues {
	return MetricValues{Requests: m.Requests, Errors: m.Errors, Connections: m.Connections, RTTMs: m.RTTMs}
}

// LogLine is one parsed cloudflared log line.
type LogLine struct {
	Seq     int64          `json:"seq"`
	TS      string         `json:"ts"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields"`
}

// LogPage is the /logs response.
type LogPage struct {
	Lines   []LogLine `json:"lines"`
	NextSeq int64     `json:"nextSeq"`
	Dropped int64     `json:"dropped"`
}

// Activity is one entry of the activity feed.
type Activity struct {
	ID    string `json:"id"`
	TS    string `json:"ts"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Tone  string `json:"tone"`
}

// VerifyResult is the outcome of POST /settings/verify.
type VerifyResult struct {
	OK           bool   `json:"ok"`
	AccountID    string `json:"accountId"`
	TokenStatus  string `json:"tokenStatus"`
	TunnelAccess bool   `json:"tunnelAccess"`
	ZoneAccess   bool   `json:"zoneAccess"`
	Error        string `json:"error"`
}

// APIError is the error every layer returns for a failure that already has a
// contract status code and code string. Message must stay free of secrets.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError builds an APIError.
func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// Errorf builds an APIError with a formatted message.
func Errorf(status int, code, format string, args ...any) *APIError {
	return &APIError{Status: status, Code: code, Message: fmt.Sprintf(format, args...)}
}

// NowRFC3339 is the single timestamp format used on the wire.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// FormatTS renders t in the wire format.
func FormatTS(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// Uptime returns whole seconds since connectedAt, 0 when not connected.
func (r Runtime) Uptime() int64 {
	if r.ConnectedAt == "" || (r.State != StateConnected && r.State != StateReconnecting) {
		return 0
	}
	t, err := time.Parse(time.RFC3339, r.ConnectedAt)
	if err != nil {
		return 0
	}
	d := time.Since(t)
	if d < 0 {
		return 0
	}
	return int64(d.Seconds())
}

// WithUptime fills UptimeSec.
func (r Runtime) WithUptime() Runtime {
	r.UptimeSec = r.Uptime()
	return r
}

// IsLiveState reports whether the state means a child process is wanted.
func IsLiveState(state string) bool {
	switch strings.ToLower(state) {
	case StateConnecting, StateConnected, StateReconnecting, StateStopping:
		return true
	}
	return false
}
