package app_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kosh/internal/app"
	"kosh/internal/auth"
	"kosh/internal/controlplane"
	"kosh/internal/hosted/hostedtest"
	"kosh/internal/share"
	"kosh/internal/store"
)

const clientID = "test-client.apps.googleusercontent.com"

// idVerifier accepts credentials of the form "id:<email>", standing in for
// Google's tokeninfo check.
type idVerifier struct{}

func (idVerifier) Verify(_ context.Context, tok string) (auth.TokenInfo, error) {
	if !strings.HasPrefix(tok, "id:") {
		return auth.TokenInfo{}, errors.New("invalid Google token")
	}
	return auth.TokenInfo{Email: strings.TrimPrefix(tok, "id:"), Aud: clientID}, nil
}

func hostedConfig() app.Config {
	k := strings.Repeat("k", 40)
	return app.Config{
		Mode: app.ModeHosted, FrontendDist: "/nonexistent", GoogleClientID: clientID,
		GoogleClientSecret: "secret", SessionSecret: k, CookieSecure: true,
		ShareKey: k, ControlPlaneKey: strings.Repeat("c", 40),
	}
}

type env struct {
	t      *testing.T
	srv    *httptest.Server
	google *hostedtest.Google
	store  *controlplane.Memory
}

func newHostedEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, google: hostedtest.New(), store: controlplane.NewMemory()}
	a, err := app.New(context.Background(), hostedConfig(), app.Deps{Verifier: idVerifier{}, Store: e.store, Google: e.google})
	if err != nil {
		t.Fatal(err)
	}
	// TLS: hosted cookies are Secure, and a cookie jar won't send those
	// over plain HTTP.
	e.srv = httptest.NewTLSServer(a.Handler)
	t.Cleanup(e.srv.Close)
	return e
}

// browser is one person's browser: its own cookie jar.
type browser struct {
	e      *env
	c      *http.Client
	header http.Header
}

func (e *env) browser() *browser {
	jar, _ := cookiejar.New(nil)
	// Copy: srv.Client() returns one shared *http.Client, and each browser
	// needs its own cookie jar.
	c := *e.srv.Client()
	c.Jar = jar
	return &browser{e: e, c: &c, header: http.Header{}}
}

func (b *browser) do(method, path string, body any) (int, string) {
	b.e.t.Helper()
	var r io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		r = bytes.NewReader(buf)
	}
	req, _ := http.NewRequest(method, b.e.srv.URL+path, r)
	for k, v := range b.header {
		req.Header[k] = v
	}
	resp, err := b.c.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

func (b *browser) json(method, path string, body any, into any) int {
	b.e.t.Helper()
	code, out := b.do(method, path, body)
	if into != nil && code == 200 {
		if err := json.Unmarshal([]byte(out), into); err != nil {
			b.e.t.Fatalf("%s %s: decoding %q: %v", method, path, out, err)
		}
	}
	return code
}

func (b *browser) login(email string) {
	b.e.t.Helper()
	if code, out := b.do("POST", "/api/auth/login", map[string]string{"credential": "id:" + email}); code != 200 {
		b.e.t.Fatalf("login %s: %d %s", email, code, out)
	}
}

type me struct {
	Email     string `json:"email"`
	Household *struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Role           string `json:"role"`
		NeedsReconnect bool   `json:"needs_reconnect"`
		SheetURL       string `json:"sheet_url"`
		Billing        struct {
			Status string `json:"status"`
		} `json:"billing"`
	} `json:"household"`
}

func (b *browser) me() me {
	b.e.t.Helper()
	var m me
	if code := b.json("GET", "/api/hosted/me", nil, &m); code != 200 {
		b.e.t.Fatalf("me: %d", code)
	}
	return m
}

func (b *browser) onboard(email, name string) {
	b.e.t.Helper()
	if code, out := b.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code(email), "name": name}); code != 200 {
		b.e.t.Fatalf("onboard %s: %d %s", email, code, out)
	}
}

