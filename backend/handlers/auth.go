package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthHandler handles Google Sign-In verification and session management.
// When allowedEmails is empty, auth is considered disabled.
type AuthHandler struct {
	allowedEmails map[string]bool
	clientID      string
	secret        string
}

func NewAuthHandler(allowedEmails []string, clientID, secret string) *AuthHandler {
	m := map[string]bool{}
	for _, e := range allowedEmails {
		if e = strings.TrimSpace(strings.ToLower(e)); e != "" {
			m[e] = true
		}
	}
	return &AuthHandler{allowedEmails: m, clientID: clientID, secret: secret}
}

// Config returns whether auth is enabled and the Google client ID for the frontend.
func (a *AuthHandler) Config(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"enabled":   true,
		"client_id": a.clientID,
	})
}

// Login verifies a Google ID token, checks the email against the allow-list,
// and sets an HMAC-signed session cookie.
func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Credential == "" {
		http.Error(w, "missing credential", http.StatusBadRequest)
		return
	}

	// Verify the Google ID token via Google's tokeninfo endpoint.
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + body.Credential)
	if err != nil {
		http.Error(w, "could not reach Google to verify token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var info struct {
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Aud           string `json:"aud"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Error         string `json:"error"`
	}
	json.Unmarshal(raw, &info)

	if info.Error != "" || resp.StatusCode != 200 {
		http.Error(w, "invalid Google token", http.StatusUnauthorized)
		return
	}
	if info.EmailVerified != "true" {
		http.Error(w, "Google account email not verified", http.StatusUnauthorized)
		return
	}
	// Verify the token was issued for this app's client ID.
	if a.clientID != "" && info.Aud != a.clientID {
		http.Error(w, "token audience mismatch", http.StatusUnauthorized)
		return
	}

	email := strings.ToLower(info.Email)
	if !a.allowedEmails[email] {
		http.Error(w, fmt.Sprintf("%s is not on the Kosh access list", info.Email), http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "kosh_session",
		Value:    a.signToken(email),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})
	writeJSON(w, map[string]string{"email": info.Email, "name": info.Name, "picture": info.Picture})
}

// Me returns the current user's email from the session cookie, or 401.
func (a *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	email, ok := a.emailFromCookie(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"email": email})
}

// Logout clears the session cookie.
func (a *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "kosh_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, map[string]string{"ok": "true"})
}

// Require is middleware that rejects requests without a valid session cookie.
func (a *AuthHandler) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.emailFromCookie(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *AuthHandler) emailFromCookie(r *http.Request) (string, bool) {
	c, err := r.Cookie("kosh_session")
	if err != nil {
		return "", false
	}
	return a.verifyToken(c.Value)
}

// Token format: base64(email) | base64(expiry_unix) | hmac(base64(email)|base64(expiry), secret)
func (a *AuthHandler) signToken(email string) string {
	exp := fmt.Sprint(time.Now().Add(30 * 24 * time.Hour).Unix())
	payload := base64.RawURLEncoding.EncodeToString([]byte(email)) + "|" +
		base64.RawURLEncoding.EncodeToString([]byte(exp))
	mac := hmac.New(sha256.New, []byte(a.secret))
	mac.Write([]byte(payload))
	return payload + "|" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *AuthHandler) verifyToken(token string) (string, bool) {
	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return "", false
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, []byte(a.secret))
	mac.Write([]byte(payload))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return "", false
	}
	expB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var exp int64
	fmt.Sscan(string(expB), &exp)
	if time.Now().Unix() > exp {
		return "", false // expired
	}
	emailB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	return string(emailB), true
}
