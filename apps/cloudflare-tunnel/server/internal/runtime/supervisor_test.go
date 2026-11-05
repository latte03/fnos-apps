package runtime

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

const testTokenValue = "eyJhIjoiZGVtby10b2tlbiJ9"

// fakeChild is a controllable cloudflared stand-in.
type fakeChild struct {
	pr              *io.PipeReader
	pw              *io.PipeWriter
	done            chan struct{}
	once            sync.Once
	waitErr         error
	ignoreTerminate bool
	termCalls       int32
	killCalls       int32
}

func newFakeChild() *fakeChild {
	pr, pw := io.Pipe()
	return &fakeChild{pr: pr, pw: pw, done: make(chan struct{})}
}

func (c *fakeChild) Output() io.Reader { return c.pr }

func (c *fakeChild) Wait() error {
	<-c.done
	return c.waitErr
}

func (c *fakeChild) Terminate() error {
	atomic.AddInt32(&c.termCalls, 1)
	if !c.ignoreTerminate {
		c.exit()
	}
	return nil
}

func (c *fakeChild) Kill() error {
	atomic.AddInt32(&c.killCalls, 1)
	c.exit()
	return nil
}

func (c *fakeChild) Close() error { return c.pr.Close() }

func (c *fakeChild) Pid() int { return 4242 }

func (c *fakeChild) exit() {
	c.once.Do(func() {
		close(c.done)
		_ = c.pw.Close()
	})
}

func (c *fakeChild) emit(line string) {
	_, _ = c.pw.Write([]byte(line + "\n"))
}

// fakeFactory records every spec and hands out scripted children.
type fakeFactory struct {
	mu       sync.Mutex
	specs    []Spec
	children []*fakeChild
	make     func() *fakeChild
	startErr error
}

func (f *fakeFactory) Start(_ context.Context, spec Spec) (Child, error) {
	f.mu.Lock()
	f.specs = append(f.specs, spec)
	startErr := f.startErr
	f.mu.Unlock()
	if startErr != nil {
		return nil, startErr
	}
	child := f.make()
	f.mu.Lock()
	f.children = append(f.children, child)
	f.mu.Unlock()
	return child, nil
}

func (f *fakeFactory) lastSpec() Spec {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.specs) == 0 {
		return Spec{}
	}
	return f.specs[len(f.specs)-1]
}

func (f *fakeFactory) startCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.specs)
}

func (f *fakeFactory) lastChild() *fakeChild {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.children) == 0 {
		return nil
	}
	return f.children[len(f.children)-1]
}

// fakeProber becomes ready after a delay and returns stable metrics.
type fakeProber struct {
	readyAfter time.Duration
}

func (p *fakeProber) Ready(_ context.Context, startedAt time.Time) bool {
	return time.Since(startedAt) >= p.readyAfter
}

func (p *fakeProber) Metrics(_ context.Context, startedAt time.Time) (model.RuntimeMetrics, bool) {
	if time.Since(startedAt) < p.readyAfter {
		return model.RuntimeMetrics{}, false
	}
	requests := int64(10)
	connections := int64(2)
	return model.RuntimeMetrics{Requests: &requests, Connections: &connections, SampledAt: model.NowRFC3339()}, true
}

// sinkRecorder captures everything the supervisor publishes.
type sinkRecorder struct {
	mu       sync.Mutex
	statuses []model.Runtime
	logs     []model.LogLine
	metrics  []model.RuntimeMetrics
}

func (s *sinkRecorder) Status(rt model.Runtime) {
	s.mu.Lock()
	s.statuses = append(s.statuses, rt)
	s.mu.Unlock()
}

func (s *sinkRecorder) Log(line model.LogLine) {
	s.mu.Lock()
	s.logs = append(s.logs, line)
	s.mu.Unlock()
}

func (s *sinkRecorder) Metrics(m model.RuntimeMetrics) {
	s.mu.Lock()
	s.metrics = append(s.metrics, m)
	s.mu.Unlock()
}

func (s *sinkRecorder) hasState(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rt := range s.statuses {
		if rt.State == state {
			return true
		}
	}
	return false
}

