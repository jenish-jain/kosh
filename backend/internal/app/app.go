// Package app assembles the Kosh HTTP server from its configuration. It's
// the wiring main used to do inline, pulled out so tests (and the hosteddev
// build) can build the exact production server with fake dependencies.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"kosh/drive"
	"kosh/internal/ai"
	"kosh/internal/api"
	"kosh/internal/auth"
	"kosh/internal/billing"
	"kosh/internal/controlplane"
	"kosh/internal/documents"
	"kosh/internal/hosted"
	"kosh/internal/share"
	"kosh/internal/store"
	"kosh/internal/tenant"
	sh "kosh/sheets"
)

// Modes.
const (
	ModeSelfHost = "selfhost"
	ModeHosted   = "hosted"
)

// Config is everything read from the environment.
type Config struct {
	Mode           string
	FrontendDist   string
	GoogleClientID string
	SessionSecret  string
	CookieSecure   bool
	ShareKey       string
	AnthropicKey   string
	PromptsDir     string
	AIEnabled      bool
	AIProvider     string
	OllamaURL      string
	OllamaModel    string

	// Self-host only.
	SpreadsheetID   string
	CredentialsPath string
	AllowedEmails   []string

	// Hosted only.
	GoogleClientSecret string
	ControlPlaneKey    string
	FirestoreProject   string
	FirestoreDatabase  string
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	RazorpayPlanMonthly   string
	RazorpayPlanAnnual    string

	// PublicConfig is merged into /api/auth/config (the hosteddev build uses
	// it to tell the frontend to show its dev login).
	PublicConfig map[string]any
}