func (b *browser) memberNames() []string {
	b.e.t.Helper()
	var d struct {
		Members []struct{ Name string } `json:"members"`
	}
	if code, out := b.do("GET", "/api/data", nil); code != 200 {
		b.e.t.Fatalf("data: %d %s", code, out)
	} else {
		json.Unmarshal([]byte(out), &d)
	}
	var names []string
	for _, m := range d.Members {
		names = append(names, m.Name)
	}
	return names
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestHosted_ConfigValidation(t *testing.T) {
	cfg := hostedConfig()
	cfg.ControlPlaneKey = "short"
	cfg.GoogleClientSecret = ""
	_, err := app.New(context.Background(), cfg, app.Deps{Verifier: idVerifier{}, Store: controlplane.NewMemory(), Google: hostedtest.New()})
	if err == nil || !strings.Contains(err.Error(), "CONTROL_PLANE_ENCRYPTION_KEY") || !strings.Contains(err.Error(), "GOOGLE_CLIENT_SECRET") {
		t.Fatalf("want missing-config error naming both vars, got %v", err)
	}
	cfg = hostedConfig()
	cfg.CookieSecure = false
	if _, err := app.New(context.Background(), cfg, app.Deps{Verifier: idVerifier{}, Store: controlplane.NewMemory(), Google: hostedtest.New()}); err == nil {
		t.Fatal("hosted mode must refuse insecure cookies")
	}
}

func TestHosted_AnonymousIsRejected(t *testing.T) {
	e := newHostedEnv(t)
	b := e.browser()
	for _, p := range []string{"/api/hosted/me", "/api/data", "/api/shares"} {
		if code, _ := b.do("GET", p, nil); code != 401 {
			t.Errorf("GET %s anonymously: want 401, got %d", p, code)
		}
	}
	if code, _ := b.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("a@x.com"), "name": "x"}); code != 401 {
		t.Errorf("onboard anonymously: want 401, got %d", code)
	}
}

func TestHosted_SignUpAndOnboard(t *testing.T) {
	e := newHostedEnv(t)
	alice := e.browser()
	alice.login("Alice@Example.com")

	if m := alice.me(); m.Email != "alice@example.com" || m.Household != nil {
		t.Fatalf("new sign-up: %+v", m)
	}
	if code, out := alice.do("GET", "/api/data", nil); code != 403 || !strings.Contains(out, "no_household") {
		t.Fatalf("data before onboarding: want 403 no_household, got %d %s", code, out)
	}

	// Drive grant for a different Google account is refused.
	if code, _ := alice.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("mallory@example.com"), "name": "Jains"}); code != 403 {
		t.Errorf("mismatched Google account: want 403, got %d", code)
	}
	// Unticked Drive permission / no offline access.
	if code, _ := alice.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("alice@example.com") + ":noscope", "name": "Jains"}); code != 400 {
		t.Errorf("scope not granted: want 400, got %d", code)
	}
	if code, _ := alice.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("alice@example.com") + ":norefresh", "name": "Jains"}); code != 400 {
		t.Errorf("no refresh token: want 400, got %d", code)
	}
	if code, _ := alice.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("alice@example.com"), "name": strings.Repeat("x", 61)}); code != 400 {
		t.Errorf("over-long name: want 400, got %d", code)
	}

	alice.onboard("alice@example.com", "  The   Jains ")
	m := alice.me()
	if m.Household == nil || m.Household.Name != "The Jains" || m.Household.Role != "owner" || m.Household.Billing.Status != "trialing" || m.Household.SheetURL == "" {
		t.Fatalf("after onboarding: %+v", m.Household)
	}

	// The new spreadsheet was provisioned: every tab, default tax rules.
	hh, _ := e.store.GetHousehold(context.Background(), m.Household.ID)
	sheet := e.google.Sheet(hh.SpreadsheetID)
	for _, tab := range []string{"Members", "MF", "Shares", "ShareAccess", "Config"} {
		if len(sheet.Data[tab]) == 0 {
			t.Errorf("tab %s not created", tab)
		}
	}
	if len(sheet.Data["TaxRules"]) != 3 {
		t.Errorf("want header + 2 seeded tax rule sets, got %d rows", len(sheet.Data["TaxRules"]))
	}
	// The refresh token is stored sealed, never in plaintext.
	if strings.Contains(hh.RefreshTokenEnc, "rt:") {
		t.Error("refresh token stored in plaintext")
	}

	if code, _ := alice.do("POST", "/api/hosted/households", map[string]string{"code": hostedtest.Code("alice@example.com"), "name": "Again"}); code != 409 {
		t.Errorf("second household: want 409, got %d", code)
	}
}

