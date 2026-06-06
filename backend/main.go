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

	port := sh.EnvOrDefault("PORT", "8080")
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	credPath := sh.EnvOrDefault("CREDENTIALS_PATH", "credentials.json")
	frontendDist := sh.EnvOrDefault("FRONTEND_DIST", "../frontend/dist")

	// Tab definitions — order controls creation order in the spreadsheet.
	koshTabs := []sh.TabDef{
		{Name: "Members",   Columns: []string{"id", "name", "full_name", "relation", "slab", "color"}},
		{Name: "MF",        Columns: []string{"id", "name", "plan", "platform", "member", "invested", "current", "sip", "notes"}},
		{Name: "Stocks",    Columns: []string{"id", "name", "ticker", "qty", "avg_price", "last_price", "member"}},
		{Name: "Metals",    Columns: []string{"id", "type", "date_purchased", "grams", "buy_rate", "today_price", "place", "member"}},
		{Name: "Fixed",     Columns: []string{"id", "kind", "name", "member", "principal", "rate", "current_value", "opened", "matures", "monthly"}},
		{Name: "Insurance", Columns: []string{"id", "name", "type", "member", "premium", "freq", "paid", "value", "cover", "maturity"}},
		{Name: "SIPs",      Columns: []string{"id", "fund", "member", "amount", "day", "status", "start_date", "platform"}},
		{Name: "Lumpsums",  Columns: []string{"id", "fund", "member", "amount", "date"}},
		{Name: "History",   Columns: []string{"month", "value"}},
		{Name: "Config",    Columns: []string{"key", "value"}},
	}

	// Try to create a real Sheets client; fall back to dev mode if credentials missing.
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

	h := handlers.NewHandler(client)
	mux := http.NewServeMux()

	// Health check — lists which sheet tabs are readable
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			return
		}
		status := map[string]interface{}{
			"mode": "dev",
		}
		if client != nil {
			status["mode"] = "live"
			status["spreadsheet_id"] = spreadsheetID
			tabs := []string{"Members", "MF", "Stocks", "Metals", "Fixed", "Insurance", "SIPs", "Lumpsums", "History", "Config"}
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

	// API routes
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			return
		}
		h.GetData(w, r)
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			return
		}
		// Route by method + path depth
		// POST /api/{sheet}        → add row
		// PUT  /api/{sheet}/{id}   → update row
		// DELETE /api/{sheet}/{id} → delete row
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		parts := strings.Split(path, "/")
		switch {
		case r.Method == http.MethodPost && len(parts) == 1:
			h.AddRow(w, r)
		case r.Method == http.MethodPut && len(parts) == 2:
			h.UpdateRow(w, r)
		case r.Method == http.MethodDelete && len(parts) == 2:
			h.DeleteRow(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// Serve frontend static files
	fs := http.FileServer(http.Dir(frontendDist))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback: if file not found, serve index.html
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
	log.Fatal(http.ListenAndServe(addr, mux))
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
