// Package hosted is the multi-tenant side of Kosh: sign-up, households,
// members, and routing each request to the customer's own Google Sheet.
// Only wired up when KOSH_MODE=hosted; self-hosted Kosh never touches it.
package hosted

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"

	"kosh/internal/auth"
	"kosh/internal/store"
	sh "kosh/sheets"
)

// DriveFileScope is the only Google data scope Kosh asks for: access to
// files Kosh itself creates, nothing else in the customer's Drive.
const DriveFileScope = "https://www.googleapis.com/auth/drive.file"

var (
	// ErrScopeNotGranted means the customer unticked the Drive permission
	// on Google's consent screen.
	ErrScopeNotGranted = errors.New("Google Drive access wasn't granted")
	// ErrNoRefreshToken means Google returned no offline grant.
	ErrNoRefreshToken = errors.New("Google didn't return long-lived access")
)

// Grant is the result of exchanging an OAuth authorization code.
type Grant struct {
	RefreshToken string
	Email        string // verified, from the ID token issued with the grant
}

// Google is everything hosted mode needs from Google, behind an interface
// so tests (and the hosteddev build) can substitute fakes.
type Google interface {
	// Exchange trades a GIS popup authorization code for a grant.
	Exchange(ctx context.Context, code string) (Grant, error)
	// TokenSource returns a source minting access tokens from refreshToken.
	TokenSource(refreshToken string) oauth2.TokenSource
	// CreateSpreadsheet creates a spreadsheet in the token owner's Drive.
	CreateSpreadsheet(ctx context.Context, ts oauth2.TokenSource, title string) (string, error)
	// Sheets opens spreadsheetID as the token owner.
	Sheets(ctx context.Context, ts oauth2.TokenSource, spreadsheetID string) (store.SheetsAPI, error)
	// Revoke invalidates refreshToken at Google.
	Revoke(ctx context.Context, refreshToken string) error
}

// RealGoogle talks to Google's OAuth and Sheets APIs.
type RealGoogle struct {
	cfg      *oauth2.Config
	verifier auth.TokenVerifier
	http     *http.Client
}

// NewRealGoogle configures the authorization-code flow for the GIS popup
// client ("postmessage" is the redirect URI GIS popups use).
func NewRealGoogle(clientID, clientSecret string, verifier auth.TokenVerifier, client *http.Client) *RealGoogle {
	return &RealGoogle{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     googleoauth.Endpoint,
			RedirectURL:  "postmessage",
			Scopes:       []string{"openid", "email", DriveFileScope},
		},
		verifier: verifier,
		http:     client,
	}
}

func (g *RealGoogle) Exchange(ctx context.Context, code string) (Grant, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.http)
	tok, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return Grant{}, fmt.Errorf("exchanging authorization code: %w", err)
	}
	idToken, _ := tok.Extra("id_token").(string)
	if idToken == "" {
		return Grant{}, errors.New("Google returned no ID token")
	}
	// Verify rather than just decode: the email is what binds this Drive
	// grant to the signed-in Kosh account.
	info, err := g.verifier.Verify(ctx, idToken)
	if err != nil {
		return Grant{}, err
	}
	if info.Aud != g.cfg.ClientID {
		return Grant{}, errors.New("ID token audience mismatch")
	}
	scopes, _ := tok.Extra("scope").(string)
	if !hasScope(scopes, DriveFileScope) {
		return Grant{}, ErrScopeNotGranted
	}
	if tok.RefreshToken == "" {
		return Grant{}, ErrNoRefreshToken
	}
	return Grant{RefreshToken: tok.RefreshToken, Email: info.Email}, nil
}

func hasScope(granted, want string) bool {
	for _, s := range strings.Fields(granted) {
		if s == want {
			return true
		}
	}
	return false
}

func (g *RealGoogle) TokenSource(refreshToken string) oauth2.TokenSource {
	// Background context: the source is cached across requests, so it must
	// outlive whichever request first created it.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, g.http)
	return g.cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
}

func (g *RealGoogle) CreateSpreadsheet(ctx context.Context, ts oauth2.TokenSource, title string) (string, error) {
	return sh.CreateSpreadsheet(ctx, ts, title)
}

func (g *RealGoogle) Sheets(ctx context.Context, ts oauth2.TokenSource, spreadsheetID string) (store.SheetsAPI, error) {
	return sh.NewClientFromTokenSource(ctx, ts, spreadsheetID)
}

func (g *RealGoogle) Revoke(ctx context.Context, refreshToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke",
		strings.NewReader(url.Values{"token": {refreshToken}}.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	// 400 means the token was already invalid — revoked either way.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("revoking token: HTTP %d", resp.StatusCode)
	}
	return nil
}
