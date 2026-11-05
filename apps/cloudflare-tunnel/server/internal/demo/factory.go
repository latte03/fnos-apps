package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
)

// Factory spawns a fake cloudflared whose stdout looks like the real JSON log.
type Factory struct{}

// Start implements runtime.Factory.
func (Factory) Start(context.Context, runtime.Spec) (runtime.Child, error) {
	return newFakeChild(), nil
}

// Prober fakes /ready and /metrics with rising counters.
type Prober struct{}

// Ready reports ready roughly one second after the attempt started, which
// gives the UI its "connecting → connected" transition.
func (Prober) Ready(_ context.Context, startedAt time.Time) bool {
	return time.Since(startedAt) >= 700*time.Millisecond
}

// Metrics synthesises plausible traffic numbers.
func (Prober) Metrics(_ context.Context, startedAt time.Time) (model.RuntimeMetrics, bool) {
	elapsed := time.Since(startedAt)
	if elapsed < time.Second {
		return model.RuntimeMetrics{}, false
	}
	secs := int64(elapsed / time.Second)
	requests := int64(12400) + secs*7
	errors := int64(3) + secs/240
	connections := int64(2)
	rtt := 38.0 + float64(secs%7)
	return model.RuntimeMetrics{
		Requests:    &requests,
		Errors:      &errors,
		Connections: &connections,
		RTTMs:       &rtt,
		SampledAt:   model.FormatTS(time.Now()),
	}, true
}

type fakeChild struct {
	pr     *io.PipeReader
	pw     *io.PipeWriter
	stop   chan struct{}
	done   chan struct{}
	closed sync.Once
}

func newFakeChild() *fakeChild {
	pr, pw := io.Pipe()
	c := &fakeChild{pr: pr, pw: pw, stop: make(chan struct{}), done: make(chan struct{})}
	go c.run()
	return c
}

type sample struct {
	level   string
	message string
	fields  map[string]any
}

var idleLogs = []sample{
	{"info", "Updated to new configuration", map[string]any{"version": 12}},
	{"info", "Serving dashboard", map[string]any{"url": "http://127.0.0.1:20241/ready"}},
	{"debug", "Heartbeat", map[string]any{"connIndex": 0}},
	{"info", "GET https://nas.example.com/ 200", map[string]any{"host": "nas.example.com", "status": 200, "duration": "18ms"}},
	{"info", "GET https://ai.example.com/api/chat 200", map[string]any{"host": "ai.example.com", "status": 200, "duration": "142ms"}},
	{"info", "POST https://ha.example.com/api/webhook 200", map[string]any{"host": "ha.example.com", "status": 200, "duration": "37ms"}},
	{"debug", "QUIC connection is stable", map[string]any{"connIndex": 1}},
	{"info", "GET https://pa.example.com/ 404", map[string]any{"host": "pa.example.com", "status": 404, "duration": "9ms"}},
}

func (c *fakeChild) run() {
	defer close(c.done)
	defer func() { _ = c.pw.Close() }()

	if !c.emit("info", "Starting metrics server", map[string]any{"addr": "127.0.0.1:20241"}) {
		return
	}
	if !c.emit("info", "Initial protocol quic", map[string]any{"protocol": "quic"}) {
		return
	}
	select {
	case <-time.After(600 * time.Millisecond):
	case <-c.stop:
		return
	}
	if !c.emit("info", "Registered tunnel connection", map[string]any{
		"connIndex": 0, "event": 0, "location": "HKG", "protocol": "quic", "ip": "198.41.200.13",
	}) {
		return
	}
	if !c.emit("info", "Registered tunnel connection", map[string]any{
		"connIndex": 1, "event": 0, "location": "HKG", "protocol": "quic", "ip": "198.41.200.73",
	}) {
		return
	}

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	for i := 1; ; i++ {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
		}
		entry := idleLogs[i%len(idleLogs)]
		if !c.emit(entry.level, entry.message, entry.fields) {
			return
		}
		if i%20 == 0 {
			if !c.emit("warn", "Connection terminated", map[string]any{"connIndex": 1, "error": "flush timeout"}) {
				return
			}
			if !c.emit("info", "Registered tunnel connection", map[string]any{
				"connIndex": 1, "event": i, "location": "HKG", "protocol": "quic", "ip": "198.41.200.73",
			}) {
				return
			}
		}
	}
}

func (c *fakeChild) emit(level, message string, fields map[string]any) bool {
	payload := map[string]any{
		"level":   level,
		"time":    time.Now().UTC().Format(time.RFC3339Nano),
		"message": message,
	}
	for key, value := range fields {
		payload[key] = value
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return true
	}
	if _, err := c.pw.Write(append(data, '\n')); err != nil {
		return false
	}
	return true
}

// Output implements runtime.Child.
func (c *fakeChild) Output() io.Reader { return c.pr }

// Wait blocks until the fake process exits.
func (c *fakeChild) Wait() error {
	<-c.done
	return nil
}

// Terminate signals the fake process to stop.
func (c *fakeChild) Terminate() error {
	c.closed.Do(func() { close(c.stop) })
	return nil
}

// Kill is identical to Terminate for the fake child.
func (c *fakeChild) Kill() error { return c.Terminate() }

// Close releases the read end.
func (c *fakeChild) Close() error { return c.pr.Close() }

// Pid returns a fixed pseudo pid.
func (c *fakeChild) Pid() int { return 4242 }

// String helps debugging.
func (c *fakeChild) String() string { return fmt.Sprintf("demo-cloudflared(%d)", c.Pid()) }
