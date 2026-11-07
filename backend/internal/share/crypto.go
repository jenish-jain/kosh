package share

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// MinKeyLength is the shortest SHARE_ENCRYPTION_KEY accepted — long enough
// that `openssl rand -hex 32` passes and a typed-in word doesn't.
const MinKeyLength = 32

// Sealer encrypts TOTP secrets at rest with AES-256-GCM, so someone with only
// view access to the Sheet can't copy a secret and mint codes.
type Sealer struct {
	aead cipher.AEAD
}

// NewSealer derives an AES-256 key from key (SHA-256) and returns a Sealer.
func NewSealer(key string) (*Sealer, error) {
	if len(key) < MinKeyLength {
		return nil, errors.New("encryption key too short")
	}
	k := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead}, nil
}

// Seal encrypts plain and returns base64(nonce | ciphertext).
func (s *Sealer) Seal(plain string) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(s.aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// Open reverses Seal.
func (s *Sealer) Open(enc string) (string, error) {
	b, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	n := s.aead.NonceSize()
	if len(b) < n {
		return "", errors.New("ciphertext too short")
	}
	plain, err := s.aead.Open(nil, b[:n], b[n:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// NewLinkToken returns a 256-bit random, URL-safe token for a share link.
func NewLinkToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what's stored instead of the raw link token. A plain SHA-256
// is enough: the token is 256 bits of randomness, so there's nothing to
// brute-force that a slow hash would protect.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func randomID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
