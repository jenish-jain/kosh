package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// demoEmail is the sentinel "identity" used for try-the-demo sessions — never
// a real Google account, so it can never collide with the allow-list.
const demoEmail = "demo@kosh.local"

// AuthHandler handles Google Sign-In verification and session management.
// When allowedEmails is empty, auth is considered disabled.
type AuthHandler struct {
	session       *Session
	verifier      TokenVerifier
	clientID      string
	allowedEmails map[string]bool
	cookieSecure  bool
	aiEnabled     bool
}

// NewHandler constructs an AuthHandler.
func NewHandler(secret, clientID string, allowedEmails []string, cookieSecure bool, verifier TokenVerifier) *AuthHandler {
	allowed := make(map[string]bool, len(allowedEmails))
	for _, e := range allowedEmails {
		if e = strings.TrimSpace(strings.ToLower(e)); e != "" {
			allowed[e] = true
		}
	}
	return &AuthHandler{
		session:       NewSession(secret),
		verifier:      verifier,
		clientID:      clientID,
		allowedEmails: allowed,
		cookieSecure:  cookieSecure,
	}
}

// SetAIEnabled marks whether the AI advisor feature is available.
func (h *AuthHandler) SetAIEnabled(enabled bool) {
	h.aiEnabled = enabled
}

// Config returns the auth configuration (client ID, whether auth is enabled).
// This is served without authentication so the frontend knows how to render the
// login page.
func (h *AuthHandler) Config(w http.ResponseWriter, r *http.Request) {
	cfg := map[string]interface{}{"enabled": true, "ai_enabled": h.aiEnabled}
	if h.clientID != "" {
		cfg["client_id"] = h.clientID
		cfg["demo"] = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// Login verifies a Google ID token, checks the email against the allow-list,
// and sets an HMAC-signed session cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Credential == "" {
		http.Error(w, "missing credential", http.StatusBadRequest)
		return
	}

	info, err := h.verifier.Verify(r.Context(), body.Credential)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Verify the token was issued for this app's client ID.
	if h.clientID != "" && info.Aud != h.clientID {
		http.Error(w, "token audience mismatch", http.StatusUnauthorized)
		return
	}

	email := strings.ToLower(info.Email)
	if !h.allowedEmails[email] {
		http.Error(w, fmt.Sprintf("%s is not on the Kosh access list", info.Email), http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "kosh_session",
		Value:    h.session.Sign(email),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})
	writeJSON(w, map[string]string{"email": info.Email, "name": info.Name, "picture": info.Picture})
}

// DemoLogin sets a session cookie for the sentinel demo identity, letting
// visitors explore the app with sample data — no Google account required.
func (h *AuthHandler) DemoLogin(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "kosh_session",
		Value:    h.session.Sign(demoEmail),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 1 day
	})
	writeJSON(w, map[string]interface{}{"email": demoEmail, "name": "Demo", "demo": true})
}

// Me returns the current user's email from the session cookie, or 401.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	email, ok := h.emailFromCookie(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if email == demoEmail {
		writeJSON(w, map[string]interface{}{"email": email, "name": "Demo", "demo": true})
		return
	}
	writeJSON(w, map[string]string{"email": email})
}

// Logout clears the session cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "kosh_session", Value: "", Path: "/", Secure: h.cookieSecure, MaxAge: -1})
	writeJSON(w, map[string]string{"ok": "true"})
}

// Require is middleware that rejects requests without a valid session cookie.
func (h *AuthHandler) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.emailFromCookie(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// IsDemoSession reports whether the request carries a valid demo session.
func (h *AuthHandler) IsDemoSession(r *http.Request) bool {
	email, ok := h.emailFromCookie(r)
	return ok && email == demoEmail
}

func (h *AuthHandler) emailFromCookie(r *http.Request) (string, bool) {
	c, err := r.Cookie("kosh_session")
	if err != nil {
		return "", false
	}
	email, err := h.session.Verify(c.Value)
	if err != nil {
		return "", false
	}
	return email, true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
