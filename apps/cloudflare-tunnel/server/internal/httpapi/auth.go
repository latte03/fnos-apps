package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// sessionCookie is the browser session cookie name from the contract.
const sessionCookie = "ct_session"

// sessionTTL bounds how long an idle browser session is kept in memory.
const sessionTTL = 12 * 60 * 60

// maxSessions caps the in-memory session table.
const maxSessions = 1024

type sessionStore struct {
	mu sync.Mutex
	m  map[string]int64 // id -> issued unix seconds
}

func newSessionStore() *sessionStore {
	return &sessionStore{m: make(map[string]int64)}
}

// issue creates a new 32 byte hex session id.
func (s *sessionStore) issue() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	id := hex.EncodeToString(buf)
	now := time.Now().Unix()
	s.mu.Lock()
	if len(s.m) >= maxSessions {
		for key, issued := range s.m {
			if now-issued > sessionTTL {
				delete(s.m, key)
			}
		}
		if len(s.m) >= maxSessions {
			// Drop an arbitrary entry; sessions are cheap to recreate.
			for key := range s.m {
				delete(s.m, key)
				break
			}
		}
	}
	s.m[id] = now
	s.mu.Unlock()
	return id
}

// valid reports whether id is a live session.
func (s *sessionStore) valid(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	issued, ok := s.m[id]
	if !ok {
		return false
	}
	if time.Now().Unix()-issued > sessionTTL {
		delete(s.m, id)
		return false
	}
	return true
}

// ensure issues a cookie when the request has no valid session.
func (s *sessionStore) ensure(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && s.valid(c.Value) {
		return
	}
	id := s.issue()
	if id == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionTTL,
	})
}

type authKindKey struct{}

func withAuthKind(ctx context.Context, kind string) context.Context {
	return context.WithValue(ctx, authKindKey{}, kind)
}

// authKind reports how the request authenticated: "session" or "key".
func authKind(r *http.Request) string {
	if v, ok := r.Context().Value(authKindKey{}).(string); ok {
		return v
	}
	return ""
}

// auth requires a session cookie or a bearer API key.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookie); err == nil && s.sessions.valid(c.Value) {
			next(w, r.WithContext(withAuthKind(r.Context(), "session")))
			return
		}
		header := r.Header.Get("Authorization")
		if len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
			token := strings.TrimSpace(header[7:])
			if token != "" {
				if _, ok := s.store.Authenticate(token); ok {
					next(w, r.WithContext(withAuthKind(r.Context(), "key")))
					return
				}
			}
		}
		writeAPIError(w, model.NewError(http.StatusUnauthorized, "unauthorized", "authentication required"))
	}
}
