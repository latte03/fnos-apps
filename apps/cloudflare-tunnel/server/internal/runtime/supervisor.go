package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// Sink receives live supervisor updates. Implementations must not block.
type Sink interface {
	Status(model.Runtime)
	Log(model.LogLine)
	Metrics(model.RuntimeMetrics)
}

// DefaultBackoff is the reconnect schedule mandated by the contract.
var DefaultBackoff = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	20 * time.Second,
	40 * time.Second,
	60 * time.Second,
}

// Options configures a Supervisor. Zero values fall back to production
// defaults; tests inject small intervals and fake factories.
type Options struct {
	Binary             string
	MetricsPort        int
	CloudflaredVersion string

	Protocol      string
	EdgeIPVersion string
	AutoReconnect bool
	MaxReconnect  int
	MaxLogLines   int

	// Mirror receives every raw cloudflared line (logs/cloudflared.log).
	Mirror io.Writer

	Factory Factory
	Prober  Prober

	Backoff         []time.Duration
	ReadyInterval   time.Duration
	MetricsInterval time.Duration
	StopGrace       time.Duration
}

// Supervisor owns at most one cloudflared child at a time.
type Supervisor struct {
	mu   sync.Mutex // guards state, metrics, gen, child, loop*, mutable options
	opMu sync.Mutex // serialises Connect/Disconnect

	opts Options
	sink Sink
	ring *Ring

	state   model.Runtime
	metrics model.RuntimeMetrics

	protocol      string
	edgeIPVersion string
	autoReconnect bool
	maxReconnect  int

	gen      int
	child    Child
	loopStop chan struct{}
	loopDone chan struct{}
	stopping bool
}

// New builds a supervisor. Nothing starts until Connect is called.
func New(opts Options, sink Sink) *Supervisor {
	if opts.Factory == nil {
		opts.Factory = ExecFactory{}
	}
	if opts.Prober == nil {
		opts.Prober = NewHTTPProber(opts.MetricsPort)
	}
	if len(opts.Backoff) == 0 {
		opts.Backoff = append([]time.Duration(nil), DefaultBackoff...)
	}
	if opts.ReadyInterval <= 0 {
		opts.ReadyInterval = time.Second
	}
	if opts.MetricsInterval <= 0 {
		opts.MetricsInterval = 5 * time.Second
	}
	if opts.StopGrace <= 0 {
		opts.StopGrace = 5 * time.Second
	}
	if opts.MaxLogLines <= 0 {
		opts.MaxLogLines = DefaultRingCapacity
	}
	protocol := opts.Protocol
	if protocol == "" {
		protocol = "quic"
	}
	edge := opts.EdgeIPVersion
	if edge == "" {
		edge = "auto"
	}
	s := &Supervisor{
		opts:          opts,
		sink:          sink,
		ring:          NewRing(opts.MaxLogLines),
		protocol:      protocol,
		edgeIPVersion: edge,
		autoReconnect: opts.AutoReconnect,
		maxReconnect:  opts.MaxReconnect,
		state: model.Runtime{
			State:              model.StateDisconnected,
			Protocol:           protocol,
			CloudflaredVersion: opts.CloudflaredVersion,
		},
	}
	return s
}

// SetPolicy updates the reconnect policy (settings.json can change at runtime).
func (s *Supervisor) SetPolicy(autoReconnect bool, maxReconnect int) {
	s.mu.Lock()
	s.autoReconnect = autoReconnect
	s.maxReconnect = maxReconnect
	s.mu.Unlock()
}

// SetTunnelOptions updates the connector flags used for the next attempt.
func (s *Supervisor) SetTunnelOptions(protocol, edgeIPVersion string) {
	s.mu.Lock()
	if protocol != "" {
		s.protocol = protocol
	}
	if edgeIPVersion != "" {
		s.edgeIPVersion = edgeIPVersion
	}
	if !model.IsLiveState(s.state.State) {
		s.state.Protocol = s.protocol
	}
	s.mu.Unlock()
}

// SetLogCapacity resizes the in-memory log buffer.
func (s *Supervisor) SetLogCapacity(n int) { s.ring.SetCapacity(n) }

// State returns the current runtime snapshot with uptime filled in.
func (s *Supervisor) State() model.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.WithUptime()
}

// Metrics returns the last metrics sample.
func (s *Supervisor) Metrics() model.RuntimeMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.metrics
}

// Logs queries the ring buffer.
func (s *Supervisor) Logs(q Query) model.LogPage { return s.ring.Query(q) }

// ClearLogs empties the ring buffer.
func (s *Supervisor) ClearLogs() { s.ring.Clear() }

// Text renders the retained log for download.
func (s *Supervisor) Text() string { return s.ring.Text() }

// Ring exposes the buffer (used by the SSE hello payload).
func (s *Supervisor) Ring() *Ring { return s.ring }

