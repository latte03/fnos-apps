package httpapi

import (
	"net/http"
	"strings"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

// maxKeyName bounds a key label.
const maxKeyName = 64

type keyView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt"`
}

func newKeyView(key store.APIKey) keyView {
	return keyView{
		ID:         key.ID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		CreatedAt:  key.CreatedAt,
		LastUsedAt: key.LastUsedAt,
	}
}

type keysResponse struct {
	Keys []keyView `json:"keys"`
}

func (s *Server) handleKeysList(w http.ResponseWriter, _ *http.Request) {
	keys := s.store.Keys()
	out := make([]keyView, 0, len(keys))
	for _, key := range keys {
		out = append(out, newKeyView(key))
	}
	writeJSON(w, http.StatusOK, keysResponse{Keys: out})
}

type createKeyRequest struct {
	Name string `json:"name"`
}

type createKeyResponse struct {
	Key   keyView `json:"key"`
	Token string  `json:"token"`
}

func (s *Server) handleKeysCreate(w http.ResponseWriter, r *http.Request) {
	var body createKeyRequest
	if !decodeJSON(w, r, &body, false) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "name is required"))
		return
	}
	if len([]rune(name)) > maxKeyName {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "name is too long"))
		return
	}
	key, token, err := s.store.AddKey(name)
	if err != nil {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "cannot store the API key"))
		return
	}
	s.addActivity("key", "API 密钥 "+name+" 已创建", "teal")
	writeJSON(w, http.StatusCreated, createKeyResponse{Key: newKeyView(key), Token: token})
}

func (s *Server) handleKeysDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "key id is required"))
		return
	}
	removed, err := s.store.RemoveKey(id)
	if err != nil {
		writeAPIError(w, model.NewError(http.StatusBadGateway, "internal_error", "cannot remove the API key"))
		return
	}
	if !removed {
		writeAPIError(w, model.NewError(http.StatusNotFound, "not_found", "API key not found"))
		return
	}
	s.addActivity("key", "API 密钥已撤销", "danger")
	writeNoContent(w)
}
