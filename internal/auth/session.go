package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// NewSessionToken returns 32 random bytes encoded as base64.RawURLEncoding,
// suitable for storage in users.session_token and as a cookie value.
func NewSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: session token rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
