package share

import (
	"bytes"
	"context"
	"errors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kosh/internal/auth"
	"kosh/internal/models"
	"kosh/internal/store"
	"kosh/internal/tenant"
)

const testSecret = "test-session-secret"

type fixture struct {
	h      *Handler
	fake   *store.FakeSheetsAPI
	clock  time.Time
	isDemo bool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	fake := store.NewFakeSheetsAPI()
	tn := tenant.New("default", fake)
	repos := tn.Repos
	for _, r := range []store.AnyRepository{repos.Shares, repos.ShareAccess} {
		header := make([]interface{}, len(r.Columns()))
		for i, c := range r.Columns() {
			header[i] = c
		}
		fake.Data[r.Sheet()] = [][]interface{}{header}
	}
	sealer, err := NewSealer(strings.Repeat("k", MinKeyLength))
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{fake: fake, clock: time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)}
	data := func(*tenant.Tenant) (*models.Data, error) {
		return &models.Data{Members: []models.Member{{ID: "m1", Name: "Asha"}}}, nil
	}
	f.h = NewHandler(tenant.Single(tn), sealer, testSecret, data, func(*http.Request) bool { return f.isDemo }, false)
	f.h.now = func() time.Time { return f.clock }
	return f
}

func (f *fixture) do(method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	return f.doToken(method, path, "", body, cookies...)
}

// doToken sends token in the share-token header, as the recipient page does.
func (f *fixture) doToken(method, path, token string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set(TokenHeader, token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	if strings.HasPrefix(path, "/api/shares") {
		f.h.Owner(rec, req)
	} else {
		f.h.Recipient(rec, req)
	}
	return rec
}

// enrol creates a recipient and returns the link token and TOTP secret.
func (f *fixture) enrol(t *testing.T, name string) (id, token, secret string) {
	t.Helper()
	rec := f.do("POST", "/api/shares", map[string]string{"name": name})
	if rec.Code != 200 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var enr enrolment
	json.NewDecoder(rec.Body).Decode(&enr)
	return enr.Share.ID, strings.TrimPrefix(enr.LinkPath, "/s#"), enr.Secret
}

func (f *fixture) code(secret string) string {
	key, _ := b32.DecodeString(secret)
	return hotp(key, f.clock.Unix()/totpStep)
}

func (f *fixture) verify(token, code string) *httptest.ResponseRecorder {
	return f.doToken("POST", "/api/share/verify", token, map[string]string{"code": code})
}

func shareCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	t.Fatalf("no %s cookie set (status %d: %s)", cookieName, rec.Code, rec.Body)
	return nil
}

