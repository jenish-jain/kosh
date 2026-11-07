package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"kosh/internal/app"
	sh "kosh/sheets"
)

// devOverrides lets the hosteddev build (hosteddev.go, behind a build tag)
// swap in fakes and add dev-only routes. Nil in every normal build.
var devOverrides func(cfg *app.Config, deps *app.Deps) func(http.Handler) http.Handler

func main() {
	_ = godotenv.Load() // load .env if present; ignore error if not

	cfg := app.ConfigFromEnv()
	deps := app.Deps{}
	wrap := func(h http.Handler) http.Handler { return h }
	if devOverrides != nil {
		wrap = devOverrides(&cfg, &deps)
	}

	a, err := app.New(context.Background(), cfg, deps)
	if err != nil {
		log.Fatalf("✗ %v", err)
	}

	addr := ":" + sh.EnvOrDefault("PORT", "8080")
	fmt.Printf("🪙  Kosh running at http://localhost%s\n", addr)
	fmt.Println("   " + a.Summary)
	log.Fatal(http.ListenAndServe(addr, wrap(a.Handler)))
}
