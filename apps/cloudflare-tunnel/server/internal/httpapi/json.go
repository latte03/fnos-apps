package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// maxRequestBody caps JSON request bodies.
const maxRequestBody = 1 << 20

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

// writeNoContent writes a bare 204.
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// writeAPIError renders the contract error envelope.
func writeAPIError(w http.ResponseWriter, err error) {
	apiErr := asAPIError(err)
	details := apiErr.Details
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, apiErr.Status, errorEnvelope{Error: errorBody{
		Code:    apiErr.Code,
		Message: apiErr.Message,
		Details: details,
	}})
}

// asAPIError normalises any error into an APIError.
func asAPIError(err error) *model.APIError {
	var apiErr *model.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	if err == nil {
		return model.NewError(http.StatusBadGateway, "internal_error", "unexpected error")
	}
	return model.NewError(http.StatusBadGateway, "internal_error", err.Error())
}

// decodeJSON decodes a JSON body. An empty body is accepted when allowEmpty is
// true (used by POST /settings/verify and /runtime/disconnect).
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	if r.Body == nil {
		if allowEmpty {
			return true
		}
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "request body is required"))
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) && allowEmpty {
			return true
		}
		writeAPIError(w, model.NewError(http.StatusBadRequest, "validation_failed", "request body is not valid JSON"))
		return false
	}
	return true
}
