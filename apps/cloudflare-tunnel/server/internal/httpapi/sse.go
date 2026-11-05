package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
)

// subscriberBuffer is how many events a single SSE client may fall behind.
const subscriberBuffer = 64

// pingInterval is the SSE keep-alive cadence.
const pingInterval = 15 * time.Second

type sseEvent struct {
	name string
	data []byte
}

type subscriber struct {
	ch chan sseEvent
}

// Hub fans events out to SSE clients without ever blocking the producer.
type Hub struct {
	mu      sync.Mutex
	subs    map[*subscriber]struct{}
	dropped int64
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[*subscriber]struct{})}
}

// Subscribe registers a client.
func (h *Hub) Subscribe() *subscriber {
	sub := &subscriber{ch: make(chan sseEvent, subscriberBuffer)}
	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()
	return sub
}

// Unsubscribe removes a client and closes its channel.
func (h *Hub) Unsubscribe(sub *subscriber) {
	h.mu.Lock()
	if _, ok := h.subs[sub]; ok {
		delete(h.subs, sub)
		close(sub.ch)
	}
	h.mu.Unlock()
}

// Broadcast marshals payload and queues it for every client. Slow clients are
// dropped rather than blocking the hub.
func (h *Hub) Broadcast(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.send(sseEvent{name: name, data: data})
}

func (h *Hub) send(ev sseEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs {
		select {
		case sub.ch <- ev:
		default:
			h.dropped++
		}
	}
}

// Count reports the number of connected clients.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// Dropped reports how many events were skipped because a client was slow.
func (h *Hub) Dropped() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dropped
}

// Status implements runtime.Sink.
func (s *Server) Status(rt model.Runtime) { s.hub.Broadcast("status", rt) }

// Log implements runtime.Sink.
func (s *Server) Log(line model.LogLine) { s.hub.Broadcast("log", line) }

// Metrics implements runtime.Sink.
func (s *Server) Metrics(m model.RuntimeMetrics) { s.hub.Broadcast("metrics", m) }

// PublishActivity pushes an activity item to SSE clients.
func (s *Server) PublishActivity(item model.Activity) { s.hub.Broadcast("activity", item) }

type helloPayload struct {
	Status model.Runtime   `json:"status"`
	Recent []model.LogLine `json:"recent"`
}

// handleEvents streams the SSE feed.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "streaming is not supported"))
		return
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-store, no-transform")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sub := s.hub.Subscribe()
	defer s.hub.Unsubscribe(sub)

	hello := helloPayload{
		Status: s.engine.State(),
		Recent: s.engine.Logs(runtime.Query{Limit: 50}).Lines,
	}
	if !writeSSE(w, flusher, "hello", hello) {
		return
	}

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-sub.ch:
			if !ok {
				return
			}
			if !writeSSEEvent(w, flusher, ev) {
				return
			}
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, name string, payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return writeSSEEvent(w, flusher, sseEvent{name: name, data: data})
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, ev sseEvent) bool {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, ev.data); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
