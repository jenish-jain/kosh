package api

import (
	"net/http"
	"os"
	"strings"

	"kosh/internal/ai"
	"kosh/internal/auth"
	"kosh/internal/documents"
)

// cors wraps a handler with CORS headers + OPTIONS pre-flight handling.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			return
		}
		next(w, r)
	}
}

// content security policy — only whitelists what Kosh actually needs.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://accounts.google.com; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self' https://oauth2.googleapis.com https://www.googleapis.com https://accounts.google.com; " +
	"frame-src https://accounts.google.com; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders adds basic security response headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		// Only advertise HSTS over HTTPS — platforms like Cloud Run terminate TLS
		// at their edge proxy, so r.TLS is nil and we must check the forwarded-proto
		// header instead.
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// NewServer builds and returns the HTTP mux with all routes registered.
// Pass aiH=nil when AI is disabled — the /api/chat route is simply not registered.
func NewServer(
	authH        *auth.AuthHandler,
	apiH         *Handler,
	docH         *documents.UploadHandler,
	aiH          *ai.Handler,
	taxH         *ai.TaxHandler,
	taxRulesH    *ai.TaxRulesHandler,
	frontendDist string,
) http.Handler {
	mux := http.NewServeMux()

	protect := func(h http.HandlerFunc) http.HandlerFunc {
		if authH != nil {
			return cors(authH.Require(h))
		}
		return cors(h)
	}

	// ── Auth routes (no auth required) ───────────────────────────────────────
	mux.HandleFunc("/api/auth/config", cors(authH.Config))
	mux.HandleFunc("/api/auth/login", cors(authH.Login))
	mux.HandleFunc("/api/auth/demo-login", cors(authH.DemoLogin))
	mux.HandleFunc("/api/auth/me", cors(authH.Me))
	mux.HandleFunc("/api/auth/logout", cors(authH.Logout))

	// ── Health check (no auth required) ─────────────────────────────────────
	mux.HandleFunc("/api/health", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))

	// ── Protected routes ─────────────────────────────────────────────────────
	mux.HandleFunc("/api/upload/", protect(docH.Handle))
	mux.HandleFunc("/api/data", protect(apiH.GetData))
	mux.HandleFunc("/api/report/export", protect(apiH.ExportReport))
	if aiH != nil {
		mux.HandleFunc("/api/chat", protect(aiH.Handle))
	}
	if taxH != nil {
		mux.HandleFunc("/api/tax/recommendations/generate", protect(taxH.Generate))
	}
	if taxRulesH != nil {
		mux.HandleFunc("/api/tax/rules/propose", protect(taxRulesH.Propose))
		mux.HandleFunc("/api/tax/rules/", protect(taxRulesH.Route))
	}
	mux.HandleFunc("/api/", protect(apiH.Mutations))

	// ── Frontend static files ────────────────────────────────────────────────
	fs := http.FileServer(http.Dir(frontendDist))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := frontendDist + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.ServeFile(w, r, frontendDist+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	return securityHeaders(mux)
}
