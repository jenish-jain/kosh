package billing

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"kosh/internal/controlplane"
)

// The defaults must reproduce hosted Kosh before tiers were configurable:
// trial and paid get everything; free is read-only but can still invite.
func TestDefaultPolicy_MatchesTheOriginalReadOnlyRule(t *testing.T) {
	d := DefaultPolicy()
	for _, f := range Features {
		if !d[TierTrial][f.Key] || !d[TierPaid][f.Key] {
			t.Errorf("%s should be on for trial and paid by default", f.Key)
		}
		if want := f.Key == FeatureInvite; d[TierFree][f.Key] != want {
			t.Errorf("free %s: got %v, want %v", f.Key, d[TierFree][f.Key], want)
		}
	}
}

func TestFeatureFor(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{"GET", "/api/data", ""},
		{"GET", "/api/shares", ""},
		{"POST", "/api/Members", FeatureEdit},
		{"PUT", "/api/Investments/abc", FeatureEdit},
		{"DELETE", "/api/Investments/abc", FeatureEdit},
		{"POST", "/api/tax/rules/propose", FeatureEdit},
		{"POST", "/api/shares", FeatureShare},
		{"POST", "/api/shares/s1/reset", FeatureShare},
		{"POST", "/api/shares/s1/revoke", ""},
		{"DELETE", "/api/shares/s1", ""},
		{"POST", "/api/chat", FeatureAsk},
		{"POST", "/api/tax/recommendations/generate", FeatureTax},
		{"POST", "/api/upload/statement", FeatureParse},
		{"POST", "/api/hosted/invites", FeatureInvite},
		// Trailing slashes reach the same handlers, so they must gate too.
		{"POST", "/api/shares/", FeatureShare},
		{"POST", "/api/shares/s1/reset/", FeatureShare},
		{"POST", "/api/hosted/invites/", FeatureInvite},
		{"POST", "/api/Members/", FeatureEdit},
		{"POST", "/api/shares/s1/revoke/", ""},
		{"DELETE", "/api/shares/s1/", ""},
		{"DELETE", "/api/shares", FeatureShare},
		{"PUT", "/api/shares/s1", FeatureShare},
		{"POST", "/api/hosted/invites/accept", ""},
		{"DELETE", "/api/hosted/household", ""},
		{"POST", "/api/billing/checkout", ""},
		{"PUT", "/api/operator/features", ""},
		{"POST", "/api/share/verify", ""},
		{"POST", "/api/auth/logout", ""},
		{"POST", "/app", ""},
	}
	for _, c := range cases {
		if got := featureFor(httptest.NewRequest(c.method, c.path, nil)); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

// flakyStore fails policy reads on demand.
type flakyStore struct {
	controlplane.Store
	fail bool
}

func (f *flakyStore) GetFeaturePolicy(ctx context.Context) (controlplane.FeaturePolicy, error) {
	if f.fail {
		return controlplane.FeaturePolicy{}, errors.New("firestore down")
	}
	return f.Store.GetFeaturePolicy(ctx)
}

func TestPolicy(t *testing.T) {
	ctx := context.Background()
	store := &flakyStore{Store: controlplane.NewMemory()}
	p := NewPolicy(store)
	clock := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	p.now = func() time.Time { return clock }

	if v, err := p.Current(ctx); err != nil || v.Tiers[TierFree][FeatureEdit] || !v.Tiers[TierPaid][FeatureShare] {
		t.Fatalf("nothing saved: want the defaults, got %+v %v", v, err)
	}

	if _, err := p.Set(ctx, map[string]map[string]bool{"gold": {FeatureShare: true}}, "op@x.com"); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("unknown tier: want ErrInvalidPolicy, got %v", err)
	}
	if _, err := p.Set(ctx, map[string]map[string]bool{TierFree: {"teleport": true}}, "op@x.com"); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("unknown feature: want ErrInvalidPolicy, got %v", err)
	}

	// A partial policy keeps the defaults for what it leaves out.
	v, err := p.Set(ctx, map[string]map[string]bool{TierFree: {FeatureEdit: true}, TierTrial: {FeatureShare: false}}, "op@x.com")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Tiers[TierFree][FeatureEdit] || v.Tiers[TierFree][FeatureShare] || v.Tiers[TierTrial][FeatureShare] || !v.Tiers[TierTrial][FeatureAsk] || v.UpdatedBy != "op@x.com" {
		t.Errorf("after set: %+v", v)
	}
	// Callers can't change the cached policy through what they're given.
	v.Tiers[TierTrial][FeatureShare] = true
	if f, _ := p.Features(ctx, TierTrial); f[FeatureShare] {
		t.Error("mutating a returned policy changed the cache")
	}

	// Another instance saves a change: picked up once the cache expires.
	store.Store.PutFeaturePolicy(ctx, controlplane.FeaturePolicy{Tiers: map[string]map[string]bool{TierPaid: {FeatureAsk: false}}})
	if f, _ := p.Features(ctx, TierPaid); !f[FeatureAsk] {
		t.Error("cache should still be fresh")
	}
	clock = clock.Add(policyTTL)
	if f, _ := p.Features(ctx, TierPaid); f[FeatureAsk] {
		t.Error("expired cache: the other instance's change wasn't picked up")
	}

	// The store going down keeps the last known policy rather than failing.
	store.fail = true
	clock = clock.Add(policyTTL)
	if f, err := p.Features(ctx, TierPaid); err != nil || f[FeatureAsk] {
		t.Errorf("store down: want the last copy, got %v %v", f, err)
	}
	// With no copy at all, it's an error (the caller answers 500).
	if _, err := NewPolicy(store).Current(ctx); err == nil {
		t.Error("store down and nothing cached: want an error")
	}
}

func TestUnavailableMessages(t *testing.T) {
	if got := unavailable(TierFree, FeatureEdit); got != "Your household is read-only. Subscribe to keep editing." {
		t.Errorf("free edit: %q", got)
	}
	if got := unavailable(TierFree, FeatureShare); got != "Family sharing isn't included in the free plan. Subscribe to use it." {
		t.Errorf("free share: %q", got)
	}
	if got := unavailable(TierPaid, FeatureAsk); got != "Ask Kosh isn't available on your plan." {
		t.Errorf("paid ask: %q", got)
	}
}
