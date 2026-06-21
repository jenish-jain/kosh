package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kosh/internal/auth"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// stubVerifier is a TokenVerifier that always returns the configured result.
type stubVerifier struct {
	info auth.TokenInfo
	err  error
}

func (s *stubVerifier) Verify(_ context.Context, _ string) (auth.TokenInfo, error) {
	return s.info, s.err
}

// newTestAuth returns an AuthHandler configured for tests:
// secret = "test-secret", one allowed email, no Google clientID, non-secure cookies.
func newTestAuth() *auth.AuthHandler {
	verifier := &stubVerifier{} // not used by demo login tests
	return auth.NewHandler(
		"test-secret",
		"", // no clientID — skips audience check in Login
		[]string{"test@example.com"},
		false, // cookieSecure=false so cookies work over plain HTTP in tests
		verifier,
	)
}

// performDemoLogin calls DemoLogin and returns the kosh_session cookie set by
// the response. The test is failed immediately if no cookie is present.
func performDemoLogin(t *testing.T, a *auth.AuthHandler) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/demo", nil)
	a.DemoLogin(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == "kosh_session" {
			return c
		}
	}
	t.Fatal("DemoLogin did not set kosh_session cookie")
	return nil
}

// ── DemoLogin ─────────────────────────────────────────────────────────────────

func TestDemoLogin_Returns200AndSetsCookie(t *testing.T) {
	a := newTestAuth()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/demo", nil)

	a.DemoLogin(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}

	var found bool
	for _, c := range res.Cookies() {
		if c.Name == "kosh_session" {
			found = true
			if c.Value == "" {
				t.Error("kosh_session cookie value is empty")
			}
		}
	}
	if !found {
		t.Error("response did not contain kosh_session cookie")
	}
}

func TestDemoLogin_ResponseBodyHasDemoFlag(t *testing.T) {
	a := newTestAuth()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/demo", nil)

	a.DemoLogin(w, r)

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}

	if body["email"] != "demo@kosh.local" {
		t.Errorf("email = %q, want %q", body["email"], "demo@kosh.local")
	}
	if body["name"] != "Demo" {
		t.Errorf("name = %q, want %q", body["name"], "Demo")
	}
	if demo, ok := body["demo"].(bool); !ok || !demo {
		t.Errorf("demo field = %v, want true", body["demo"])
	}
}

// ── Me ────────────────────────────────────────────────────────────────────────

func TestMe_Returns401WithNoCookie(t *testing.T) {
	a := newTestAuth()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)

	a.Me(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMe_ReturnsDemoUserAfterDemoLogin(t *testing.T) {
	a := newTestAuth()
	cookie := performDemoLogin(t, a)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r.AddCookie(cookie)

	a.Me(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if body["email"] != "demo@kosh.local" {
		t.Errorf("email = %q, want %q", body["email"], "demo@kosh.local")
	}
	if demo, ok := body["demo"].(bool); !ok || !demo {
		t.Errorf("demo = %v, want true", body["demo"])
	}
}

func TestMe_Returns401ForTamperedToken(t *testing.T) {
	a := newTestAuth()
	cookie := performDemoLogin(t, a)

	// Tamper: flip the last character of the cookie value to invalidate the HMAC.
	original := cookie.Value
	tampered := original[:len(original)-1] + "X"
	if tampered == original {
		// In the unlikely event the last char was already 'X', change to 'Y'.
		tampered = original[:len(original)-1] + "Y"
	}
	cookie.Value = tampered

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r.AddCookie(cookie)

	a.Me(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for tampered token", w.Code)
	}
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestLogout_SetsExpiredCookie(t *testing.T) {
	a := newTestAuth()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	a.Logout(w, r)

	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "kosh_session" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("kosh_session MaxAge = %d, want -1", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("response did not contain kosh_session cookie")
	}
}

func TestLogout_ReturnsOK(t *testing.T) {
	a := newTestAuth()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	a.Logout(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if body["ok"] != "true" {
		t.Errorf("body[\"ok\"] = %q, want %q", body["ok"], "true")
	}
}

// ── IsDemoSession ─────────────────────────────────────────────────────────────

func TestIsDemoSession_TrueForDemoSession(t *testing.T) {
	a := newTestAuth()
	cookie := performDemoLogin(t, a)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)

	if !a.IsDemoSession(r) {
		t.Error("IsDemoSession = false, want true after DemoLogin")
	}
}

func TestIsDemoSession_FalseForNoCookie(t *testing.T) {
	a := newTestAuth()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if a.IsDemoSession(r) {
		t.Error("IsDemoSession = true, want false when no cookie present")
	}
}

// ── Require middleware ────────────────────────────────────────────────────────

func TestRequire_AllowsValidSession(t *testing.T) {
	a := newTestAuth()
	cookie := performDemoLogin(t, a)

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.AddCookie(cookie)

	a.Require(next)(w, r)

	if !reached {
		t.Error("next handler was not called for valid session")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequire_Blocks401ForNoSession(t *testing.T) {
	a := newTestAuth()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called with no session")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)

	a.Require(next)(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequire_Blocks401ForInvalidToken(t *testing.T) {
	a := newTestAuth()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called with invalid token")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// Provide a structurally valid but cryptographically invalid token.
	r.AddCookie(&http.Cookie{
		Name:  "kosh_session",
		Value: strings.Repeat("a", 20) + "|" + strings.Repeat("b", 10) + "|" + strings.Repeat("c", 43),
	})

	a.Require(next)(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
