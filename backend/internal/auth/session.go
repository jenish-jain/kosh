package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Session signs and verifies HMAC-based session tokens.
type Session struct {
	secret string
}

// NewSession creates a Session with the given HMAC secret.
func NewSession(secret string) *Session {
	return &Session{secret: secret}
}

// Sign creates a signed token encoding email and a 30-day expiry.
// Token format: base64(email) | base64(expiry_unix) | hmac(payload, secret)
func (s *Session) Sign(email string) string {
	exp := fmt.Sprint(time.Now().Add(30 * 24 * time.Hour).Unix())
	payload := base64.RawURLEncoding.EncodeToString([]byte(email)) + "|" +
		base64.RawURLEncoding.EncodeToString([]byte(exp))
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	return payload + "|" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verify validates a signed token and returns the email, or an error if
// the token is invalid or expired.
func (s *Session) Verify(token string) (string, error) {
	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return "", fmt.Errorf("invalid token signature")
	}
	expB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token expiry encoding")
	}
	var exp int64
	fmt.Sscan(string(expB), &exp)
	if time.Now().Unix() > exp {
		return "", fmt.Errorf("token expired")
	}
	emailB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid token email encoding")
	}
	return string(emailB), nil
}
