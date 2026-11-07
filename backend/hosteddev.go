//go:build hosteddev

// Dev-only build for exercising hosted mode locally without Google or real
// payments: `go build -tags hosteddev`. Never built by the Dockerfile.
//
// - Sign in at /app with any email (no Google).
// - "Connect Google Drive" is simulated; spreadsheets are in-memory fakes.
// - POST /api/dev/revoke {"email"} simulates the customer revoking Kosh's
//   Google access, to exercise the reconnect flow.
// - Control plane: in-memory, or the Firestore emulator when
//   FIRESTORE_EMULATOR_HOST is set (exercising the real Firestore store).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"kosh/internal/app"
	"kosh/internal/auth"
	"kosh/internal/controlplane"
	"kosh/internal/hosted/hostedtest"
)

type devVerifier struct{ clientID string }

func (v devVerifier) Verify(_ context.Context, tok string) (auth.TokenInfo, error) {
	email, ok := strings.CutPrefix(tok, "dev:")
	if !ok || !strings.Contains(email, "@") {
		return auth.TokenInfo{}, errors.New("invalid Google token")
	}
	return auth.TokenInfo{Email: email, Aud: v.clientID}, nil
}

func init() {
	devOverrides = func(cfg *app.Config, deps *app.Deps) func(http.Handler) http.Handler {
		log.Println("⚠ HOSTEDDEV BUILD — fake Google sign-in and Drive. Never deploy this binary.")
		cfg.Mode = app.ModeHosted
		dev := strings.Repeat("d", 40)
		for _, p := range []*string{&cfg.SessionSecret, &cfg.ShareKey, &cfg.ControlPlaneKey} {
			if len(*p) < 32 {
				*p = dev
			}
		}
		cfg.GoogleClientID, cfg.GoogleClientSecret = "dev-client", "dev-secret"
		cfg.CookieSecure = true // browsers accept Secure cookies on http://localhost
		cfg.PublicConfig = map[string]any{"dev_login": true}

		google := hostedtest.New()
		deps.Google = google
		deps.Verifier = devVerifier{clientID: cfg.GoogleClientID}
		if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
			cfg.FirestoreProject, cfg.FirestoreDatabase = "kosh-dev", "(default)"
			log.Println("  control plane: Firestore emulator at " + os.Getenv("FIRESTORE_EMULATOR_HOST"))
		} else {
			deps.Store = controlplane.NewMemory()
			log.Println("  control plane: in-memory")
		}

		return func(next http.Handler) http.Handler {
			mux := http.NewServeMux()
			// Signs in through the real /api/auth/login path, with a
			// credential devVerifier accepts.
			mux.HandleFunc("/api/dev/login", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Email string }
				json.NewDecoder(r.Body).Decode(&body)
				cred, _ := json.Marshal(map[string]string{"credential": "dev:" + strings.TrimSpace(body.Email)})
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/api/auth/login"
				r2.Body = ioNopCloser(cred)
				r2.ContentLength = int64(len(cred))
				next.ServeHTTP(w, r2)
			})
			mux.HandleFunc("/api/dev/revoke", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Email string }
				json.NewDecoder(r.Body).Decode(&body)
				google.RevokeAll(controlplane.NormalizeEmail(body.Email))
				w.Write([]byte(`{"status":"ok"}`))
			})
			mux.Handle("/", next)
			return mux
		}
	}
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(b []byte) nopCloser { return nopCloser{bytes.NewReader(b)} }
