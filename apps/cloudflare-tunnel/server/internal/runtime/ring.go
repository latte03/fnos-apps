package runtime

import (
	"strings"
	"sync"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// DefaultRingCapacity mirrors the default maxLogLines setting.
const DefaultRingCapacity = 2000

// Query filters the log ring.
type Query struct {
	AfterSeq int64
	Limit    int
	Level    string
	Query    string
}

// Ring is a fixed capacity log buffer with monotonic sequence numbers. The
// sequence counter never resets, so a client that reconnects with an old
// afterSeq simply receives the retained tail.
type Ring struct {
	mu       sync.Mutex
	buf      []model.LogLine
	head     int
	capacity int
	seq      int64
	dropped  int64
}

// NewRing builds a ring buffer holding at most capacity lines.
func NewRing(capacity int) *Ring {
	if capacity <= 0 {
		capacity = DefaultRingCapacity
	}
	return &Ring{
		buf:      make([]model.LogLine, 0, capacity),
		capacity: capacity,
	}
}

// Append stores line, assigning the next sequence number.
func (r *Ring) Append(line model.LogLine) model.LogLine {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	line.Seq = r.seq
	if line.Fields == nil {
		line.Fields = map[string]any{}
	}
	if len(r.buf) < r.capacity {
		r.buf = append(r.buf, line)
		return line
	}
	r.buf[r.head] = line
	r.head = (r.head + 1) % r.capacity
	r.dropped++
	return line
}

// Query returns the newest matching lines up to q.Limit.
func (r *Ring) Query(q Query) model.LogPage {
	limit := q.Limit
	if limit <= 0 {
		limit = 500
	}
	needle := strings.ToLower(strings.TrimSpace(q.Query))
	r.mu.Lock()
	matches := make([]model.LogLine, 0, 128)
	for i := 0; i < len(r.buf); i++ {
		line := r.buf[(r.head+i)%len(r.buf)]
		if line.Seq <= q.AfterSeq {
			continue
		}
		if !levelMatch(q.Level, line.Level) {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(line.Message), needle) {
			continue
		}
		matches = append(matches, line)
	}
	if len(matches) > limit {
		matches = matches[len(matches)-limit:]
	}
	next := q.AfterSeq
	if len(matches) > 0 {
		next = matches[len(matches)-1].Seq
	}
	dropped := r.dropped
	r.mu.Unlock()
	return model.LogPage{Lines: matches, NextSeq: next, Dropped: dropped}
}

// Recent returns the newest n lines.
func (r *Ring) Recent(n int) []model.LogLine {
	if n <= 0 {
		n = 50
	}
	return r.Query(Query{Limit: n}).Lines
}

// Clear empties the buffer but keeps the sequence counter monotonic.
func (r *Ring) Clear() {
	r.mu.Lock()
	r.buf = r.buf[:0]
	r.head = 0
	r.dropped = 0
	r.mu.Unlock()
}

// Len reports the retained line count.
func (r *Ring) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.buf)
}

// Capacity reports the configured capacity.
func (r *Ring) Capacity() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.capacity
}

// SetCapacity resizes the buffer, keeping the newest lines.
func (r *Ring) SetCapacity(capacity int) {
	if capacity <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if capacity == r.capacity && r.head == 0 {
		return
	}
	ordered := make([]model.LogLine, 0, len(r.buf))
	for i := 0; i < len(r.buf); i++ {
		ordered = append(ordered, r.buf[(r.head+i)%len(r.buf)])
	}
	if len(ordered) > capacity {
		ordered = ordered[len(ordered)-capacity:]
	}
	next := make([]model.LogLine, len(ordered), capacity)
	copy(next, ordered)
	r.buf = next
	r.head = 0
	r.capacity = capacity
}

// Text renders every retained line for download.
func (r *Ring) Text() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	for i := 0; i < len(r.buf); i++ {
		line := r.buf[(r.head+i)%len(r.buf)]
		b.WriteString(line.TS)
		b.WriteByte(' ')
		b.WriteString(line.Level)
		b.WriteByte(' ')
		b.WriteString(line.Message)
		b.WriteByte('\n')
	}
	return b.String()
}

func levelMatch(filter, level string) bool {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return true
	default:
		return strings.EqualFold(strings.TrimSpace(filter), level)
	}
}