// ConfigFromEnv reads Config from environment variables.
func ConfigFromEnv() Config {
	return Config{
		Mode:               sh.EnvOrDefault("KOSH_MODE", ModeSelfHost),
		FrontendDist:       sh.EnvOrDefault("FRONTEND_DIST", "../frontend/dist"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		CookieSecure:       sh.EnvOrDefault("COOKIE_SECURE", "true") != "false",
		ShareKey:           os.Getenv("SHARE_ENCRYPTION_KEY"),
		AnthropicKey:       os.Getenv("ANTHROPIC_API_KEY"),
		PromptsDir:         sh.EnvOrDefault("PROMPTS_DIR", "prompts"),
		AIEnabled:          sh.EnvOrDefault("AI_ENABLED", "false") == "true",
		AIProvider:         sh.EnvOrDefault("AI_PROVIDER", "ollama"),
		OllamaURL:          sh.EnvOrDefault("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:        sh.EnvOrDefault("OLLAMA_MODEL", "llama3.1"),
		SpreadsheetID:      os.Getenv("SPREADSHEET_ID"),
		CredentialsPath:    sh.EnvOrDefault("CREDENTIALS_PATH", "credentials.json"),
		AllowedEmails:      strings.Split(os.Getenv("ALLOWED_EMAILS"), ","),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		ControlPlaneKey:    os.Getenv("CONTROL_PLANE_ENCRYPTION_KEY"),
		FirestoreProject:   sh.EnvOrDefault("FIRESTORE_PROJECT", os.Getenv("GOOGLE_CLOUD_PROJECT")),
		FirestoreDatabase:  sh.EnvOrDefault("FIRESTORE_DATABASE", "(default)"),
		RazorpayKeyID:         os.Getenv("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     os.Getenv("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		RazorpayPlanMonthly:   os.Getenv("RAZORPAY_PLAN_MONTHLY"),
		RazorpayPlanAnnual:    os.Getenv("RAZORPAY_PLAN_ANNUAL"),
	}
}

// Deps are external services; nil fields get their real implementation.
type Deps struct {
	Verifier   auth.TokenVerifier
	Store      controlplane.Store // hosted
	Google     hosted.Google      // hosted
	Gateway    billing.Gateway    // hosted
	AIProvider ai.Provider
}

// App is a built server.
type App struct {
	Handler http.Handler
	// Summary is a human-readable startup line for main to print.
	Summary string
}

// New builds the server. Hosted mode fails fast on missing configuration
// rather than starting half-secured.
func New(ctx context.Context, cfg Config, deps Deps) (*App, error) {
	if deps.Verifier == nil {
		deps.Verifier = auth.NewGoogleTokenVerifier(http.DefaultClient)
	}
	switch cfg.Mode {
	case ModeSelfHost, ModeHosted:
	default:
		return nil, fmt.Errorf("KOSH_MODE must be %q or %q, got %q", ModeSelfHost, ModeHosted, cfg.Mode)
	}
	hostedMode := cfg.Mode == ModeHosted
	if hostedMode {
		if err := validateHosted(cfg); err != nil {
			return nil, err
		}
	}

	// ── Auth ─────────────────────────────────────────────────────────────────
	authH := auth.NewHandler(cfg.SessionSecret, cfg.GoogleClientID, cfg.AllowedEmails, cfg.CookieSecure, deps.Verifier)
	authH.SetConfigValue("mode", cfg.Mode)
	for k, v := range cfg.PublicConfig {
		authH.SetConfigValue(k, v)
	}
	isDemo := func(r *http.Request) bool { return authH.IsDemoSession(r) }
	switch {
	case hostedMode:
		authH.SetOpenSignup(true)
	case cfg.GoogleClientID != "" && cfg.SessionSecret != "":
		log.Printf("✓ Auth enabled — %d allowed email(s)", len(cfg.AllowedEmails))
	default:
		log.Println("⚠ Auth disabled (GOOGLE_CLIENT_ID / SESSION_SECRET not set)")
	}

	// ── Data source: one household (self-host) or many (hosted) ──────────────
	var (
		tenants  tenant.Resolver
		devMode  bool
		summary  string
		hostedH  *hosted.Handler
		billingH *billing.Handler
		cpStore  controlplane.Store
	)
	if hostedMode {
		var err error
		cpStore = deps.Store
		if cpStore == nil {
			if cpStore, err = controlplane.NewFirestore(ctx, cfg.FirestoreProject, cfg.FirestoreDatabase); err != nil {
				return nil, err
			}
		}
		google := deps.Google
		if google == nil {
			google = hosted.NewRealGoogle(cfg.GoogleClientID, cfg.GoogleClientSecret, deps.Verifier, http.DefaultClient)
		}
		sealer, err := controlplane.NewTokenSealer(cfg.ControlPlaneKey)
		if err != nil {
			return nil, err
		}
		resolver := hosted.NewResolver(cpStore, sealer, google, authH.SessionEmail)
		tenants = resolver
		hostedH = hosted.NewHandler(cpStore, sealer, google, resolver, authH.SessionEmail, isDemo)
		gw := deps.Gateway
		if gw == nil {
			gw = billing.NewRazorpay(cfg.RazorpayKeyID, cfg.RazorpayKeySecret, cfg.RazorpayWebhookSecret, http.DefaultClient)
		}
		billingH, err = billing.NewHandler(ctx, cpStore, gw, map[string]string{
			billing.PlanMonthly: cfg.RazorpayPlanMonthly, billing.PlanAnnual: cfg.RazorpayPlanAnnual,
		}, authH.SessionEmail, isDemo)
		if err != nil {
			return nil, err
		}
		hostedH.SetSubscriptionCanceller(billingH.CancelNow)
		summary = "Mode: hosted (multi-household, Firestore control plane)"
	} else {
		var t *tenant.Tenant
		t, devMode = selfHostTenant(cfg)
		tenants = tenant.Single(t)
		summary = "Mode: live (reading/writing Google Sheets)"
		if devMode {
			summary = "Mode: dev (serving dev_data.json — mutations are no-ops)"
		}
	}

	apiH := api.NewHandler(tenants, devMode, isDemo)
	// Propose doesn't itself call an AI provider (that happens via the existing
	// /api/upload/tax_rules parse step) — it's always available, unlike aiH/taxH.
	taxRulesH := ai.NewTaxRulesHandler(tenants, isDemo, apiH.IsSampleData)

	// ── Document upload handler ──────────────────────────────────────────────
	// Drive uploads use a per-request OAuth token from the frontend (drive.file
	// scope). The factory creates a new Drive client from that token each call.
	driveFactory := func(ctx context.Context, accessToken string) (drive.API, error) {
		return drive.NewClientFromToken(ctx, accessToken)
	}
	var parser documents.Parser
	if cfg.AnthropicKey != "" {
		parser = documents.NewClaudeParser(cfg.AnthropicKey, cfg.PromptsDir, http.DefaultClient)
	}
	docH := documents.NewUploadHandler(driveFactory, parser)

	// ── AI Financial Advisor ─────────────────────────────────────────────────
	var aiH *ai.Handler
	var taxH *ai.TaxHandler
	if provider := aiProvider(cfg, deps); provider != nil {
		aiH = ai.NewHandler(provider, apiH.LoadData, isDemo)
		taxH = ai.NewTaxHandler(provider, tenants, apiH.LoadData, isDemo, apiH.IsSampleData)
		authH.SetAIEnabled(true)
	}

	// ── Family sharing ───────────────────────────────────────────────────────
	// Needs SESSION_SECRET (share sessions are signed with a key derived from
	// it — an empty secret would make them forgeable) and SHARE_ENCRYPTION_KEY
	// (TOTP secrets are encrypted at rest in the Shares tab).
	var shareH *share.Handler
	switch {
	case cfg.ShareKey == "":
		log.Println("⚠ Family sharing disabled (SHARE_ENCRYPTION_KEY not set)")
	case cfg.SessionSecret == "":
		log.Println("⚠ Family sharing disabled (SESSION_SECRET not set)")
	default:
		sealer, err := share.NewSealer(cfg.ShareKey)
		if err != nil {
			log.Printf("⚠ Family sharing disabled: SHARE_ENCRYPTION_KEY must be at least %d characters (try `openssl rand -hex 32`)", share.MinKeyLength)
			break
		}
		shareH = share.NewHandler(tenants, sealer, cfg.SessionSecret, apiH.FamilyData, isDemo, cfg.CookieSecure)
		if hostedMode {
			shareH.SetTokenIndex(cpStore)
		}
		authH.SetShareEnabled(true)
		switch {
		case devMode:
			log.Println("✓ Family sharing enabled (dev mode — shares live in memory and reset on restart)")
		default:
			log.Println("✓ Family sharing enabled")
		}
	}

	var extra func(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc)
	var opts []api.ServerOption
	if hostedH != nil {
		extra = func(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc) {
			mux.HandleFunc("/api/hosted/", protect(hostedH.Routes))
			// The webhook is unauthenticated by design — trusted only once
			// its signature verifies.
			mux.HandleFunc("/api/billing/webhook", billingH.Webhook)
			mux.HandleFunc("/api/billing/", protect(billingH.Routes))
		}
		opts = append(opts,
			api.WithMiddleware(billingH.Enforce),
			api.WithMiddleware(hostedH.Quotas),
			// Razorpay Checkout: its script, its payment iframe, and the
			// endpoints the iframe's parent page talks to.
			api.WithCSPSources("script-src", "https://checkout.razorpay.com"),
			api.WithCSPSources("frame-src", "https://api.razorpay.com", "https://checkout.razorpay.com"),
			api.WithCSPSources("connect-src", "https://api.razorpay.com", "https://lumberjack.razorpay.com"),
			api.WithCSPSources("img-src", "https://*.razorpay.com"),
		)
	}
	handler := api.NewServer(authH, apiH, docH, aiH, taxH, taxRulesH, shareH, extra, cfg.FrontendDist, opts...)
	return &App{Handler: handler, Summary: summary}, nil
}

func validateHosted(cfg Config) error {
	var missing []string
	need := func(name, v string, minLen int) {
		if len(v) < minLen || v == "" {
			if minLen > 1 {
				name += fmt.Sprintf(" (at least %d characters)", minLen)
			}
			missing = append(missing, name)
		}
	}
	need("GOOGLE_CLIENT_ID", cfg.GoogleClientID, 1)
	need("GOOGLE_CLIENT_SECRET", cfg.GoogleClientSecret, 1)
	need("SESSION_SECRET", cfg.SessionSecret, 32)
	need("SHARE_ENCRYPTION_KEY", cfg.ShareKey, share.MinKeyLength)
	need("CONTROL_PLANE_ENCRYPTION_KEY", cfg.ControlPlaneKey, controlplane.MinKeyLength)
	need("RAZORPAY_KEY_ID", cfg.RazorpayKeyID, 1)
	need("RAZORPAY_KEY_SECRET", cfg.RazorpayKeySecret, 1)
	need("RAZORPAY_WEBHOOK_SECRET", cfg.RazorpayWebhookSecret, 16)
	need("RAZORPAY_PLAN_MONTHLY", cfg.RazorpayPlanMonthly, 1)
	need("RAZORPAY_PLAN_ANNUAL", cfg.RazorpayPlanAnnual, 1)
	if len(missing) > 0 {
		return errors.New("hosted mode needs: " + strings.Join(missing, ", "))
	}
	if !cfg.CookieSecure {
		return errors.New("hosted mode requires COOKIE_SECURE=true")
	}
	return nil
}

// selfHostTenant builds the one household a self-hosted deploy serves —
// the configured spreadsheet, or an in-memory fake in dev mode.
func selfHostTenant(cfg Config) (*tenant.Tenant, bool) {
	if cfg.SpreadsheetID == "" {
		log.Println("⚠ SPREADSHEET_ID not set — running in dev mode (dev_data.json)")
		return devTenant(), true
	}
	if _, err := os.Stat(cfg.CredentialsPath); err != nil {
		log.Printf("⚠ %s not found — running in dev mode", cfg.CredentialsPath)
		return devTenant(), true
	}
	c, err := sh.NewClient(cfg.CredentialsPath, cfg.SpreadsheetID)
	if err != nil {
		log.Printf("⚠ Sheets client error: %v — running in dev mode", err)
		return devTenant(), true
	}
	t := tenant.New("default", c)
	log.Printf("✓ Connected to Google Sheets (%s)", cfg.SpreadsheetID)
	fmt.Println("  Verifying tabs…")
	if err := tenant.Provision(t); err != nil {
		log.Printf("⚠ %v", err)
	}
	return t, false
}

// devTenant: data reads are served from dev_data.json, and a fake API backs
// the repositories so they're never nil. Family shares are the one thing
// dev mode persists (in memory), so their tabs get headers.
func devTenant() *tenant.Tenant {
	fake := store.NewFakeSheetsAPI()
	t := tenant.New("default", fake)
	for _, r := range []store.AnyRepository{t.Repos.Shares, t.Repos.ShareAccess} {
		fake.EnsureTabs([]sh.TabDef{{Name: r.Sheet(), Columns: r.Columns()}})
	}
	return t
}

func aiProvider(cfg Config, deps Deps) ai.Provider {
	if deps.AIProvider != nil {
		return deps.AIProvider
	}
	if !cfg.AIEnabled {
		return nil
	}
	switch cfg.AIProvider {
	case "anthropic":
		if cfg.AnthropicKey == "" {
			log.Println("⚠ AI_PROVIDER=anthropic but ANTHROPIC_API_KEY not set — AI disabled")
			return nil
		}
		log.Println("✓ AI advisor enabled — provider: Anthropic")
		return ai.NewAnthropicProvider(cfg.AnthropicKey, "", nil)
	default: // ollama
		log.Printf("✓ AI advisor enabled — provider: Ollama (%s @ %s)", cfg.OllamaModel, cfg.OllamaURL)
		return ai.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel, nil)
	}
}
