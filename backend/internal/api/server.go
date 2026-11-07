package api

import (
	"net/http"
	"os"
	"strings"

	"kosh/internal/ai"
	"kosh/internal/auth"
	"kosh/internal/documents"
	"kosh/internal/share"
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

// cspDirectives is the content security policy — only what Kosh actually
// needs. Modes can add sources via WithCSPSources.
var cspDirectives = []struct {
	name    string
	sources []string
}{
	{"default-src", []string{"'self'"}},
	{"script-src", []string{"'self'", "https://accounts.google.com"}},
	{"style-src", []string{"'self'", "'unsafe-inline'", "https://fonts.googleapis.com"}},
	{"font-src", []string{"'self'", "https://fonts.gstatic.com"}},
	{"img-src", []string{"'self'", "data:"}},
	{"connect-src", []string{"'self'", "https://oauth2.googleapis.com", "https://www.googleapis.com", "https://accounts.google.com"}},
	{"frame-src", []string{"https://accounts.google.com"}},
	{"object-src", []string{"'none'"}},
	{"base-uri", []string{"'self'"}},
	{"form-action", []string{"'self'"}},
	{"frame-ancestors", []string{"'none'"}},
}

func buildCSP(extra map[string][]string) string {
	parts := make([]string, 0, len(cspDirectives))
	for _, d := range cspDirectives {
		parts = append(parts, d.name+" "+strings.Join(append(append([]string{}, d.sources...), extra[d.name]...), " "))
	}
	return strings.Join(parts, "; ")
}

// ServerOption customises NewServer.
type ServerOption func(*serverOptions)

type serverOptions struct {
	csp        map[string][]string
	middleware []func(http.Handler) http.Handler
}

// WithCSPSources adds sources to a CSP directive (e.g. a payment provider's
// checkout script and frame).
func WithCSPSources(directive string, sources ...string) ServerOption {
	return func(o *serverOptions) { o.csp[directive] = append(o.csp[directive], sources...) }
}

// WithMiddleware wraps the routes, inside the security headers.
func WithMiddleware(m func(http.Handler) http.Handler) ServerOption {
	return func(o *serverOptions) { o.middleware = append(o.middleware, m) }
}

// securityHeaders adds basic security response headers to every response.
func securityHeaders(next http.Handler, contentSecurityPolicy string) http.Handler {
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
		// This deployment is fronted by a Netlify reverse-proxy (see
		// netlify/netlify.toml) for the custom domain — without an explicit
		// no-store, a CDN/edge layer (or an intermediate browser cache) can
		// cache a dynamic, per-session /api/ GET response and serve it stale
		// on a later request that hits a different edge node or an expired-
		// but-still-served cache entry. Static frontend assets are excluded
		// so JS/CSS bundles keep benefiting from normal caching.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		// Family-share pages carry the secret link token in the URL: never
		// leak it via Referer, never cache it, never let it be indexed.
		if strings.HasPrefix(r.URL.Path, "/s/") || strings.HasPrefix(r.URL.Path, "/api/share/") {
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cache-Control", "no-store")
			h.Set("X-Robots-Tag", "noindex, nofollow")
		}
		next.ServeHTTP(w, r)
	})
}

// NewServer builds and returns the HTTP mux with all routes registered.
// Pass aiH=nil when AI is disabled — the /api/chat route is simply not registered.
// Likewise shareH=nil when family sharing isn't configured. extra, if set,
// registers mode-specific routes (hosted mode's /api/hosted/*); it receives
// the session-checking wrapper so it can protect them.
func NewServer(
	authH        *auth.AuthHandler,
	apiH         *Handler,
	docH         *documents.UploadHandler,
	aiH          *ai.Handler,
	taxH         *ai.TaxHandler,
	taxRulesH    *ai.TaxRulesHandler,
	shareH       *share.Handler,
	extra        func(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc),
	frontendDist string,
	opts ...ServerOption,
) http.Handler {
	o := serverOptions{csp: map[string][]string{}}
	for _, opt := range opts {
		opt(&o)
	}
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
	if shareH != nil {
		mux.HandleFunc("/api/shares", protect(shareH.Owner))
		mux.HandleFunc("/api/shares/", protect(shareH.Owner))
		// Recipient routes are deliberately outside protect: they're gated
		// by link token + TOTP code, then by the path-scoped share cookie.
		mux.HandleFunc("/api/share/", cors(shareH.Recipient))
	}
	if extra != nil {
		extra(mux, protect)
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

	var h http.Handler = mux
	for i := len(o.middleware) - 1; i >= 0; i-- {
		h = o.middleware[i](h)
	}
	return securityHeaders(h, buildCSP(o.csp))
}