func TestHosted_HouseholdsAreIsolated(t *testing.T) {
	e := newHostedEnv(t)
	alice, bob := e.browser(), e.browser()
	alice.login("alice@x.com")
	alice.onboard("alice@x.com", "Alice's")
	bob.login("bob@x.com")
	bob.onboard("bob@x.com", "Bob's")
	// Guard the harness itself: each browser must keep its own session.
	if alice.me().Email != "alice@x.com" || bob.me().Email != "bob@x.com" {
		t.Fatal("test browsers are sharing cookies")
	}

	if code, out := alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"}); code != 200 {
		t.Fatalf("alice add: %d %s", code, out)
	}
	bob.do("POST", "/api/Members", map[string]string{"id": "m9", "name": "Bharat"})

	if got := alice.memberNames(); len(got) != 1 || got[0] != "Asha" {
		t.Errorf("alice sees %v", got)
	}
	if got := bob.memberNames(); len(got) != 1 || got[0] != "Bharat" {
		t.Errorf("bob sees %v", got)
	}
	// Bob can't touch Alice's row even knowing its ID.
	bob.do("PUT", "/api/Members/m1", map[string]string{"name": "pwned"})
	bob.do("DELETE", "/api/Members/m1", nil)
	if got := alice.memberNames(); len(got) != 1 || got[0] != "Asha" {
		t.Errorf("bob modified alice's data: %v", got)
	}
}

func TestHosted_InvitesAndMembers(t *testing.T) {
	e := newHostedEnv(t)
	alice, carol, dave := e.browser(), e.browser(), e.browser()
	alice.login("alice@x.com")
	alice.onboard("alice@x.com", "Jains")
	alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"})

	var inv struct {
		LinkPath string `json:"link_path"`
		Invite   struct {
			ID string `json:"id"`
		} `json:"invite"`
	}
	if code := alice.json("POST", "/api/hosted/invites", map[string]string{"email": "Carol@X.com"}, &inv); code != 200 || !strings.HasPrefix(inv.LinkPath, "/join#") {
		t.Fatalf("invite: %d %+v", code, inv)
	}
	token := strings.TrimPrefix(inv.LinkPath, "/join#")
	if code, _ := alice.do("POST", "/api/hosted/invites", map[string]string{"email": "not an email"}); code != 400 {
		t.Errorf("bad email: want 400, got %d", code)
	}

	// Someone else holding the link can't use it.
	dave.login("dave@x.com")
	if code, _ := dave.do("POST", "/api/hosted/invites/accept", map[string]string{"token": token}); code != 404 {
		t.Errorf("wrong account accepting: want 404, got %d", code)
	}
	carol.login("carol@x.com")
	if code, out := carol.do("POST", "/api/hosted/invites/accept", map[string]string{"token": token}); code != 200 {
		t.Fatalf("carol accept: %d %s", code, out)
	}
	if code, _ := carol.do("POST", "/api/hosted/invites/accept", map[string]string{"token": token}); code == 200 {
		t.Error("invite reused")
	}

	// Carol sees the household's data as a member, but not the Sheet link.
	if got := carol.memberNames(); len(got) != 1 || got[0] != "Asha" {
		t.Errorf("carol sees %v", got)
	}
	if m := carol.me(); m.Household.Role != "member" || m.Household.SheetURL != "" {
		t.Errorf("carol's view: %+v", m.Household)
	}
	// Members can't manage membership.
	if code, _ := carol.do("POST", "/api/hosted/invites", map[string]string{"email": "eve@x.com"}); code != 403 {
		t.Errorf("member inviting: want 403, got %d", code)
	}
	if code, _ := carol.do("DELETE", "/api/hosted/members/alice@x.com", nil); code != 403 {
		t.Errorf("member removing owner: want 403, got %d", code)
	}
	// The owner can't remove someone outside the household.
	dave.onboard("dave@x.com", "Daves")
	if code, _ := alice.do("DELETE", "/api/hosted/members/dave@x.com", nil); code != 404 {
		t.Errorf("removing another household's owner: want 404, got %d", code)
	}
	if m := dave.me(); m.Household == nil {
		t.Fatal("dave was removed from his own household")
	}

	if code, _ := alice.do("DELETE", "/api/hosted/members/carol@x.com", nil); code != 200 {
		t.Fatalf("remove carol: %d", code)
	}
	if code, _ := carol.do("GET", "/api/data", nil); code != 403 {
		t.Errorf("removed member reading data: want 403, got %d", code)
	}
}

