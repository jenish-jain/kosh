package app_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"kosh/internal/ai"
	"kosh/internal/app"
	"kosh/internal/controlplane"
	"kosh/internal/hosted"
	"kosh/internal/hosted/hostedtest"
)

// stubAI answers every chat, failing while fail is set.
type stubAI struct {
	calls atomic.Int32
	fail  atomic.Bool
}

func (s *stubAI) Chat(context.Context, []ai.Message) (string, error) {
	s.calls.Add(1)
	if s.fail.Load() {
		return "", errors.New("provider down")
	}
	return "ok", nil
}

func newHostedEnvWithAI(t *testing.T, provider ai.Provider) *env {
	t.Helper()
	e := &env{t: t, google: hostedtest.New(), store: controlplane.NewMemory(), rzp: hostedtest.NewRazorpay()}
	a, err := app.New(context.Background(), hostedConfig(), app.Deps{Verifier: idVerifier{}, Store: e.store, Google: e.google, Gateway: e.rzp, AIProvider: provider})
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewTLSServer(a.Handler)
	t.Cleanup(e.srv.Close)
	return e
}

func (b *browser) ask() int {
	b.e.t.Helper()
	code, _ := b.do("POST", "/api/chat", map[string]any{"messages": []map[string]string{{"role": "user", "content": "hi"}}})
	return code
}

func TestQuota_AskIsMeteredPerHouseholdAndRefundedOnFailure(t *testing.T) {
	provider := &stubAI{}
	e := newHostedEnvWithAI(t, provider)
	alice, hid := e.onboarded("alice@x.com")
	bob, _ := e.onboarded("bob@x.com")
	limit := hosted.MonthlyQuotas["ask"]
	period := time.Now().UTC().Format("2006-01")

	// Fill Alice's allowance to one below the limit directly, then use the
	// rest through the API.
	for i := 0; i < limit-1; i++ {
		e.store.ConsumeQuota(context.Background(), hid, "ask", period, limit)
	}
	// A failing AI call must not use up the last question.
	provider.fail.Store(true)
	if code := alice.ask(); code < 500 {
		t.Fatalf("failing provider: want 5xx, got %d", code)
	}
	provider.fail.Store(false)
	if code := alice.ask(); code != 200 {
		t.Fatalf("last question after a refunded failure: want 200, got %d", code)
	}
	before := provider.calls.Load()
	code, out := alice.do("POST", "/api/chat", map[string]any{"messages": []map[string]string{{"role": "user", "content": "hi"}}})
	if code != 429 || !strings.Contains(out, "questions to Ask Kosh") {
		t.Fatalf("over quota: want 429 with explanation, got %d %s", code, out)
	}
	if provider.calls.Load() != before {
		t.Error("the AI provider was called despite the quota being exhausted")
	}
	// Quotas are per household.
	if code := bob.ask(); code != 200 {
		t.Errorf("another household's quota: want 200, got %d", code)
	}

	var usage struct {
		Used   map[string]int `json:"used"`
		Limits map[string]int `json:"limits"`
	}
	alice.json("GET", "/api/hosted/usage", nil, &usage)
	if usage.Used["ask"] != limit || usage.Limits["ask"] != limit {
		t.Errorf("usage: %+v", usage)
	}
}

func TestDeleteHousehold(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	_, subID := alice.checkout("monthly")
	e.rzp.SetStatus(subID, "active", time.Now().AddDate(0, 1, 0))
	e.webhook("subscription.activated", subID, false)
	var inv struct {
		LinkPath string `json:"link_path"`
	}
	alice.json("POST", "/api/hosted/invites", map[string]string{"email": "bob@x.com"}, &inv)
	bob := e.browser()
	bob.login("bob@x.com")
	bob.do("POST", "/api/hosted/invites/accept", map[string]string{"token": strings.TrimPrefix(inv.LinkPath, "/join#")})
	var share struct {
		LinkPath string `json:"link_path"`
	}
	alice.json("POST", "/api/shares", map[string]string{"name": "Mom"}, &share)
	before, _ := e.store.GetHousehold(context.Background(), hid)
	sheetID := before.SpreadsheetID
	sealer, _ := controlplane.NewTokenSealer(hostedConfig().ControlPlaneKey)
	refreshToken, err := sealer.Open(before.RefreshTokenEnc, hid)
	if err != nil {
		t.Fatal(err)
	}

	if code, _ := bob.do("DELETE", "/api/hosted/household", map[string]string{"confirm": "Household of alice@x.com"}); code != 403 {
		t.Errorf("member deleting: want 403, got %d", code)
	}
	if code, _ := alice.do("DELETE", "/api/hosted/household", map[string]string{"confirm": "wrong name"}); code != 400 {
		t.Errorf("wrong confirmation: want 400, got %d", code)
	}
	if code, out := alice.do("DELETE", "/api/hosted/household", map[string]string{"confirm": "Household of alice@x.com"}); code != 200 {
		t.Fatalf("delete: %d %s", code, out)
	}

	if sub, _ := e.rzp.FetchSubscription(context.Background(), subID); sub.Status != "cancelled" {
		t.Errorf("subscription should be cancelled immediately, is %s", sub.Status)
	}
	if _, err := e.store.GetHousehold(context.Background(), hid); !errors.Is(err, controlplane.ErrNotFound) {
		t.Error("household record survived")
	}
	for _, b := range []*browser{alice, bob} {
		if m := b.me(); m.Household != nil {
			t.Errorf("%s still has a household", m.Email)
		}
	}
	mom := e.browser()
	mom.header.Set("X-Kosh-Share-Token", strings.TrimPrefix(share.LinkPath, "/s#"))
	if code, _ := mom.do("GET", "/api/share/report", nil); code != 401 {
		t.Errorf("share link after deletion: want 401, got %d", code)
	}
	// Kosh's Google access is revoked at Google, not just forgotten.
	if _, err := e.google.TokenSource(refreshToken).Token(); err == nil {
		t.Error("the household's Google grant still works after deletion")
	}
	// The customer's spreadsheet is theirs and is left untouched.
	if e.google.Sheet(sheetID) == nil {
		t.Error("the customer's spreadsheet was deleted")
	}
	// They can start over.
	alice.onboard("alice@x.com", "Fresh start")
}

func TestDeleteHousehold_BillingFailureAborts(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	// Point the household at a subscription the gateway can't find: the
	// cancel fails, so nothing may be deleted (they'd keep being charged).
	e.store.UpdateHousehold(context.Background(), hid, func(h *controlplane.Household) error {
		h.Billing.SubscriptionID = "sub_missing"
		return nil
	})
	if code, _ := alice.do("DELETE", "/api/hosted/household", map[string]string{"confirm": "Household of alice@x.com"}); code != 500 {
		t.Fatalf("want 500 when billing can't be cancelled, got %d", code)
	}
	if _, err := e.store.GetHousehold(context.Background(), hid); err != nil {
		t.Fatal("household deleted even though billing wasn't stopped")
	}
}
