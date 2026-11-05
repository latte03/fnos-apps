package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
)

// maxLogLimit bounds a single /logs page.
const maxLogLimit = 5000

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	level := strings.ToLower(strings.TrimSpace(query.Get("level")))
	switch level {
	case "", "all", "debug", "info", "warn", "error":
	default:
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "level must be all, debug, info, warn or error"))
		return
	}
	limit := intQuery(query.Get("limit"), 500)
	if limit < 1 {
		limit = 1
	}
	if limit > maxLogLimit {
		limit = maxLogLimit
	}
	afterSeq := int64(intQuery(query.Get("afterSeq"), 0))
	page := s.engine.Logs(runtime.Query{
		AfterSeq: afterSeq,
		Limit:    limit,
		Level:    level,
		Query:    query.Get("q"),
	})
	if page.Lines == nil {
		page.Lines = []model.LogLine{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleLogsClear(w http.ResponseWriter, _ *http.Request) {
	s.engine.ClearLogs()
	s.addActivity("logs", "日志已清空", "neutral")
	writeNoContent(w)
}

func (s *Server) handleLogsDownload(w http.ResponseWriter, _ *http.Request) {
	body := s.engine.Text()
	header := w.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("Content-Disposition", `attachment; filename="cloudflared.log"`)
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, body)
}

func intQuery(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