func TestHosted_InviteLimit(t *testing.T) {
	e := newHostedEnv(t)
	alice := e.browser()
	alice.login("alice@x.com")
	alice.onboard("alice@x.com", "Jains")
	for i := 0; i < 5; i++ {
		if code, _ := alice.do("POST", "/api/hosted/invites", map[string]string{"email": string(rune('a'+i)) + "@y.com"}); code != 200 {
			t.Fatalf("invite %d: %d", i, code)
		}
	}
	if code, _ := alice.do("POST", "/api/hosted/invites", map[string]string{"email": "f@y.com"}); code != 409 {
		t.Errorf("7th person: want 409, got %d", code)
	}
}

func TestHosted_FamilyShareRoutesToTheRightHousehold(t *testing.T) {
	e := newHostedEnv(t)
	alice, bob := e.browser(), e.browser()
	alice.login("alice@x.com")
	alice.onboard("alice@x.com", "Alice's")
	alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"})
	bob.login("bob@x.com")
	bob.onboard("bob@x.com", "Bob's")

	var enr struct {
		LinkPath string `json:"link_path"`
		Secret   string `json:"secret"`
	}
	if code := alice.json("POST", "/api/shares", map[string]string{"name": "Mom"}, &enr); code != 200 {
		t.Fatalf("create share: %d", code)
	}
	token := strings.TrimPrefix(enr.LinkPath, "/s#")

	// Bob's session can't see Alice's shares.
	var bobShares []any
	bob.json("GET", "/api/shares", nil, &bobShares)
	if len(bobShares) != 0 {
		t.Errorf("bob sees alice's shares: %v", bobShares)
	}

	mom := e.browser() // no Kosh account at all
	mom.header.Set(share.TokenHeader, token)
	if code, _ := mom.do("GET", "/api/share/report", nil); code != 401 {
		t.Errorf("report before code: want 401, got %d", code)
	}
	code, _ := mom.do("POST", "/api/share/verify", map[string]string{"code": totpNow(t, enr.Secret)})
	if code != 200 {
		t.Fatalf("verify: %d", code)
	}
	var rep struct {
		Recipient string `json:"recipient"`
	}
	if code := mom.json("GET", "/api/share/report", nil, &rep); code != 200 || rep.Recipient != "Mom" {
		t.Fatalf("report: %d %+v", code, rep)
	}
	// The recipient's share cookie grants nothing on app routes.
	if code, _ := mom.do("GET", "/api/data", nil); code != 401 {
		t.Errorf("share session on /api/data: want 401, got %d", code)
	}

	stranger := e.browser()
	stranger.header.Set(share.TokenHeader, "made-up-token")
	if code, out := stranger.do("POST", "/api/share/verify", map[string]string{"code": "123456"}); code != 401 || !strings.Contains(out, "isn't valid") {
		t.Errorf("unknown link: want generic 401, got %d %s", code, out)
	}
}

