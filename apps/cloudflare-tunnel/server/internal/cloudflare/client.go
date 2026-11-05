// Package cloudflare is a minimal Cloudflare API v4 client covering only the
// calls this app needs. The token is passed per call and never stored.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// DefaultBaseURL is the public API root.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// maxResponseBody caps how much of an upstream payload we buffer.
const maxResponseBody = 8 << 20

// Options configures a Client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	// RetryDelay is the pause before the single retry on 429/5xx.
	RetryDelay time.Duration
	// ZoneTTL is how long the zone list is cached.
	ZoneTTL time.Duration
}

// Client talks to the Cloudflare API v4.
type Client struct {
	base       string
	http       *http.Client
	retryDelay time.Duration
	zoneTTL    time.Duration

	mu      sync.Mutex
	zones   []model.Zone
	zonesAt time.Time
}

// New builds a client from opts.
func New(opts Options) *Client {
	c := &Client{
		base:       strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		http:       opts.HTTPClient,
		retryDelay: opts.RetryDelay,
		zoneTTL:    opts.ZoneTTL,
	}
	if c.base == "" {
		c.base = DefaultBaseURL
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 20 * time.Second}
	}
	if c.retryDelay <= 0 {
		c.retryDelay = time.Second
	}
	if c.zoneTTL <= 0 {
		c.zoneTTL = 5 * time.Minute
	}
	return c
}

type apiErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success  bool            `json:"success"`
	Errors   []apiErrorBody  `json:"errors"`
	Messages []apiErrorBody  `json:"messages"`
	Result   json.RawMessage `json:"result"`
}

// do performs one logical call with a single retry on 429/5xx.
func (c *Client) do(ctx context.Context, token, method, path string, body any) (json.RawMessage, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, model.NewError(400, "validation_failed", "request body is not encodable")
		}
		payload = b
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(c.retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, model.NewError(504, "upstream_timeout", "cloudflare request timed out")
			case <-timer.C:
			}
		}
		result, retry, err := c.once(ctx, token, method, path, payload)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

func (c *Client) once(ctx context.Context, token, method, path string, payload []byte) (json.RawMessage, bool, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bodyReader)
	if err != nil {
		return nil, false, model.NewError(502, "upstream_error", "cannot build cloudflare request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, transportError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, false, model.NewError(502, "upstream_error", "cannot read cloudflare response")
	}
	var env envelope
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			if resp.StatusCode >= 400 {
				return nil, retryableStatus(resp.StatusCode), &model.APIError{
					Status:  statusFor(resp.StatusCode),
					Code:    codeFor(resp.StatusCode),
					Message: upstreamText(raw, resp.Status),
					Details: map[string]any{"upstreamStatus": resp.StatusCode},
				}
			}
			return nil, false, model.NewError(502, "upstream_error", "cloudflare returned an unreadable response")
		}
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, envelopeError(resp.StatusCode, env)
	}
	if resp.StatusCode >= 400 || (!env.Success && len(raw) > 0) {
		return nil, false, envelopeError(resp.StatusCode, env)
	}
	return env.Result, false, nil
}

func envelopeError(status int, env envelope) *model.APIError {
	message := ""
	if len(env.Errors) > 0 {
		message = strings.TrimSpace(env.Errors[0].Message)
	}
	if message == "" {
		message = strings.TrimSpace(http.StatusText(status))
	}
	if message == "" {
		message = "cloudflare request failed"
	}
	details := map[string]any{"upstreamStatus": status}
	if len(env.Errors) > 0 {
		details["upstreamCode"] = env.Errors[0].Code
	}
	return &model.APIError{
		Status:  statusFor(status),
		Code:    codeFor(status),
		Message: message,
		Details: details,
	}
}

func statusFor(upstream int) int {
	switch {
	case upstream == http.StatusUnauthorized || upstream == http.StatusForbidden:
		// A 401 here is Cloudflare rejecting the API token, not our browser
		// session. Returning 400 keeps the UI from treating it as an expired
		// session and surfaces Cloudflare's message inline instead.
		return http.StatusBadRequest
	case upstream == http.StatusNotFound:
		return http.StatusNotFound
	case upstream == http.StatusConflict:
		return http.StatusConflict
	case upstream == http.StatusBadRequest || upstream == http.StatusUnprocessableEntity:
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

func codeFor(upstream int) string {
	switch {
	case upstream == http.StatusUnauthorized:
		return "cloudflare_auth"
	case upstream == http.StatusForbidden:
		return "cloudflare_forbidden"
	case upstream == http.StatusNotFound:
		return "not_found"
	case upstream == http.StatusConflict:
		return "conflict"
	case upstream == http.StatusBadRequest || upstream == http.StatusUnprocessableEntity:
		return "validation_failed"
	default:
		return "upstream_error"
	}
}

func retryableStatus(upstream int) bool {
	return upstream == http.StatusTooManyRequests || upstream >= 500
}

func transportError(err error) *model.APIError {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || os.IsTimeout(err) {
		return model.NewError(504, "upstream_timeout", "cloudflare did not answer in time")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return model.NewError(504, "upstream_timeout", "cloudflare did not answer in time")
	}
	msg := "cannot reach cloudflare"
	if err != nil {
		msg = "cannot reach cloudflare: " + sanitise(err.Error())
	}
	return model.NewError(502, "upstream_unreachable", msg)
}

// sanitise trims a potentially long transport error for the API envelope.
func sanitise(msg string) string {
	return upstreamText([]byte(msg), "")
}

// upstreamText converts a body to a short printable message.
func upstreamText(raw []byte, fallback string) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return fallback
	}
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) > 240 {
		runes := []rune(text)
		text = string(runes[:240]) + "…"
	}
	return text
}

// esc escapes one URL path segment.
func esc(s string) string { return url.PathEscape(s) }

// queryEscape escapes a query value.
func queryEscape(s string) string { return url.QueryEscape(s) }
