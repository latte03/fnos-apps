package store

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// HashToken returns the lowercase hex SHA-256 of a bearer token. Only this
// value is written to disk.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// PreviewToken masks a secret, keeping the last four characters. The API token
// is never exposed beyond this preview.
func PreviewToken(token string) string {
	if token == "" {
		return ""
	}
	tail := token
	if len(tail) > 4 {
		tail = tail[len(tail)-4:]
	}
	return "••••" + tail
}

func subtleEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
