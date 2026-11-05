package runtime

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// LogTimeFormat keeps millisecond resolution for log lines; it is still a
// valid RFC 3339 timestamp.
const LogTimeFormat = "2006-01-02T15:04:05.000Z07:00"

// ParseLine converts one line of cloudflared output into a LogLine. JSON lines
// (--logformat json) are split into level/time/message plus the remaining
// fields; anything else is kept verbatim at level info.
func ParseLine(raw string, now time.Time) (model.LogLine, bool) {
	text := strings.TrimSpace(strings.TrimRight(raw, "\r"))
	if text == "" {
		return model.LogLine{}, false
	}
	line := model.LogLine{
		TS:      now.UTC().Format(LogTimeFormat),
		Level:   "info",
		Message: text,
		Fields:  map[string]any{},
	}
	if !strings.HasPrefix(text, "{") {
		return line, true
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return line, true
	}
	fields := make(map[string]any, len(payload))
	for key, value := range payload {
		switch key {
		case "level":
			if s, ok := value.(string); ok {
				line.Level = NormalizeLevel(s)
			}
		case "time":
			if s, ok := value.(string); ok {
				if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
					line.TS = ts.UTC().Format(LogTimeFormat)
				} else {
					fields["time"] = s
				}
			}
		case "message":
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				line.Message = s
			}
		default:
			fields[key] = value
		}
	}
	line.Fields = fields
	return line, true
}

// NormalizeLevel maps cloudflared and zap level names onto the four levels the
// UI filters by.
func NormalizeLevel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "trace":
		return "debug"
	case "info", "notice":
		return "info"
	case "warn", "warning":
		return "warn"
	case "error", "fatal", "panic", "critical", "dpanic":
		return "error"
	case "":
		return "info"
	}
	return "info"
}

// FieldString reads a string valued field.
func FieldString(fields map[string]any, key string) (string, bool) {
	if fields == nil {
		return "", false
	}
	if v, ok := fields[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}