func (s *sinkRecorder) lastError() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.statuses) - 1; i >= 0; i-- {
		if s.statuses[i].LastError != "" {
			return s.statuses[i].LastError
		}
	}
	return ""
}

func (s *sinkRecorder) logCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.logs)
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func newTestSupervisor(t *testing.T, factory Factory, prober Prober, sink Sink, mutate func(*Options)) *Supervisor {
	t.Helper()
	opts := Options{
		Binary:          "cloudflared",
		MetricsPort:     20241,
		Protocol:        "quic",
		EdgeIPVersion:   "auto",
		AutoReconnect:   true,
		MaxReconnect:    3,
		MaxLogLines:     100,
		Factory:         factory,
		Prober:          prober,
		Backoff:         []time.Duration{5 * time.Millisecond},
		ReadyInterval:   10 * time.Millisecond,
		MetricsInterval: 20 * time.Millisecond,
		StopGrace:       time.Second,
	}
	if mutate != nil {
		mutate(&opts)
	}
	return New(opts, sink)
}

func TestSupervisorConnectFlow(t *testing.T) {
	factory := &fakeFactory{make: func() *fakeChild {
		child := newFakeChild()
		go func() {
			child.emit(`{"level":"info","time":"2026-11-05T12:00:00Z","message":"Starting metrics server"}`)
			child.emit(`{"level":"info","time":"2026-11-05T12:00:01Z","message":"Registered tunnel connection","connIndex":0,"location":"HKG"}`)
		}()
		return child
	}}
	prober := &fakeProber{readyAfter: 20 * time.Millisecond}
	sink := &sinkRecorder{}
	sup := newTestSupervisor(t, factory, prober, sink, nil)

	if got := sup.State().State; got != model.StateDisconnected {
		t.Fatalf("initial state = %q", got)
	}
	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	eventually(t, 2*time.Second, func() bool {
		return sup.State().State == model.StateConnected
	}, "connected state")

	state := sup.State()
	if state.TunnelID != "tun-1" || state.TunnelName != "fnos-main" || state.ConnectedAt == "" {
		t.Fatalf("unexpected runtime: %+v", state)
	}
	if state.Protocol != "quic" {
		t.Fatalf("protocol = %q", state.Protocol)
	}
	eventually(t, 2*time.Second, func() bool { return sup.State().EdgeLocation == "HKG" }, "edge location")
	eventually(t, 2*time.Second, func() bool { return sup.State().Connections == 2 }, "connections from metrics")
	if sink.logCount() == 0 {
		t.Fatal("no log lines were published")
	}

	spec := factory.lastSpec()
	if !slices.Contains(spec.Env, "TUNNEL_TOKEN="+testTokenValue) {
		t.Fatalf("token missing from env: %v", spec.Env)
	}
	for _, arg := range spec.Args {
		if strings.Contains(arg, testTokenValue) {
			t.Fatalf("token leaked into argv: %q", spec.Args)
		}
	}
	wantArgs := []string{
		"tunnel", "--no-autoupdate", "--output", "json",
		"--metrics", "127.0.0.1:20241",
		"--loglevel", "info", "run",
		"--protocol", "quic",
	}
	if !slices.Equal(spec.Args, wantArgs) {
		t.Fatalf("args = %v, want %v", spec.Args, wantArgs)
	}

	if err := sup.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := sup.State().State; got != model.StateDisconnected {
		t.Fatalf("state after disconnect = %q", got)
	}
	if child := factory.lastChild(); child == nil || atomic.LoadInt32(&child.termCalls) == 0 {
		t.Fatal("child was not terminated")
	}
	if sup.State().ConnectedAt != "" || sup.State().Connections != 0 {
		t.Fatalf("runtime not reset: %+v", sup.State())
	}
}

