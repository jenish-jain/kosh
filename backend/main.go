package main

import (
	"encoding/json"
	"fmt"
	"kosh/handlers"
	sh "kosh/sheets"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // load .env if present; ignore error if not

	port           := sh.EnvOrDefault("PORT", "8080")
	spreadsheetID  := os.Getenv("SPREADSHEET_ID")
	credPath       := sh.EnvOrDefault("CREDENTIALS_PATH", "credentials.json")
	frontendDist   := sh.EnvOrDefault("FRONTEND_DIST", "../frontend/dist")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	sessionSecret  := os.Getenv("SESSION_SECRET")
	allowedRaw     := os.Getenv("ALLOWED_EMAILS") // comma-separated
	anthropicKey   := os.Getenv("ANTHROPIC_API_KEY")
	promptsDir     := sh.EnvOrDefault("PROMPTS_DIR", "prompts")
	cookieSecure   := sh.EnvOrDefault("COOKIE_SECURE", "true") != "false"

	// Tab definitions — order controls creation order in the spreadsheet.
	koshTabs := []sh.TabDef{
		{Name: "Members",   Columns: []string{"id", "name", "full_name", "relation", "slab", "color"}},
		{Name: "MF",        Columns: []string{"id", "name", "plan", "platform", "member", "invested", "current", "sip", "notes"}},
		{Name: "Stocks",    Columns: []string{"id", "name", "ticker", "qty", "avg_price", "last_price", "member"}},
		{Name: "Metals",    Columns: []string{"id", "type", "date_purchased", "grams", "buy_rate", "today_price", "place", "member"}},
		{Name: "Fixed",     Columns: []string{"id", "kind", "name", "member", "principal", "rate", "current_value", "opened", "matures", "monthly"}},
		{Name: "Insurance", Columns: []string{"id", "name", "type", "member", "premium", "freq", "paid", "value", "cover", "maturity", "due_date"}},
		{Name: "Loans",     Columns: []string{"id", "lender", "type", "member", "principal", "outstanding", "rate", "emi", "emi_day", "started", "tenure_months"}},
		{Name: "SIPs",      Columns: []string{"id", "fund", "member", "amount", "day", "status", "start_date", "platform"}},
		{Name: "Lumpsums",  Columns: []string{"id", "fund", "member", "amount", "date"}},
		{Name: "History",   Columns: []string{"month", "value"}},
		{Name: "Config",    Columns: []string{"key", "value"}},
	}

	// ── Sheets client ──────────────────────────────────────────────────────────
	var client *sh.Client
	if spreadsheetID != "" {
		if _, err := os.Stat(credPath); err == nil {
			c, err := sh.NewClient(credPath, spreadsheetID)
			if err != nil {
				log.Printf("⚠ Sheets client error: %v — running in dev mode", err)
			} else {
				client = c
				log.Printf("✓ Connected to Google Sheets (%s)", spreadsheetID)
				fmt.Println("  Verifying tabs…")
				if err := client.EnsureTabs(koshTabs); err != nil {
					log.Printf("⚠ EnsureTabs: %v", err)
				}
			}
		} else {
			log.Printf("⚠ %s not found — running in dev mode", credPath)
		}
	} else {
		log.Println("⚠ SPREADSHEET_ID not set — running in dev mode (dev_data.json)")
	}

	// ── Auth ───────────────────────────────────────────────────────────────────
	// Auth is enabled only when GOOGLE_CLIENT_ID + SESSION_SECRET are both set.
	// Without them the app runs open (useful for local dev).
	var auth *handlers.AuthHandler
	authEnabled := googleClientID != "" && sessionSecret != ""
	if authEnabled {
		emails := strings.Split(allowedRaw, ",")
		auth = handlers.NewAuthHandler(emails, googleClientID, sessionSecret, cookieSecure)
		log.Printf("✓ Auth enabled — %d allowed email(s)", len(emails))
	} else {
		log.Println("⚠ Auth disabled (GOOGLE_CLIENT_ID / SESSION_SECRET not set)")
	}

	// protect wraps a handler with auth if auth is enabled.
	protect := func(next http.HandlerFunc) http.HandlerFunc {
		if auth != nil {
			return auth.Require(next)
		}
		return next
	}

	h := handlers.NewHandler(client)
	upload := handlers.NewUploadHandler(anthropicKey, promptsDir)
	mux := http.NewServeMux()

	// ── Auth routes (always registered, no auth required) ─────────────────────
	mux.HandleFunc("/api/auth/config", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		if authEnabled {
			auth.Config(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
		}
	})
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		if auth != nil {
			auth.Login(w, r)
		} else {
			http.Error(w, "auth not configured", http.StatusNotFound)
		}
	})
	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		if auth != nil {
			auth.Me(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
		}
	})
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		if auth != nil {
			auth.Logout(w, r)
		} else {
			http.Error(w, "auth not configured", http.StatusNotFound)
		}
	})

	// ── Health check (no auth required — useful for diagnostics) ──────────────
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		status := map[string]interface{}{"mode": "dev", "auth": authEnabled}
		if client != nil {
			status["mode"] = "live"
			status["spreadsheet_id"] = spreadsheetID
			tabs := []string{"Members", "MF", "Stocks", "Metals", "Fixed", "Insurance", "Loans", "SIPs", "Lumpsums", "History", "Config"}
			tabStatus := map[string]string{}
			for _, tab := range tabs {
				if _, err := client.ReadSheet(tab); err != nil {
					tabStatus[tab] = err.Error()
				} else {
					tabStatus[tab] = "ok"
				}
			}
			status["tabs"] = tabStatus
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// ── Document upload (protected) ───────────────────────────────────────────
	mux.HandleFunc("/api/upload/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			return
		}
		protect(upload.Handle)(w, r)
	})

	// ── Protected API routes ───────────────────────────────────────────────────
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		protect(h.GetData)(w, r)
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions { return }
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		parts := strings.Split(path, "/")
		switch {
		case r.Method == http.MethodPost && len(parts) == 1:
			protect(h.AddRow)(w, r)
		case r.Method == http.MethodPut && len(parts) == 2:
			protect(h.UpdateRow)(w, r)
		case r.Method == http.MethodDelete && len(parts) == 2:
			protect(h.DeleteRow)(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// ── Frontend static files ──────────────────────────────────────────────────
	fs := http.FileServer(http.Dir(frontendDist))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := frontendDist + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.ServeFile(w, r, frontendDist+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	addr := ":" + port
	fmt.Printf("🪙  Kosh running at http://localhost%s\n", addr)
	if client == nil {
		fmt.Println("   Mode: dev (serving dev_data.json — mutations are no-ops)")
	} else {
		fmt.Println("   Mode: live (reading/writing Google Sheets)")
	}
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// content security policy — only whitelists what Kosh actually needs:
// Google Identity Services (sign-in script + OAuth popups), Google's token/Drive
// endpoints, and Google Fonts. Inline styles are allowed because the UI relies
// heavily on React inline `style={{...}}` attributes.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://accounts.google.com; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self' https://oauth2.googleapis.com https://www.googleapis.com https://accounts.google.com; " +
	"frame-src https://accounts.google.com; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders adds the standard set of browser security headers to every
// response — important since Kosh serves financial data over the open web.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		// Only advertise HSTS over HTTPS — Railway terminates TLS at its edge proxy,
		// so r.TLS is nil and we must check the forwarded-proto header instead.
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