// Connect starts (or restarts) the tunnel. It is idempotent for the tunnel that
// is already live.
func (s *Supervisor) Connect(tunnelID, tunnelName, token string) error {
	if strings.TrimSpace(tunnelID) == "" {
		return model.NewError(400, "validation_failed", "tunnelId is required")
	}
	if strings.TrimSpace(token) == "" {
		return model.NewError(400, "validation_failed", "tunnel token is unavailable")
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	if s.state.TunnelID == tunnelID && model.IsLiveState(s.state.State) && !s.stopping {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.stop()

	s.mu.Lock()
	s.gen++
	gen := s.gen
	s.stopping = false
	s.state = model.Runtime{
		State:              model.StateConnecting,
		TunnelID:           tunnelID,
		TunnelName:         tunnelName,
		Protocol:           s.protocol,
		CloudflaredVersion: s.opts.CloudflaredVersion,
	}
	s.metrics = model.RuntimeMetrics{}
	stop := make(chan struct{})
	done := make(chan struct{})
	s.loopStop = stop
	s.loopDone = done
	snap := s.state.WithUptime()
	s.mu.Unlock()

	s.sink.Status(snap)
	go s.loop(gen, stop, done, tunnelID, tunnelName, token)
	return nil
}

// Disconnect stops the child (SIGTERM, then SIGKILL after the grace period).
func (s *Supervisor) Disconnect() error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.stop()
	return nil
}

// Shutdown stops the child, honouring ctx.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		_ = s.Disconnect()
		close(ch)
	}()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// stop terminates the current attempt and leaves the supervisor disconnected.
func (s *Supervisor) stop() {
	s.mu.Lock()
	done := s.loopDone
	if done == nil {
		wasLive := s.state.State != model.StateDisconnected
		s.state.State = model.StateDisconnected
		s.state.ConnectedAt = ""
		s.state.Connections = 0
		s.state.ReconnectAttempt = 0
		snap := s.state.WithUptime()
		s.mu.Unlock()
		if wasLive {
			s.sink.Status(snap)
		}
		return
	}
	s.stopping = true
	child := s.child
	stop := s.loopStop
	s.state.State = model.StateStopping
	stoppingSnap := s.state.WithUptime()
	s.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if child != nil {
		_ = child.Terminate()
	}
	s.sink.Status(stoppingSnap)

	select {
	case <-done:
	case <-time.After(s.opts.StopGrace):
		s.mu.Lock()
		child = s.child
		s.mu.Unlock()
		if child != nil {
			_ = child.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}

	s.mu.Lock()
	s.loopStop = nil
	s.loopDone = nil
	s.child = nil
	s.stopping = false
	s.state.State = model.StateDisconnected
	s.state.ConnectedAt = ""
	s.state.Connections = 0
	s.state.ReconnectAttempt = 0
	s.state.EdgeLocation = ""
	final := s.state.WithUptime()
	s.mu.Unlock()
	s.sink.Status(final)
}

func (s *Supervisor) loop(gen int, stop, done chan struct{}, tunnelID, tunnelName, token string) {
	defer close(done)
	attempt := 0
	for {
		if attempt > 0 {
			s.setState(gen, model.StateReconnecting, func(rt *model.Runtime) {
				rt.ReconnectAttempt = attempt
			})
			if !waitFor(stop, s.backoffDelay(attempt)) {
				return
			}
			s.setState(gen, model.StateConnecting, func(rt *model.Runtime) {
				rt.ReconnectAttempt = attempt
			})
		}
		child, err := s.opts.Factory.Start(context.Background(), s.spec(token))
		if err != nil {
			s.setState(gen, model.StateError, func(rt *model.Runtime) {
				rt.LastError = "cannot start cloudflared: " + err.Error()
				rt.Connections = 0
				rt.ConnectedAt = ""
			})
			if !s.retryAllowed(attempt) {
				return
			}
			attempt++
			continue
		}

		s.mu.Lock()
		stopping := s.stopping
		if s.gen == gen {
			s.child = child
		}
		s.mu.Unlock()
		if stopping {
			_ = child.Terminate()
		}

		startedAt := time.Now()
		ctx, cancel := context.WithCancel(context.Background())
		pumpDone := make(chan struct{})
		go s.pump(gen, child, pumpDone)
		go s.pollReady(gen, ctx, startedAt, attempt)
		go s.pollMetrics(gen, ctx, startedAt)

		waitErr := child.Wait()
		cancel()
		select {
		case <-pumpDone:
		case <-time.After(time.Second):
		}
		_ = child.Close()
		<-pumpDone

		s.mu.Lock()
		if s.gen == gen {
			s.child = nil
		}
		stopping = s.stopping
		s.mu.Unlock()
		if stopping {
			// stop() publishes the final disconnected state.
			return
		}

		message := "cloudflared exited unexpectedly"
		if waitErr != nil {
			message = "cloudflared exited: " + waitErr.Error()
		}
		s.setState(gen, model.StateError, func(rt *model.Runtime) {
			rt.LastError = message
			rt.Connections = 0
			rt.ConnectedAt = ""
			rt.EdgeLocation = ""
		})
		if !s.retryAllowed(attempt) {
			return
		}
		attempt++
	}
}

func (s *Supervisor) pump(gen int, child Child, done chan struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(child.Output())
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		raw := scanner.Text()
		line, ok := ParseLine(raw, time.Now())
		if !ok {
			continue
		}
		line = s.ring.Append(line)
		if s.opts.Mirror != nil {
			_, _ = io.WriteString(s.opts.Mirror, strings.TrimRight(raw, "\r")+"\n")
		}
		s.sink.Log(line)
		if strings.Contains(line.Message, "Registered tunnel connection") {
			if location, ok := FieldString(line.Fields, "location"); ok && location != "" {
				s.updateState(gen, func(rt *model.Runtime) { rt.EdgeLocation = location })
			}
		}
	}
}

