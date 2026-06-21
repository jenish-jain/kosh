package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TokenInfo holds the verified claims from a Google ID token.
type TokenInfo struct {
	Email   string
	Name    string
	Picture string
	Aud     string
}

// TokenVerifier verifies a Google ID token and returns the verified claims.
type TokenVerifier interface {
	Verify(ctx context.Context, idToken string) (TokenInfo, error)
}

// GoogleTokenVerifier calls the Google tokeninfo endpoint.
type GoogleTokenVerifier struct {
	client *http.Client
}

// NewGoogleTokenVerifier returns a verifier that uses the given HTTP client.
func NewGoogleTokenVerifier(client *http.Client) *GoogleTokenVerifier {
	return &GoogleTokenVerifier{client: client}
}

type googleTokenResp struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Aud           string `json:"aud"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Error         string `json:"error"`
}

// Verify calls Google's tokeninfo endpoint and returns the verified claims.
func (v *GoogleTokenVerifier) Verify(ctx context.Context, idToken string) (TokenInfo, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TokenInfo{}, fmt.Errorf("building tokeninfo request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return TokenInfo{}, fmt.Errorf("could not reach Google to verify token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var info googleTokenResp
	json.Unmarshal(raw, &info) //nolint:errcheck — error surfaces via info.Error field

	if info.Error != "" || resp.StatusCode != http.StatusOK {
		return TokenInfo{}, fmt.Errorf("invalid Google token")
	}
	if info.EmailVerified != "true" {
		return TokenInfo{}, fmt.Errorf("Google account email not verified")
	}
	return TokenInfo{
		Email:   info.Email,
		Name:    info.Name,
		Picture: info.Picture,
		Aud:     info.Aud,
	}, nil
}
