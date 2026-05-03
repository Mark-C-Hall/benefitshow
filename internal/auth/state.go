package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// NewState returns 16 random bytes encoded as base64.RawURLEncoding,
// for use as the OAuth `state` parameter.
func NewState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: state rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SignState returns "<state>.<hex(HMAC-SHA256(secret, state))>" so the cookie
// value can be verified came from this server.
func SignState(secret []byte, state string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	return state + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifyState splits a signed value, checks the HMAC in constant time, and
// returns the raw state on success.
func VerifyState(secret []byte, signed string) (string, bool) {
	dot := strings.LastIndexByte(signed, '.')
	if dot <= 0 || dot == len(signed)-1 {
		return "", false
	}
	state, sigHex := signed[:dot], signed[dot+1:]
	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return "", false
	}
	return state, true
}