func (s *Supervisor) pollReady(gen int, ctx context.Context, startedAt time.Time, attempt int) {
	ticker := time.NewTicker(s.opts.ReadyInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if !s.opts.Prober.Ready(ctx, startedAt) {
			continue
		}
		s.setState(gen, model.StateConnected, func(rt *model.Runtime) {
			rt.ConnectedAt = model.NowRFC3339()
			rt.ReconnectAttempt = attempt
			rt.LastError = ""
		})
		// Refresh metrics right away so the UI does not wait for the next tick.
		s.sampleMetrics(gen, ctx, startedAt)
		return
	}
}

func (s *Supervisor) pollMetrics(gen int, ctx context.Context, startedAt time.Time) {
	s.sampleMetrics(gen, ctx, startedAt)
	ticker := time.NewTicker(s.opts.MetricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleMetrics(gen, ctx, startedAt)
		}
	}
}

func (s *Supervisor) sampleMetrics(gen int, ctx context.Context, startedAt time.Time) {
	sample, ok := s.opts.Prober.Metrics(ctx, startedAt)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	s.metrics = sample
	connectionsChanged := false
	if sample.Connections != nil && int(*sample.Connections) != s.state.Connections {
		s.state.Connections = int(*sample.Connections)
		connectionsChanged = true
	}
	var status model.Runtime
	if connectionsChanged {
		status = s.state.WithUptime()
	}
	s.mu.Unlock()
	if connectionsChanged {
		s.sink.Status(status)
	}
	s.sink.Metrics(sample)
}

func (s *Supervisor) setState(gen int, state string, mutate func(*model.Runtime)) {
	s.updateState(gen, func(rt *model.Runtime) {
		rt.State = state
		if mutate != nil {
			mutate(rt)
		}
	})
}

func (s *Supervisor) updateState(gen int, mutate func(*model.Runtime)) {
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	mutate(&s.state)
	snap := s.state.WithUptime()
	s.mu.Unlock()
	s.sink.Status(snap)
}

func (s *Supervisor) retryAllowed(attempt int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.autoReconnect && s.maxReconnect > 0 && attempt < s.maxReconnect
}

func (s *Supervisor) backoffDelay(attempt int) time.Duration {
	idx := attempt - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.opts.Backoff) {
		idx = len(s.opts.Backoff) - 1
	}
	return s.opts.Backoff[idx]
}

func (s *Supervisor) spec(token string) Spec {
	s.mu.Lock()
	protocol := s.protocol
	edge := s.edgeIPVersion
	s.mu.Unlock()

	// cloudflared 2026.10.7 parses --no-autoupdate, --output, --metrics,
	// --edge-ip-version and --loglevel on the `tunnel` command, so they must
	// precede `run`. `--logformat` does not exist; JSON logs use --output
	// json. Only --protocol is a run-level flag.
	args := []string{"tunnel", "--no-autoupdate"}
	if edge != "" && edge != "auto" {
		args = append(args, "--edge-ip-version", edge)
	}
	args = append(args,
		"--output", "json",
		"--metrics", fmt.Sprintf("127.0.0.1:%d", s.opts.MetricsPort),
		"--loglevel", "info",
		"run",
	)
	if protocol != "" && protocol != "auto" {
		args = append(args, "--protocol", protocol)
	}
	return Spec{Path: s.opts.Binary, Args: args, Env: childEnv(token)}
}

// childEnv builds a small, predictable environment. The tunnel token is only
// ever passed here.
func childEnv(token string) []string {
	keys := []string{
		"PATH", "HOME", "LANG", "LC_ALL", "TZ",
		"SSL_CERT_FILE", "SSL_CERT_DIR",
		"TMPDIR", "TEMP", "TMP",
		"SystemRoot", "USERPROFILE", "ComSpec",
	}
	env := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			env = append(env, key+"="+value)
		}
	}
	env = append(env, "TUNNEL_TOKEN="+token)
	return env
}

func waitFor(stop <-chan struct{}, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}
