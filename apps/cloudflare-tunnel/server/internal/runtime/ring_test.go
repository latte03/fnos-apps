package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

func TestParseLineJSON(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	raw := `{"level":"info","time":"2026-11-05T12:00:01.123456789Z","message":"Registered tunnel connection","connIndex":0,"location":"HKG"}`
	line, ok := ParseLine(raw, now)
	if !ok {
		t.Fatal("expected a parsed line")
	}
	if line.Level != "info" || line.Message != "Registered tunnel connection" {
		t.Fatalf("unexpected line: %+v", line)
	}
	if !strings.HasPrefix(line.TS, "2026-11-05T12:00:01") {
		t.Fatalf("timestamp not taken from the payload: %q", line.TS)
	}
	if location, ok := FieldString(line.Fields, "location"); !ok || location != "HKG" {
		t.Fatalf("location missing: %+v", line.Fields)
	}
	if _, ok := line.Fields["message"]; ok {
		t.Fatal("message must not be duplicated into fields")
	}
	if _, ok := line.Fields["level"]; ok {
		t.Fatal("level must not be duplicated into fields")
	}
	if _, ok := line.Fields["time"]; ok {
		t.Fatal("time must not be duplicated into fields")
	}
}

func TestParseLinePlainText(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	line, ok := ParseLine("  plain output line \r", now)
	if !ok {
		t.Fatal("expected a parsed line")
	}
	if line.Message != "plain output line" || line.Level != "info" {
		t.Fatalf("unexpected line: %+v", line)
	}
	if line.Fields == nil || len(line.Fields) != 0 {
		t.Fatalf("expected empty fields, got %+v", line.Fields)
	}
	if line.TS != now.UTC().Format(LogTimeFormat) {
		t.Fatalf("unexpected ts %q", line.TS)
	}
}

func TestParseLineRejectsBlank(t *testing.T) {
	if _, ok := ParseLine("   \n", time.Now()); ok {
		t.Fatal("blank lines must be skipped")
	}
}

func TestParseLineNormalisesLevels(t *testing.T) {
	cases := map[string]string{
		"WARN":     "warn",
		"warning":  "warn",
		"fatal":    "error",
		"panic":    "error",
		"DEBUG":    "debug",
		"nonsense": "info",
	}
	for input, want := range cases {
		if got := NormalizeLevel(input); got != want {
			t.Fatalf("NormalizeLevel(%q) = %q, want %q", input, got, want)
		}
	}
}

func appendLine(r *Ring, level, message string) model.LogLine {
	return r.Append(model.LogLine{
		TS:      "2026-11-05T12:00:00.000Z",
		Level:   level,
		Message: message,
		Fields:  map[string]any{},
	})
}

func TestRingCapacityAndDrops(t *testing.T) {
	ring := NewRing(2)
	appendLine(ring, "info", "one")
	appendLine(ring, "info", "two")
	appendLine(ring, "info", "three")

	page := ring.Query(Query{Limit: 10})
	if len(page.Lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(page.Lines))
	}
	if page.Lines[0].Message != "two" || page.Lines[1].Message != "three" {
		t.Fatalf("wrong retained lines: %+v", page.Lines)
	}
	if page.Lines[1].Seq != 3 {
		t.Fatalf("sequence not monotonic: %d", page.Lines[1].Seq)
	}
	if page.Dropped != 1 {
		t.Fatalf("dropped = %d, want 1", page.Dropped)
	}
}

