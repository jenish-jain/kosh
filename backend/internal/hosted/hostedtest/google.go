// Package hostedtest provides a fake Google for hosted-mode tests and the
// hosteddev build. Never wired into a production binary.
package hostedtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"kosh/internal/hosted"
	"kosh/internal/store"
	sh "kosh/sheets"
)

// Google is an in-memory hosted.Google. Authorization codes have the form
// "code:<email>" (or "code:<email>:noscope" / ":norefresh" to simulate a
// customer unticking Drive access or Google withholding offline access).
// Each created spreadsheet is a FakeSheetsAPI.
type Google struct {
	mu      sync.Mutex
	sheets  map[string]*store.FakeSheetsAPI
	owners  map[string]string // spreadsheet -> owner email
	revoked map[string]bool   // refresh tokens
	// revokedAt: every token for an email issued at or before this instant
	// is revoked (the customer removed Kosh's access in their account).
	revokedAt map[string]int64
	n         int
	issued    int64
}

// New returns an empty fake.
func New() *Google {
	return &Google{sheets: map[string]*store.FakeSheetsAPI{}, owners: map[string]string{}, revoked: map[string]bool{}, revokedAt: map[string]int64{}}
}

// Code returns an authorization code that exchanges to a grant for email.
func Code(email string) string { return "code:" + email }

func (g *Google) Exchange(_ context.Context, code string) (hosted.Grant, error) {
	parts := strings.Split(code, ":")
	if len(parts) < 2 || parts[0] != "code" || parts[1] == "" {
		return hosted.Grant{}, errors.New("invalid_grant")
	}
	if len(parts) == 3 {
		switch parts[2] {
		case "noscope":
			return hosted.Grant{}, hosted.ErrScopeNotGranted
		case "norefresh":
			return hosted.Grant{}, hosted.ErrNoRefreshToken
		}
	}
	g.mu.Lock()
	g.issued++
	seq := g.issued
	g.mu.Unlock()
	return hosted.Grant{RefreshToken: fmt.Sprintf("rt:%s:%d", parts[1], seq), Email: parts[1]}, nil
}

// Revoke simulates the customer removing Kosh's access in their Google
// account: the token's next refresh fails with invalid_grant.
func (g *Google) Revoke(_ context.Context, refreshToken string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.revoked[refreshToken] = true
	return nil
}

// RevokeAll revokes every token issued for email so far; grants issued
// afterwards (a reconnect) work.
func (g *Google) RevokeAll(email string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.revokedAt[email] = g.issued
}

type source struct {
	g  *Google
	rt string
}

func (s source) Token() (*oauth2.Token, error) {
	s.g.mu.Lock()
	defer s.g.mu.Unlock()
	parts := strings.Split(strings.TrimPrefix(s.rt, "rt:"), ":")
	var seq int64
	fmt.Sscan(parts[len(parts)-1], &seq)
	if s.g.revoked[s.rt] || seq <= s.g.revokedAt[parts[0]] {
		return nil, &oauth2.RetrieveError{ErrorCode: "invalid_grant"}
	}
	return &oauth2.Token{AccessToken: "at:" + s.rt, Expiry: time.Now().Add(time.Second)}, nil
}

func (g *Google) TokenSource(refreshToken string) oauth2.TokenSource {
	return source{g: g, rt: refreshToken}
}

func emailOf(ts oauth2.TokenSource) (string, error) {
	tok, err := ts.Token()
	if err != nil {
		return "", err
	}
	return strings.Split(strings.TrimPrefix(tok.AccessToken, "at:rt:"), ":")[0], nil
}

func (g *Google) CreateSpreadsheet(_ context.Context, ts oauth2.TokenSource, _ string) (string, error) {
	email, err := emailOf(ts)
	if err != nil {
		return "", err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	id := fmt.Sprintf("sheet-%d", g.n)
	g.sheets[id] = store.NewFakeSheetsAPI()
	g.owners[id] = email
	return id, nil
}

// Sheets enforces drive.file semantics: a token can only open spreadsheets
// created by the same Google account.
func (g *Google) Sheets(_ context.Context, ts oauth2.TokenSource, id string) (store.SheetsAPI, error) {
	email, err := emailOf(ts)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.sheets[id]
	if !ok || g.owners[id] != email {
		return nil, errors.New("404: spreadsheet not found")
	}
	return &authedSheets{ts: ts, inner: s}, nil
}

// authedSheets checks the token on every call, like the real client (which
// attaches a fresh access token to each request) — so a revoked grant
// fails the next Sheets operation, not just the next client construction.
type authedSheets struct {
	ts    oauth2.TokenSource
	inner *store.FakeSheetsAPI
}

func (a *authedSheets) ReadSheet(name string) ([][]interface{}, error) {
	if _, err := a.ts.Token(); err != nil {
		return nil, err
	}
	return a.inner.ReadSheet(name)
}

func (a *authedSheets) ReadSheets(names []string) (map[string][][]interface{}, error) {
	if _, err := a.ts.Token(); err != nil {
		return nil, err
	}
	return a.inner.ReadSheets(names)
}

func (a *authedSheets) AppendRow(sheet string, row []interface{}) (int, error) {
	if _, err := a.ts.Token(); err != nil {
		return 0, err
	}
	return a.inner.AppendRow(sheet, row)
}

func (a *authedSheets) UpdateRow(sheet, id string, row []interface{}) error {
	if _, err := a.ts.Token(); err != nil {
		return err
	}
	return a.inner.UpdateRow(sheet, id, row)
}

func (a *authedSheets) DeleteRow(sheet, id string) error {
	if _, err := a.ts.Token(); err != nil {
		return err
	}
	return a.inner.DeleteRow(sheet, id)
}

func (a *authedSheets) EnsureTabs(tabs []sh.TabDef) error {
	if _, err := a.ts.Token(); err != nil {
		return err
	}
	return a.inner.EnsureTabs(tabs)
}

// Sheet returns a created spreadsheet's fake, for assertions.
func (g *Google) Sheet(id string) *store.FakeSheetsAPI {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sheets[id]
}

var _ hosted.Google = (*Google)(nil)
