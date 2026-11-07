// Package billing handles hosted-mode subscriptions: plans, checkout via
// Razorpay, webhook-driven subscription state, and which features each
// tier of household gets.
package billing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"kosh/internal/controlplane"
)

// Tiers a household can be on. Which one is decided by its billing state
// (TierFor); what each tier may do is the operator's feature policy.
const (
	// TierFree: no running trial or paid plan — the trial ended, or the
	// subscription lapsed or was cancelled and its paid time is used up.
	TierFree  = "free"
	TierTrial = "trial"
	TierPaid  = "paid"
)

// Tiers in display order.
var Tiers = []TierInfo{
	{TierFree, "Free", "After the trial, or once a subscription ends"},
	{TierTrial, "Trial", "The first 14 days"},
	{TierPaid, "Paid", "Monthly or annual subscribers"},
}

type TierInfo struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Help  string `json:"help"`
}

// Features the operator can switch on or off per tier. Reading and
// exporting data is never gated — it's the customer's data, and it's in
// their own Google Sheet regardless — and neither is anything needed to
// leave or fix an account (billing, deleting the household, revoking
// share links).
const (
	FeatureEdit   = "edit"
	FeatureShare  = "share"
	FeatureInvite = "invite"
	FeatureAsk    = "ask"
	FeatureParse  = "parse"
	FeatureTax    = "tax"
)

// Features in display order.
var Features = []FeatureInfo{
	{FeatureEdit, "Adding and editing data", "Any change to the household's data from Kosh"},
	{FeatureShare, "Family sharing", "Creating or resetting share links. Existing links keep working and can always be revoked."},
	{FeatureInvite, "Household invites", "Inviting people to sign in to the household"},
	{FeatureAsk, "Ask Kosh", "AI questions about the household's finances"},
	{FeatureParse, "Document scanning", "AI reading of uploaded statements and documents"},
	{FeatureTax, "Tax recommendations", "AI tax-saving suggestions"},
}

type FeatureInfo struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Help  string `json:"help"`
}

func featureLabel(key string) string {
	for _, f := range Features {
		if f.Key == key {
			return f.Label
		}
	}
	return key
}

// DefaultPolicy is what each tier gets until the operator saves a policy,
// and for any feature a saved policy doesn't mention. Trial and paid get
// everything; free is read-only but can still invite people (so a lapsed
// household's family can see their data).
func DefaultPolicy() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, t := range Tiers {
		out[t.Key] = map[string]bool{}
		for _, f := range Features {
			out[t.Key][f.Key] = t.Key != TierFree || f.Key == FeatureInvite
		}
	}
	return out
}

// TierFor decides a household's tier from its billing state at now.
func TierFor(b controlplane.Billing, now time.Time) string {
	switch b.Status {
	case controlplane.StatusTrialing:
		if now.Before(b.TrialEndsAt) {
			return TierTrial
		}
	case controlplane.StatusActive, controlplane.StatusPastDue:
		// past_due: Razorpay is still retrying the charge — keep the
		// household working while that plays out.
		return TierPaid
	case controlplane.StatusCancelled:
		// Cancelled at period end: paid-for time is honoured.
		if now.Before(b.CurrentPeriodEnd) {
			return TierPaid
		}
	}
	return TierFree
}

// policyTTL is how long an instance trusts its copy of the policy, so an
// operator's change reaches every instance within this time.
const policyTTL = 30 * time.Second

// ErrInvalidPolicy: a policy naming a tier or feature that doesn't exist.
var ErrInvalidPolicy = errors.New("invalid feature policy")

// PolicyView is the effective policy: the saved one over the defaults.
type PolicyView struct {
	Tiers     map[string]map[string]bool `json:"policy"`
	UpdatedBy string                     `json:"updated_by,omitempty"`
	UpdatedAt time.Time                  `json:"updated_at,omitempty"`
}

// Policy serves the operator's feature policy, cached briefly.
type Policy struct {
	store controlplane.Store
	now   func() time.Time

	mu        sync.Mutex
	view      *PolicyView
	fetchedAt time.Time
}

func NewPolicy(store controlplane.Store) *Policy {
	return &Policy{store: store, now: time.Now}
}