func TestRingFiltersAndLimit(t *testing.T) {
	ring := NewRing(50)
	appendLine(ring, "info", "Started metrics server")
	appendLine(ring, "warn", "Connection terminated")
	appendLine(ring, "info", "GET https://nas.example.com/ 200")
	appendLine(ring, "error", "origin refused connection")

	page := ring.Query(Query{Limit: 10, Level: "warn"})
	if len(page.Lines) != 1 || page.Lines[0].Level != "warn" {
		t.Fatalf("level filter failed: %+v", page.Lines)
	}

	page = ring.Query(Query{Limit: 10, Query: "NAS.EXAMPLE"})
	if len(page.Lines) != 1 {
		t.Fatalf("substring filter is not case insensitive: %+v", page.Lines)
	}

	page = ring.Query(Query{Limit: 10, AfterSeq: 3})
	if len(page.Lines) != 1 || page.Lines[0].Seq != 4 {
		t.Fatalf("afterSeq filter failed: %+v", page.Lines)
	}
	if page.NextSeq != 4 {
		t.Fatalf("nextSeq = %d, want 4", page.NextSeq)
	}

	page = ring.Query(Query{Limit: 2})
	if len(page.Lines) != 2 || page.Lines[0].Seq != 3 || page.Lines[1].Seq != 4 {
		t.Fatalf("limit must keep the newest lines: %+v", page.Lines)
	}

	recent := ring.Recent(1)
	if len(recent) != 1 || recent[0].Seq != 4 {
		t.Fatalf("Recent: %+v", recent)
	}
}

func TestRingClearKeepsSequence(t *testing.T) {
	ring := NewRing(10)
	appendLine(ring, "info", "before")
	ring.Clear()
	if ring.Len() != 0 {
		t.Fatalf("ring not cleared: %d", ring.Len())
	}
	line := appendLine(ring, "info", "after")
	if line.Seq != 2 {
		t.Fatalf("sequence restarted: %d", line.Seq)
	}
	if got := ring.Text(); !strings.Contains(got, "after") || strings.Contains(got, "before") {
		t.Fatalf("unexpected text output: %q", got)
	}
}

func TestRingSetCapacity(t *testing.T) {
	ring := NewRing(10)
	for i := 0; i < 6; i++ {
		appendLine(ring, "info", "line")
	}
	ring.SetCapacity(3)
	if ring.Len() != 3 || ring.Capacity() != 3 {
		t.Fatalf("resize failed: len=%d cap=%d", ring.Len(), ring.Capacity())
	}
	page := ring.Query(Query{Limit: 10})
	if page.Lines[2].Seq != 6 {
		t.Fatalf("resize dropped the newest line: %+v", page.Lines)
	}
}

func TestParseMetrics(t *testing.T) {
	body := `# HELP cloudflared_tunnel_ha_connections Number of active connections
# TYPE cloudflared_tunnel_ha_connections gauge
cloudflared_tunnel_ha_connections{connIndex="0"} 1
cloudflared_tunnel_ha_connections{connIndex="1"} 1
cloudflared_tunnel_total_requests 12400
cloudflared_tunnel_request_errors 3
cloudflared_tunnel_rtt{connIndex="0"} 0.038
`
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	metrics := ParseMetrics(body, now)
	if metrics.Requests == nil || *metrics.Requests != 12400 {
		t.Fatalf("requests = %v", metrics.Requests)
	}
	if metrics.Errors == nil || *metrics.Errors != 3 {
		t.Fatalf("errors = %v", metrics.Errors)
	}
	if metrics.Connections == nil || *metrics.Connections != 2 {
		t.Fatalf("connections = %v", metrics.Connections)
	}
	if metrics.RTTMs == nil || *metrics.RTTMs != 38 {
		t.Fatalf("rtt = %v", metrics.RTTMs)
	}
	if metrics.SampledAt == "" {
		t.Fatal("sampledAt missing")
	}
}

func TestParseMetricsMissingValuesStayNil(t *testing.T) {
	metrics := ParseMetrics("# no metrics yet\n", time.Now())
	if metrics.Requests != nil || metrics.Errors != nil || metrics.Connections != nil || metrics.RTTMs != nil {
		t.Fatalf("expected nil values, got %+v", metrics)
	}
}

func TestParseMetricsAlternateNames(t *testing.T) {
	body := "cloudflared_tunnel_request_total 7\ncloudflared_tunnel_rtt_seconds 0.12\n"
	metrics := ParseMetrics(body, time.Now())
	if metrics.Requests == nil || *metrics.Requests != 7 {
		t.Fatalf("alternate request counter not read: %v", metrics.Requests)
	}
	if metrics.RTTMs == nil || *metrics.RTTMs != 120 {
		t.Fatalf("alternate rtt not converted: %v", metrics.RTTMs)
	}
}
