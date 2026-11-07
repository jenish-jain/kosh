package api

import (
	"strings"
	"testing"
)

func TestBuildCSP_DefaultMatchesOriginalPolicy(t *testing.T) {
	want := "default-src 'self'; " +
		"script-src 'self' https://accounts.google.com; " +
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
		"font-src 'self' https://fonts.gstatic.com; " +
		"img-src 'self' data:; " +
		"connect-src 'self' https://oauth2.googleapis.com https://www.googleapis.com https://accounts.google.com; " +
		"frame-src https://accounts.google.com; " +
		"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
	if got := buildCSP(nil); got != want {
		t.Fatalf("self-host CSP changed:\n got %s\nwant %s", got, want)
	}
}

func TestBuildCSP_AddsSourcesWithoutLeakingAcrossCalls(t *testing.T) {
	got := buildCSP(map[string][]string{"script-src": {"https://checkout.example"}})
	if !strings.Contains(got, "script-src 'self' https://accounts.google.com https://checkout.example;") {
		t.Fatalf("source not added: %s", got)
	}
	if strings.Contains(buildCSP(nil), "checkout.example") {
		t.Fatal("extra sources leaked into the base policy")
	}
}