// Current returns the effective policy. If the store can't be reached but
// a copy was loaded before, that copy is used rather than failing every
// request.
func (p *Policy) Current(ctx context.Context) (PolicyView, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.view != nil && p.now().Sub(p.fetchedAt) < policyTTL {
		return copyView(*p.view), nil
	}
	saved, err := p.store.GetFeaturePolicy(ctx)
	switch {
	case errors.Is(err, controlplane.ErrNotFound):
		saved = controlplane.FeaturePolicy{}
	case err != nil:
		if p.view != nil {
			log.Printf("⚠ billing: reloading feature policy (using the last copy): %v", err)
			return copyView(*p.view), nil
		}
		return PolicyView{}, err
	}
	v := PolicyView{Tiers: merge(saved.Tiers), UpdatedBy: saved.UpdatedBy, UpdatedAt: saved.UpdatedAt}
	p.view, p.fetchedAt = &v, p.now()
	return copyView(v), nil
}

// Features returns what tier may do.
func (p *Policy) Features(ctx context.Context, tier string) (map[string]bool, error) {
	v, err := p.Current(ctx)
	if err != nil {
		return nil, err
	}
	return v.Tiers[tier], nil
}

// Set saves a new policy. Every tier and feature named must exist; ones
// left out keep their defaults.
func (p *Policy) Set(ctx context.Context, tiers map[string]map[string]bool, by string) (PolicyView, error) {
	if err := validate(tiers); err != nil {
		return PolicyView{}, err
	}
	saved := controlplane.FeaturePolicy{Tiers: merge(tiers), UpdatedBy: by, UpdatedAt: p.now().UTC()}
	if err := p.store.PutFeaturePolicy(ctx, saved); err != nil {
		return PolicyView{}, err
	}
	v := PolicyView{Tiers: saved.Tiers, UpdatedBy: saved.UpdatedBy, UpdatedAt: saved.UpdatedAt}
	p.mu.Lock()
	p.view, p.fetchedAt = &v, p.now()
	p.mu.Unlock()
	return copyView(v), nil
}

func validate(tiers map[string]map[string]bool) error {
	known := func(key string, list []string) bool {
		for _, k := range list {
			if k == key {
				return true
			}
		}
		return false
	}
	var tierKeys, featureKeys []string
	for _, t := range Tiers {
		tierKeys = append(tierKeys, t.Key)
	}
	for _, f := range Features {
		featureKeys = append(featureKeys, f.Key)
	}
	for tier, fs := range tiers {
		if !known(tier, tierKeys) {
			return fmt.Errorf("%w: unknown tier %q", ErrInvalidPolicy, tier)
		}
		for f := range fs {
			if !known(f, featureKeys) {
				return fmt.Errorf("%w: unknown feature %q", ErrInvalidPolicy, f)
			}
		}
	}
	return nil
}

// merge lays saved choices over the defaults, ignoring tiers or features
// that no longer exist.
func merge(saved map[string]map[string]bool) map[string]map[string]bool {
	out := DefaultPolicy()
	for tier, fs := range saved {
		if _, ok := out[tier]; !ok {
			continue
		}
		for f, on := range fs {
			if _, ok := out[tier][f]; ok {
				out[tier][f] = on
			}
		}
	}
	return out
}

func copyView(v PolicyView) PolicyView {
	v.Tiers = merge(v.Tiers)
	return v
}

// featureFor maps a request to the feature it uses, or "" for requests
// that are never gated: reads, and the account, billing, recipient and
// safety actions a household must always keep.
func featureFor(r *http.Request) string {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/api/") {
		return ""
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return ""
	}
	switch {
	case p == "/api/hosted/invites":
		return FeatureInvite
	case p == "/api/shares", strings.HasPrefix(p, "/api/shares/") && strings.HasSuffix(p, "/reset"):
		return FeatureShare
	case strings.HasPrefix(p, "/api/shares/"):
		return "" // revoke and delete
	case p == "/api/chat":
		return FeatureAsk
	case p == "/api/tax/recommendations/generate":
		return FeatureTax
	case strings.HasPrefix(p, "/api/upload/"):
		return FeatureParse
	}
	for _, prefix := range []string{"/api/auth/", "/api/hosted/", "/api/billing/", "/api/share/", "/api/operator/", "/api/dev/"} {
		if strings.HasPrefix(p, prefix) {
			return ""
		}
	}
	return FeatureEdit
}

// unavailable explains a 402 to the customer.
func unavailable(tier, feature string) string {
	if tier == TierFree && feature == FeatureEdit {
		return "Your household is read-only. Subscribe to keep editing."
	}
	switch tier {
	case TierFree:
		return featureLabel(feature) + " isn't included in the free plan. Subscribe to use it."
	case TierTrial:
		return featureLabel(feature) + " isn't included in the free trial. Subscribe to use it."
	}
	return featureLabel(feature) + " isn't available on your plan."
}