func (f *fixture) report(token string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	return f.doToken("GET", "/api/share/report", token, nil, cookies...)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCreate_StoresOnlyHashAndCiphertext(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	for _, row := range f.fake.Data["Shares"][1:] {
		for _, cell := range row {
			s, _ := cell.(string)
			if s == token || s == secret {
				t.Fatalf("raw token/secret persisted in Shares tab: %v", row)
			}
		}
	}
}

func TestList_NeverExposesSecrets(t *testing.T) {
	f := newFixture(t)
	f.enrol(t, "Mom")
	body := f.do("GET", "/api/shares", nil).Body.String()
	for _, leaked := range []string{"token_hash", "totp_secret", "secret"} {
		if strings.Contains(body, leaked) {
			t.Errorf("list response contains %q: %s", leaked, body)
		}
	}
}

func TestVerify_ThenReport(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")

	if rec := f.report(token); rec.Code != 401 {
		t.Fatalf("report before verify: want 401, got %d", rec.Code)
	}
	rec := f.verify(token, f.code(secret))
	if rec.Code != 200 {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	c := shareCookie(t, rec)
	if c.Path != cookiePath || !c.HttpOnly || c.MaxAge != int(sessionTTL.Seconds()) {
		t.Errorf("unexpected cookie attributes: %+v", c)
	}
	rep := f.report(token, c)
	if rep.Code != 200 {
		t.Fatalf("report: %d %s", rep.Code, rep.Body)
	}
	var got sharedReport
	json.NewDecoder(rep.Body).Decode(&got)
	if got.Recipient != "Mom" || got.Scope != "Whole family" || len(got.Tables) == 0 {
		t.Errorf("unexpected report: %+v", got)
	}
}

func TestVerify_RejectsReplayedCode(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	code := f.code(secret)
	if rec := f.verify(token, code); rec.Code != 200 {
		t.Fatalf("first verify: %d", rec.Code)
	}
	if rec := f.verify(token, code); rec.Code != 401 {
		t.Fatalf("replayed code: want 401, got %d", rec.Code)
	}
}

func TestVerify_LocksOutAfterMaxAttempts(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	for i := 1; i < maxAttempts; i++ {
		if rec := f.verify(token, "000000"); rec.Code != 401 {
			t.Fatalf("attempt %d: want 401, got %d", i, rec.Code)
		}
	}
	if rec := f.verify(token, "000000"); rec.Code != 429 {
		t.Fatalf("attempt %d: want 429 lockout, got %d", maxAttempts, rec.Code)
	}
	// Even the right code is refused while locked.
	if rec := f.verify(token, f.code(secret)); rec.Code != 429 {
		t.Fatalf("correct code while locked: want 429, got %d", rec.Code)
	}
	f.clock = f.clock.Add(lockout + time.Minute)
	if rec := f.verify(token, f.code(secret)); rec.Code != 200 {
		t.Fatalf("after lockout expires: want 200, got %d %s", rec.Code, rec.Body)
	}
}

func TestVerify_UnknownTokenIsGeneric(t *testing.T) {
	f := newFixture(t)
	rec := f.verify("not-a-real-token", "123456")
	if rec.Code != 401 || !strings.Contains(rec.Body.String(), "isn't valid") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestRevoke_BlocksVerifyAndExistingSession(t *testing.T) {
	f := newFixture(t)
	id, token, secret := f.enrol(t, "Mom")
	c := shareCookie(t, f.verify(token, f.code(secret)))

	if rec := f.do("POST", "/api/shares/"+id+"/revoke", nil); rec.Code != 200 {
		t.Fatalf("revoke: %d", rec.Code)
	}
	if rec := f.report(token, c); rec.Code != 401 {
		t.Errorf("report with pre-revoke cookie: want 401, got %d", rec.Code)
	}
	f.clock = f.clock.Add(time.Minute)
	if rec := f.verify(token, f.code(secret)); rec.Code != 401 {
		t.Errorf("verify after revoke: want 401, got %d", rec.Code)
	}
}

func TestReset_InvalidatesOldLinkAndSecret(t *testing.T) {
	f := newFixture(t)
	id, oldToken, oldSecret := f.enrol(t, "Mom")
	rec := f.do("POST", "/api/shares/"+id+"/reset", nil)
	var enr enrolment
	json.NewDecoder(rec.Body).Decode(&enr)
	newToken := strings.TrimPrefix(enr.LinkPath, "/s#")

	if r := f.verify(oldToken, f.code(oldSecret)); r.Code != 401 {
		t.Errorf("old link after reset: want 401, got %d", r.Code)
	}
	if r := f.verify(newToken, f.code(oldSecret)); r.Code != 401 {
		t.Errorf("old secret on new link: want 401, got %d", r.Code)
	}
	if r := f.verify(newToken, f.code(enr.Secret)); r.Code != 200 {
		t.Errorf("new link + new secret: want 200, got %d", r.Code)
	}
}

func TestExpiry(t *testing.T) {
	f := newFixture(t)
	rec := f.do("POST", "/api/shares", map[string]string{"name": "Dad", "expires": "2026-11-07"})
	var enr enrolment
	json.NewDecoder(rec.Body).Decode(&enr)
	token := strings.TrimPrefix(enr.LinkPath, "/s#")

	if r := f.verify(token, f.code(enr.Secret)); r.Code != 200 {
		t.Fatalf("on expiry day: want 200, got %d", r.Code)
	}
	f.clock = time.Date(2026, 11, 8, 0, 0, 1, 0, time.UTC)
	if r := f.verify(token, f.code(enr.Secret)); r.Code != 401 {
		t.Fatalf("day after expiry: want 401, got %d", r.Code)
	}
}

func TestSession_ExpiresAfterOneDay(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	c := shareCookie(t, f.verify(token, f.code(secret)))
	// The cookie's MaxAge handles browsers; the signed expiry handles a
	// cookie value replayed past it. Re-sign with an expiry in the past.
	c.Value = f.h.session.SignFor(mustVerify(t, f, c), -time.Hour)
	if rec := f.report(token, c); rec.Code != 401 {
		t.Fatalf("expired session: want 401, got %d", rec.Code)
	}
}

func mustVerify(t *testing.T, f *fixture, c *http.Cookie) string {
	t.Helper()
	id, err := f.h.session.Verify(c.Value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSession_BoundToRecipient(t *testing.T) {
	f := newFixture(t)
	_, momToken, momSecret := f.enrol(t, "Mom")
	_, dadToken, _ := f.enrol(t, "Dad")
	c := shareCookie(t, f.verify(momToken, f.code(momSecret)))
	if rec := f.report(dadToken, c); rec.Code != 401 {
		t.Fatalf("Mom's session on Dad's link: want 401, got %d", rec.Code)
	}
}

func TestSession_KeySeparatedFromOwnerSessions(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	shareVal := shareCookie(t, f.verify(token, f.code(secret))).Value

	// A share cookie must not verify as an owner session...
	if _, err := auth.NewSession(testSecret).Verify(shareVal); err == nil {
		t.Error("share session token accepted by owner session verifier")
	}
	// ...and an owner session must not work as a share session.
	owner := &http.Cookie{Name: cookieName, Value: auth.NewSession(testSecret).Sign(mustVerify(t, f, &http.Cookie{Value: shareVal}))}
	if rec := f.report(token, owner); rec.Code != 401 {
		t.Errorf("owner-signed token accepted as share session: %d", rec.Code)
	}
}

func TestOwner_BlockedInDemo(t *testing.T) {
	f := newFixture(t)
	f.isDemo = true
	if rec := f.do("GET", "/api/shares", nil); rec.Code != 403 {
		t.Fatalf("demo list: want 403, got %d", rec.Code)
	}
}

func TestAccessLog(t *testing.T) {
	f := newFixture(t)
	id, token, secret := f.enrol(t, "Mom")
	f.verify(token, "000000")
	f.verify(token, f.code(secret))
	var logs []models.ShareAccess
	json.NewDecoder(f.do("GET", "/api/shares/"+id+"/access", nil).Body).Decode(&logs)
	events := map[string]bool{}
	for _, l := range logs {
		events[l.Event] = true
	}
	if !events["bad_code"] || !events["verified"] {
		t.Fatalf("expected bad_code + verified events, got %+v", logs)
	}
}

func TestReport_IgnoresTokenInQueryString(t *testing.T) {
	f := newFixture(t)
	_, token, secret := f.enrol(t, "Mom")
	c := shareCookie(t, f.verify(token, f.code(secret)))
	// The old ?token= form must not work: tokens stay out of URLs and logs.
	if rec := f.do("GET", "/api/share/report?token="+token, nil, c); rec.Code != 401 {
		t.Fatalf("query-string token: want 401, got %d", rec.Code)
	}
}

func TestLinkPath_KeepsTokenInFragment(t *testing.T) {
	f := newFixture(t)
	rec := f.do("POST", "/api/shares", map[string]string{"name": "Mom"})
	var enr enrolment
	json.NewDecoder(rec.Body).Decode(&enr)
	if !strings.HasPrefix(enr.LinkPath, "/s#") {
		t.Fatalf("link should carry the token in the fragment, got %q", enr.LinkPath)
	}
}

// fakeIndex records the share-token index the hosted service keeps.
type fakeIndex struct {
	m    map[string]string
	fail bool
}

func (x *fakeIndex) PutShareToken(_ context.Context, hash, tenantID string) error {
	if x.fail {
		return errors.New("index down")
	}
	x.m[hash] = tenantID
	return nil
}

func (x *fakeIndex) DeleteShareToken(_ context.Context, hash string) error {
	delete(x.m, hash)
	return nil
}

func TestTokenIndex_TracksCreateResetAndDelete(t *testing.T) {
	f := newFixture(t)
	idx := &fakeIndex{m: map[string]string{}}
	f.h.SetTokenIndex(idx)

	id, token, _ := f.enrol(t, "Mom")
	if idx.m[HashToken(token)] != "default" || len(idx.m) != 1 {
		t.Fatalf("create should index the token under the tenant: %+v", idx.m)
	}
	rec := f.do("POST", "/api/shares/"+id+"/reset", nil)
	var enr enrolment
	json.NewDecoder(rec.Body).Decode(&enr)
	newToken := strings.TrimPrefix(enr.LinkPath, "/s#")
	if _, stale := idx.m[HashToken(token)]; stale || idx.m[HashToken(newToken)] != "default" || len(idx.m) != 1 {
		t.Fatalf("reset should swap the indexed token: %+v", idx.m)
	}
	f.do("DELETE", "/api/shares/"+id, nil)
	if len(idx.m) != 0 {
		t.Fatalf("delete should drop the index entry: %+v", idx.m)
	}
}

func TestTokenIndex_FailureBlocksCreate(t *testing.T) {
	f := newFixture(t)
	f.h.SetTokenIndex(&fakeIndex{m: map[string]string{}, fail: true})
	if rec := f.do("POST", "/api/shares", map[string]string{"name": "Mom"}); rec.Code != 500 {
		t.Fatalf("want 500 when the index can't be written, got %d", rec.Code)
	}
	if len(f.fake.Data["Shares"]) != 1 {
		t.Fatal("a share row was written even though its link could never resolve")
	}
}