func TestSupervisorUnexpectedExitReconnects(t *testing.T) {
	factory := &fakeFactory{make: func() *fakeChild {
		child := newFakeChild()
		child.waitErr = errors.New("boom")
		go func() {
			time.Sleep(20 * time.Millisecond)
			child.exit()
		}()
		return child
	}}
	sink := &sinkRecorder{}
	sup := newTestSupervisor(t, factory, &fakeProber{readyAfter: time.Second}, sink, nil)

	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	eventually(t, 2*time.Second, func() bool { return sink.hasState(model.StateError) }, "error state")
	eventually(t, 2*time.Second, func() bool { return sink.hasState(model.StateReconnecting) }, "reconnecting state")
	eventually(t, 2*time.Second, func() bool { return factory.startCount() >= 2 }, "a restart")
	if got := sink.lastError(); !strings.Contains(got, "boom") {
		t.Fatalf("last error = %q", got)
	}
	if got := sup.State().ReconnectAttempt; got < 1 {
		t.Fatalf("reconnectAttempt = %d", got)
	}
	_ = sup.Disconnect()
}

func TestSupervisorStopEscalatesToKill(t *testing.T) {
	factory := &fakeFactory{make: func() *fakeChild {
		child := newFakeChild()
		child.ignoreTerminate = true
		return child
	}}
	sink := &sinkRecorder{}
	sup := newTestSupervisor(t, factory, &fakeProber{readyAfter: 5 * time.Millisecond}, sink, func(o *Options) {
		o.StopGrace = 30 * time.Millisecond
	})
	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	eventually(t, 2*time.Second, func() bool { return sup.State().State == model.StateConnected }, "connected state")
	if err := sup.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := sup.State().State; got != model.StateDisconnected {
		t.Fatalf("state = %q", got)
	}
	child := factory.lastChild()
	if child == nil || atomic.LoadInt32(&child.killCalls) == 0 {
		t.Fatal("SIGKILL escalation was not used")
	}
	if child != nil && atomic.LoadInt32(&child.termCalls) == 0 {
		t.Fatal("SIGTERM was not attempted first")
	}
}

func TestSupervisorConnectIsIdempotent(t *testing.T) {
	factory := &fakeFactory{make: func() *fakeChild { return newFakeChild() }}
	sup := newTestSupervisor(t, factory, &fakeProber{readyAfter: 5 * time.Millisecond}, &sinkRecorder{}, nil)
	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	eventually(t, 2*time.Second, func() bool { return sup.State().State == model.StateConnected }, "connected state")
	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if factory.startCount() != 1 {
		t.Fatalf("connect restarted the child %d times", factory.startCount())
	}
	_ = sup.Disconnect()
}

func TestSupervisorStartFailureRetries(t *testing.T) {
	factory := &fakeFactory{startErr: errors.New("no such file"), make: func() *fakeChild { return newFakeChild() }}
	sink := &sinkRecorder{}
	sup := newTestSupervisor(t, factory, &fakeProber{readyAfter: time.Millisecond}, sink, func(o *Options) {
		o.MaxReconnect = 2
	})
	if err := sup.Connect("tun-1", "fnos-main", testTokenValue); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	eventually(t, 2*time.Second, func() bool { return factory.startCount() >= 3 }, "retry attempts")
	eventually(t, 2*time.Second, func() bool {
		return strings.Contains(sink.lastError(), "no such file")
	}, "start error surfaced")
	if got := sup.State().State; got != model.StateError {
		t.Fatalf("state = %q, want error", got)
	}
}

func TestSpecOmitsAutoFlags(t *testing.T) {
	sup := New(Options{Binary: "cloudflared", MetricsPort: 20241, Protocol: "auto", EdgeIPVersion: "auto"}, &sinkRecorder{})
	spec := sup.spec("tok")
	want := []string{
		"tunnel", "--no-autoupdate", "--output", "json",
		"--metrics", "127.0.0.1:20241",
		"--loglevel", "info", "run",
	}
	if !slices.Equal(spec.Args, want) {
		t.Fatalf("args = %v, want %v", spec.Args, want)
	}
	if !slices.Contains(spec.Env, "TUNNEL_TOKEN=tok") {
		t.Fatalf("env = %v", spec.Env)
	}
}

func TestSpecIncludesExplicitFlags(t *testing.T) {
	sup := New(Options{Binary: "cloudflared", MetricsPort: 20242, Protocol: "http2", EdgeIPVersion: "4"}, &sinkRecorder{})
	spec := sup.spec("tok")
	joined := strings.Join(spec.Args, " ")
	for _, want := range []string{"--protocol http2", "--edge-ip-version 4", "--metrics 127.0.0.1:20242"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}
