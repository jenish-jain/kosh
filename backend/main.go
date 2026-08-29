package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"kosh/drive"
	"kosh/internal/ai"
	"kosh/internal/api"
	"kosh/internal/auth"
	"kosh/internal/documents"
	"kosh/internal/store"
	sh "kosh/sheets"
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

	// ── Auth ───────────────────────────────────────────────────────────────────
	// Auth is always constructed — NewServer will use authH for /api/auth/* routes.
	// The handler is aware of its own configuration (clientID, allowedEmails).
	allowedEmails := strings.Split(allowedRaw, ",")
	verifier := auth.NewGoogleTokenVerifier(http.DefaultClient)
	authH := auth.NewHandler(sessionSecret, googleClientID, allowedEmails, cookieSecure, verifier)
	if googleClientID != "" && sessionSecret != "" {
		log.Printf("✓ Auth enabled — %d allowed email(s)", len(allowedEmails))
	} else {
		log.Println("⚠ Auth disabled (GOOGLE_CLIENT_ID / SESSION_SECRET not set)")
	}

	// ── Sheets client ──────────────────────────────────────────────────────────
	var sheetClient *sh.Client
	var repos *store.Repositories
	if spreadsheetID != "" {
		if _, err := os.Stat(credPath); err == nil {
			c, err := sh.NewClient(credPath, spreadsheetID)
			if err != nil {
				log.Printf("⚠ Sheets client error: %v — running in dev mode", err)
			} else {
				sheetClient = c
				repos = store.NewRepositories(c)
				log.Printf("✓ Connected to Google Sheets (%s)", spreadsheetID)
				fmt.Println("  Verifying tabs…")
				// Build tab definitions from repositories (single source of truth).
				tabs := []sh.TabDef{
					{Name: repos.Members.Sheet(), Columns: repos.Members.Columns()},
					{Name: repos.MF.Sheet(), Columns: repos.MF.Columns()},
					{Name: repos.Stocks.Sheet(), Columns: repos.Stocks.Columns()},
					{Name: repos.Metals.Sheet(), Columns: repos.Metals.Columns()},
					{Name: repos.Fixed.Sheet(), Columns: repos.Fixed.Columns()},
					{Name: repos.Insurance.Sheet(), Columns: repos.Insurance.Columns()},
					{Name: repos.Loans.Sheet(), Columns: repos.Loans.Columns()},
					{Name: repos.NPS.Sheet(), Columns: repos.NPS.Columns()},
					{Name: repos.SIPs.Sheet(), Columns: repos.SIPs.Columns()},
					{Name: repos.Lumpsums.Sheet(), Columns: repos.Lumpsums.Columns()},
					{Name: repos.History.Sheet(), Columns: repos.History.Columns()},
					{Name: repos.Income.Sheet(), Columns: repos.Income.Columns()},
					{Name: repos.TaxRecommendations.Sheet(), Columns: repos.TaxRecommendations.Columns()},
					{Name: "Config", Columns: []string{"key", "value"}},
				}
				if err := sheetClient.EnsureTabs(tabs); err != nil {
					log.Printf("⚠ EnsureTabs: %v", err)
				}
			}
		} else {
			log.Printf("⚠ %s not found — running in dev mode", credPath)
		}
	} else {
		log.Println("⚠ SPREADSHEET_ID not set — running in dev mode (dev_data.json)")
	}

	// In dev mode (sheetClient == nil), repos is nil — NewHandler handles this
	// by loading dev data. Use a fake API so repos is always non-nil.
	if repos == nil {
		repos = store.NewRepositories(store.NewFakeSheetsAPI())
	}

	// ── API handler ────────────────────────────────────────────────────────────
	isDemo := func(r *http.Request) bool {
		return authH.IsDemoSession(r)
	}
	// Pass sheetClient as configAPI (nil in dev mode is fine — Handler handles it).
	apiH := api.NewHandler(repos, sheetClient, isDemo)

	// ── Document upload handler ────────────────────────────────────────────────
	// Drive uploads use a per-request OAuth token from the frontend (drive.file
	// scope). The factory creates a new Drive client from that token each call.
	var driveFactory documents.DriveClientFactory
	driveFactory = func(ctx context.Context, accessToken string) (drive.API, error) {
		return drive.NewClientFromToken(ctx, accessToken)
	}
	var parser documents.Parser
	if anthropicKey != "" {
		parser = documents.NewClaudeParser(anthropicKey, promptsDir, http.DefaultClient)
	}
	docH := documents.NewUploadHandler(driveFactory, parser)

	// ── AI Financial Advisor ───────────────────────────────────────────────────
	var aiH *ai.Handler
	var taxH *ai.TaxHandler
	if sh.EnvOrDefault("AI_ENABLED", "false") == "true" {
		aiProvider := sh.EnvOrDefault("AI_PROVIDER", "ollama")
		var provider ai.Provider
		switch aiProvider {
		case "anthropic":
			key := os.Getenv("ANTHROPIC_API_KEY")
			if key == "" {
				log.Println("⚠ AI_PROVIDER=anthropic but ANTHROPIC_API_KEY not set — AI disabled")
			} else {
				provider = ai.NewAnthropicProvider(key, "", nil)
				log.Println("✓ AI advisor enabled — provider: Anthropic")
			}
		default: // ollama
			ollamaURL := sh.EnvOrDefault("OLLAMA_URL", "http://localhost:11434")
			ollamaModel := sh.EnvOrDefault("OLLAMA_MODEL", "llama3.1")
			provider = ai.NewOllamaProvider(ollamaURL, ollamaModel, nil)
			log.Printf("✓ AI advisor enabled — provider: Ollama (%s @ %s)", ollamaModel, ollamaURL)
		}
		if provider != nil {
			aiH = ai.NewHandler(provider, apiH.LoadData, isDemo)
			taxH = ai.NewTaxHandler(provider, repos, apiH.LoadData, isDemo, apiH.IsSampleData)
			authH.SetAIEnabled(true)
		}
	}

	// ── HTTP server ────────────────────────────────────────────────────────────
	handler := api.NewServer(authH, apiH, docH, aiH, taxH, frontendDist)

	addr := ":" + port
	fmt.Printf("🪙  Kosh running at http://localhost%s\n", addr)
	if sheetClient == nil {
		fmt.Println("   Mode: dev (serving dev_data.json — mutations are no-ops)")
	} else {
		fmt.Println("   Mode: live (reading/writing Google Sheets)")
	}
	log.Fatal(http.ListenAndServe(addr, handler))
}
