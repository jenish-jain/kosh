package controlplane

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// MinKeyLength is the shortest CONTROL_PLANE_ENCRYPTION_KEY accepted.
const MinKeyLength = 32

// TokenSealer encrypts per-household secrets (Google refresh tokens) with
// AES-256-GCM. Every ciphertext is bound to an associated-data string — the
// household ID — so a token copied from one household's record onto
// another's fails to decrypt instead of granting access to the wrong Drive.
type TokenSealer struct {
	aead cipher.AEAD
}

// NewTokenSealer derives an AES-256 key from key via SHA-256.
func NewTokenSealer(key string) (*TokenSealer, error) {
	if len(key) < MinKeyLength {
		return nil, errors.New("encryption key too short")
	}
	k := sha256.Sum256([]byte("kosh-control-plane:" + key))
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TokenSealer{aead: aead}, nil
}

// Seal encrypts plain bound to aad, returning base64(nonce | ciphertext).
func (s *TokenSealer) Seal(plain, aad string) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(s.aead.Seal(nonce, nonce, []byte(plain), []byte(aad))), nil
}

// Open decrypts enc, failing unless it was sealed with the same aad.
func (s *TokenSealer) Open(enc, aad string) (string, error) {
	b, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	n := s.aead.NonceSize()
	if len(b) < n {
		return "", errors.New("ciphertext too short")
	}
	plain, err := s.aead.Open(nil, b[:n], b[n:], []byte(aad))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
