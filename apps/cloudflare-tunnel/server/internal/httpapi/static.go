package httpapi

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/assets"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

var contentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".json":        "application/json; charset=utf-8",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".ico":         "image/x-icon",
	".webp":        "image/webp",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".map":         "application/json; charset=utf-8",
	".txt":         "text/plain; charset=utf-8",
	".webmanifest": "application/manifest+json",
}

func contentTypeFor(name string) string {
	if ct, ok := contentTypes[strings.ToLower(path.Ext(name))]; ok {
		return ct
	}
	return "application/octet-stream"
}

// handleStatic serves the embedded SPA. Unknown non-API paths fall back to
// index.html so the hash router keeps working on reload.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAPIError(w, model.NewError(http.StatusNotFound, "not_found", "unknown endpoint "+r.Method+" "+r.URL.Path))
		return
	}
	urlPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if urlPath == "" || urlPath == "." {
		s.serveIndex(w, r)
		return
	}
	if strings.HasPrefix(urlPath, "api/") {
		s.handleAPINotFound(w, r)
		return
	}
	dist := assets.Dist()
	if info, err := fs.Stat(dist, urlPath); err == nil && !info.IsDir() {
		if data, err := fs.ReadFile(dist, urlPath); err == nil {
			header := w.Header()
			header.Set("Content-Type", contentTypeFor(urlPath))
			if strings.HasPrefix(urlPath, "assets/") {
				header.Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				header.Set("Cache-Control", "no-cache")
			}
			header.Set("X-Content-Type-Options", "nosniff")
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			_, _ = w.Write(data)
			return
		}
	}
	s.serveIndex(w, r)
}

// serveIndex serves the SPA shell and makes sure the browser holds a session.
func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("index.html")
	if err != nil {
		http.Error(w, "index.html is missing from the embedded bundle", http.StatusInternalServerError)
		return
	}
	s.sessions.ensure(w, r)
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
	header.Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write(data)
}