func TestHosted_RevokedGoogleAccessNeedsReconnect(t *testing.T) {
	e := newHostedEnv(t)
	alice, carol := e.browser(), e.browser()
	alice.login("alice@x.com")
	alice.onboard("alice@x.com", "Jains")
	alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"})

	e.google.RevokeAll("alice@x.com") // e.g. Alice removed Kosh in her Google account
	// The request that discovers it must say so — never render an empty
	// dashboard that looks like her data is gone.
	for i := 0; i < 2; i++ {
		if code, out := alice.do("GET", "/api/data", nil); code != 409 || !strings.Contains(out, "reconnect_required") {
			t.Fatalf("request %d after revocation: want 409 reconnect_required, got %d %s", i, code, out)
		}
	}
	if m := alice.me(); !m.Household.NeedsReconnect {
		t.Error("me should report needs_reconnect")
	}

	// Only the owner can reconnect, and only with their own account.
	carol.login("carol@x.com")
	if code, _ := carol.do("POST", "/api/hosted/reconnect", map[string]string{"code": hostedtest.Code("carol@x.com")}); code != 403 {
		t.Errorf("non-member reconnect: want 403, got %d", code)
	}
	if code, _ := alice.do("POST", "/api/hosted/reconnect", map[string]string{"code": hostedtest.Code("mallory@x.com")}); code != 403 {
		t.Errorf("reconnect with another account: want 403, got %d", code)
	}
	if code, out := alice.do("POST", "/api/hosted/reconnect", map[string]string{"code": hostedtest.Code("alice@x.com")}); code != 200 {
		t.Fatalf("reconnect: %d %s", code, out)
	}
	if got := alice.memberNames(); len(got) != 1 || got[0] != "Asha" {
		t.Errorf("after reconnect alice sees %v", got)
	}
}

func TestHosted_DemoSessionCantUseHostedAPI(t *testing.T) {
	e := newHostedEnv(t)
	demo := e.browser()
	demo.do("POST", "/api/auth/demo-login", nil)
	if code, _ := demo.do("GET", "/api/hosted/me", nil); code != 403 {
		t.Errorf("demo /me: want 403, got %d", code)
	}
	if code, _ := demo.do("GET", "/api/data", nil); code != 200 {
		t.Errorf("demo data: want sample data 200, got %d", code)
	}
}

func TestSelfHost_NoHostedRoutes(t *testing.T) {
	cfg := app.Config{Mode: app.ModeSelfHost, FrontendDist: "/nonexistent", SessionSecret: "s", AllowedEmails: []string{"o@x.com"}}
	a, err := app.New(context.Background(), cfg, app.Deps{Verifier: idVerifier{}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler)
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	c := srv.Client()
	c.Jar = jar
	c.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(`{"credential":"id:o@x.com"}`))
	resp, _ := c.Get(srv.URL + "/api/hosted/me")
	if resp.StatusCode != 404 {
		t.Errorf("self-host /api/hosted/me: want 404, got %d", resp.StatusCode)
	}
	resp, _ = c.Get(srv.URL + "/api/data")
	if resp.StatusCode != 200 {
		t.Errorf("self-host data: want 200, got %d", resp.StatusCode)
	}
	// The allow-list still applies when self-hosted.
	jar2, _ := cookiejar.New(nil)
	c.Jar = jar2
	resp, _ = c.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(`{"credential":"id:stranger@x.com"}`))
	if resp.StatusCode != 403 {
		t.Errorf("self-host stranger login: want 403, got %d", resp.StatusCode)
	}
}

var _ store.SheetsAPI = (*store.FakeSheetsAPI)(nil)

// totpNow computes the current RFC 6238 code for a base32 secret, as the
// recipient's authenticator app would.
func totpNow(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatal(err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(time.Now().Unix()/30))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[off:off+4])&0x7fffffff)%1_000_000)
}
