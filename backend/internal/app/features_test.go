package app_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"kosh/internal/billing"
)

type featurePolicy struct {
	Tiers []struct {
		Key string `json:"key"`
	} `json:"tiers"`
	Features []struct {
		Key string `json:"key"`
	} `json:"features"`
	Policy    map[string]map[string]bool `json:"policy"`
	Defaults  map[string]map[string]bool `json:"defaults"`
	UpdatedBy string                     `json:"updated_by"`
}

// operator signs in the account listed in OPERATOR_EMAILS (hostedConfig).
func (e *env) operator() *browser {
	e.t.Helper()
	op := e.browser()
	op.login("op@x.com")
	return op
}

func (b *browser) setPolicy(p map[string]map[string]bool) {
	b.e.t.Helper()
	if code, out := b.do("PUT", "/api/operator/features", map[string]any{"policy": p}); code != 200 {
		b.e.t.Fatalf("saving policy: %d %s", code, out)
	}
}

func TestFeatures_OnlyOperatorsSeeOrChangeThePolicy(t *testing.T) {
	e := newHostedEnv(t)
	alice, _ := e.onboarded("alice@x.com")
	anon := e.browser()

	for _, b := range []*browser{alice, anon} {
		if code, _ := b.do("GET", "/api/operator/features", nil); code != 404 && code != 401 {
			t.Errorf("non-operator read: want 404/401, got %d", code)
		}
		if code, _ := b.do("PUT", "/api/operator/features", map[string]any{"policy": map[string]map[string]bool{"free": {"edit": true}}}); code != 404 && code != 401 {
			t.Errorf("non-operator write: want 404/401, got %d", code)
		}
	}
	if alice.me().Operator {
		t.Error("alice is flagged as an operator")
	}

	// The operator needs no household of their own, and the email match is
	// case- and space-insensitive.
	op := e.operator()
	if !op.me().Operator {
		t.Error("operator not flagged in /me")
	}
	var got featurePolicy
	if code := op.json("GET", "/api/operator/features", nil, &got); code != 200 {
		t.Fatalf("operator read: %d", code)
	}
	if len(got.Tiers) != 3 || len(got.Features) != len(billing.Features) || got.Policy["free"]["edit"] || !got.Policy["paid"]["share"] || got.UpdatedBy != "" {
		t.Errorf("defaults: %+v", got)
	}

	for _, bad := range []string{
		`{"policy": {"gold": {"share": true}}}`,
		`{"policy": {"free": {"teleport": true}}}`,
		`{"policy": {"free": {"share": "yes"}}}`,
		`{"policy": {"free": {"share": true}}, "extra": 1}`,
		`{}`,
	} {
		if code, _ := op.do("PUT", "/api/operator/features", json.RawMessage(bad)); code != 400 {
			t.Errorf("%s: want 400, got %d", bad, code)
		}
	}
}

func TestFeatures_TogglesApplyPerTier(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com") // will lapse to free
	bob, _ := e.onboarded("bob@x.com")       // stays in trial
	op := e.operator()

	if m := bob.me(); m.Household.Tier != "trial" || !m.Household.Features["share"] {
		t.Errorf("trial defaults: %+v", m.Household)
	}

	// Free: allow editing but not sharing. Trial: no sharing either.
	op.setPolicy(map[string]map[string]bool{
		"free":  {"edit": true, "share": false},
		"trial": {"share": false},
	})
	e.endTrial(hid)

	m := alice.me()
	if m.Household.Tier != "free" || !m.Household.Features["edit"] || m.Household.Features["share"] || m.Household.Features["ask"] {
		t.Errorf("free features in /me: %+v", m.Household)
	}
	if code, out := alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"}); code != 200 {
		t.Errorf("free edit switched on: want 200, got %d %s", code, out)
	}
	if code, out := alice.do("POST", "/api/shares", map[string]string{"name": "Mom"}); code != 402 || !strings.Contains(out, "Family sharing isn't included in the free plan") {
		t.Errorf("free sharing switched off: want 402 with explanation, got %d %s", code, out)
	}
	if code, _ := alice.do("POST", "/api/shares/", map[string]string{"name": "Mom"}); code != 402 {
		t.Errorf("free sharing via a trailing slash: want 402, got %d", code)
	}
	if code, out := bob.do("POST", "/api/shares", map[string]string{"name": "Mom"}); code != 402 || !strings.Contains(out, "free trial") {
		t.Errorf("trial sharing switched off: want 402, got %d %s", code, out)
	}
	if code, _ := bob.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Ravi"}); code != 200 {
		t.Errorf("trial edit (left at default): want 200, got %d", code)
	}
	// Paid wasn't mentioned, so it keeps everything.
	_, subID := alice.checkout("monthly")
	e.rzp.SetStatus(subID, "active", time.Now().AddDate(0, 1, 0))
	e.webhook("subscription.activated", subID, false)
	if code, out := alice.do("POST", "/api/shares", map[string]string{"name": "Mom"}); code != 200 {
		t.Errorf("paid sharing: want 200, got %d %s", code, out)
	}

	// Switching it back on takes effect immediately on this instance.
	op.setPolicy(map[string]map[string]bool{"trial": {"share": true}})
	if code, _ := bob.do("POST", "/api/shares", map[string]string{"name": "Mom"}); code != 200 {
		t.Errorf("trial sharing switched back on: want 200, got %d", code)
	}
	var got featurePolicy
	op.json("GET", "/api/operator/features", nil, &got)
	if got.UpdatedBy != "op@x.com" || got.Policy["free"]["edit"] {
		t.Errorf("a save replaces the whole policy (free edit back to default): %+v", got)
	}
}

func TestFeatures_SwitchedOffInvitesAndAI(t *testing.T) {
	provider := &stubAI{}
	e := newHostedEnvWithAI(t, provider)
	alice, _ := e.onboarded("alice@x.com")
	e.operator().setPolicy(map[string]map[string]bool{"trial": {"ask": false, "invite": false}})

	if code := alice.ask(); code != 402 {
		t.Errorf("ask switched off: want 402, got %d", code)
	}
	if provider.calls.Load() != 0 {
		t.Error("the AI provider was called for a switched-off feature")
	}
	var usage struct {
		Used map[string]int `json:"used"`
	}
	alice.json("GET", "/api/hosted/usage", nil, &usage)
	if usage.Used["ask"] != 0 {
		t.Errorf("a refused question used up allowance: %v", usage.Used)
	}
	if code, _ := alice.do("POST", "/api/hosted/invites", map[string]string{"email": "bob@x.com"}); code != 402 {
		t.Errorf("invites switched off: want 402, got %d", code)
	}
	// Account safety actions are never gated.
	if code, _ := alice.do("DELETE", "/api/hosted/household", map[string]string{"confirm": "Household of alice@x.com"}); code != 200 {
		t.Errorf("delete household: want 200, got %d", code)
	}
}
