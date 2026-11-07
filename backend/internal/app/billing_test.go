package app_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"kosh/internal/controlplane"
	"kosh/internal/hosted/hostedtest"
)

// onboarded returns a signed-in owner with a household, plus its ID.
func (e *env) onboarded(email string) (*browser, string) {
	e.t.Helper()
	b := e.browser()
	b.login(email)
	b.onboard(email, "Household of "+email)
	return b, b.me().Household.ID
}

func (e *env) endTrial(householdID string) {
	e.t.Helper()
	e.store.UpdateHousehold(context.Background(), householdID, func(h *controlplane.Household) error {
		h.Billing.TrialEndsAt = time.Now().Add(-time.Minute)
		return nil
	})
}

func (e *env) billing(id string) controlplane.Billing {
	e.t.Helper()
	h, err := e.store.GetHousehold(context.Background(), id)
	if err != nil {
		e.t.Fatal(err)
	}
	return h.Billing
}

// webhook posts a signed (or deliberately mis-signed) Razorpay webhook.
func (e *env) webhook(event, subID string, tamper bool) int {
	e.t.Helper()
	body, sig := hostedtest.WebhookBody(event, subID)
	if tamper {
		sig = strings.Repeat("0", len(sig))
	}
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/billing/webhook", bytes.NewReader(body))
	req.Header.Set("X-Razorpay-Signature", sig)
	req.Header.Set("X-Razorpay-Event-Id", "evt_"+event+"_"+subID)
	resp, err := e.srv.Client().Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func (b *browser) checkout(plan string) (int, string) {
	b.e.t.Helper()
	var out struct {
		SubscriptionID string `json:"subscription_id"`
		KeyID          string `json:"key_id"`
	}
	code := b.json("POST", "/api/billing/checkout", map[string]string{"plan": plan}, &out)
	return code, out.SubscriptionID
}

func TestBilling_PlansComeFromTheGateway(t *testing.T) {
	e := newHostedEnv(t)
	alice, _ := e.onboarded("alice@x.com")
	var out struct {
		KeyID string `json:"key_id"`
		Plans []struct {
			ID     string `json:"id"`
			Amount int64  `json:"amount"`
		} `json:"plans"`
	}
	alice.json("GET", "/api/billing/plans", nil, &out)
	if out.KeyID != hostedtest.RazorpayKeyID || len(out.Plans) != 2 || out.Plans[0].Amount != 14900 || out.Plans[1].ID != "annual" {
		t.Fatalf("plans: %+v", out)
	}
}

func TestBilling_LapsedHouseholdIsReadOnlyButKeepsSafetyActions(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"})
	var enr struct {
		Share struct {
			ID string `json:"id"`
		} `json:"share"`
	}
	alice.json("POST", "/api/shares", map[string]string{"name": "Mom"}, &enr)

	e.endTrial(hid)

	if code, out := alice.do("POST", "/api/Members", map[string]string{"id": "m2", "name": "Ravi"}); code != 402 || !strings.Contains(out, "subscription_required") {
		t.Errorf("edit after trial: want 402 subscription_required, got %d %s", code, out)
	}
	if code, _ := alice.do("POST", "/api/shares", map[string]string{"name": "Dad"}); code != 402 {
		t.Errorf("new share link after trial: want 402, got %d", code)
	}
	if code, _ := alice.do("POST", "/api/chat", map[string]string{"message": "hi"}); code != 402 {
		t.Errorf("AI after trial: want 402, got %d", code)
	}
	// Still readable, exportable, and share links can still be shut off.
	if got := alice.memberNames(); len(got) != 1 {
		t.Errorf("read after trial: %v", got)
	}
	if code, _ := alice.do("GET", "/api/report/export?format=pdf&sections=summary", nil); code != 200 {
		t.Errorf("export after trial: want 200, got %d", code)
	}
	if code, _ := alice.do("POST", "/api/shares/"+enr.Share.ID+"/revoke", nil); code != 200 {
		t.Errorf("revoke after trial: want 200, got %d", code)
	}
	if code, _ := alice.do("DELETE", "/api/shares/"+enr.Share.ID, nil); code != 200 {
		t.Errorf("delete share after trial: want 200, got %d", code)
	}
}

func TestBilling_SubscribeActivateAndCancel(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	e.endTrial(hid)

	code, subID := alice.checkout("monthly")
	if code != 200 || subID == "" {
		t.Fatalf("checkout: %d", code)
	}
	// Abandon and retry: the same unfinished subscription is reused.
	if _, again := alice.checkout("monthly"); again != subID {
		t.Errorf("retrying checkout created a second subscription: %s vs %s", again, subID)
	}

	// Checkout's success callback: a forged signature is refused...
	if code, _ := alice.do("POST", "/api/billing/verify", map[string]string{
		"razorpay_payment_id": "pay_1", "razorpay_subscription_id": subID, "razorpay_signature": "deadbeef",
	}); code != 400 {
		t.Errorf("forged signature: want 400, got %d", code)
	}
	// ...a genuine one activates.
	e.rzp.SetStatus(subID, "active", time.Now().AddDate(0, 1, 0))
	if code, out := alice.do("POST", "/api/billing/verify", map[string]string{
		"razorpay_payment_id": "pay_1", "razorpay_subscription_id": subID,
		"razorpay_signature": hostedtest.PaymentSignature("pay_1", subID),
	}); code != 200 {
		t.Fatalf("verify: %d %s", code, out)
	}
	if b := e.billing(hid); b.Status != controlplane.StatusActive || b.Plan != "monthly" || b.CurrentPeriodEnd.IsZero() {
		t.Fatalf("after verify: %+v", b)
	}
	if code, _ := alice.do("POST", "/api/Members", map[string]string{"id": "m1", "name": "Asha"}); code != 200 {
		t.Errorf("edit when subscribed: want 200, got %d", code)
	}
	if code, _ := alice.checkout("annual"); code != 409 {
		t.Errorf("second live subscription: want 409, got %d", code)
	}

	// Cancel: paid-for time is honoured.
	if code, _ := alice.do("POST", "/api/billing/cancel", nil); code != 200 {
		t.Fatalf("cancel: %d", code)
	}
	if b := e.billing(hid); !b.CancelAtPeriodEnd || b.Status != controlplane.StatusActive {
		t.Errorf("after cancel: %+v", b)
	}
	e.rzp.SetStatus(subID, "cancelled", time.Time{}) // cycle ends
	e.webhook("subscription.cancelled", subID, false)
	if b := e.billing(hid); b.Status != controlplane.StatusCancelled {
		t.Errorf("after cancelled webhook: %+v", b)
	}
	if code, _ := alice.do("POST", "/api/Members", map[string]string{"id": "m2", "name": "Ravi"}); code != 200 {
		t.Errorf("cancelled but inside paid period: want 200, got %d", code)
	}
}

func TestBilling_SubscribingDuringTrialDefersTheFirstCharge(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	_, subID := alice.checkout("annual")
	sub, _ := e.rzp.FetchSubscription(context.Background(), subID)
	trialEnd := e.billing(hid).TrialEndsAt
	if sub.StartAt != trialEnd.Unix() {
		t.Errorf("first charge should wait for the trial (%v), start_at=%v", trialEnd, time.Unix(sub.StartAt, 0))
	}
}

func TestBilling_Webhooks(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	_, subID := alice.checkout("monthly")

	e.rzp.SetStatus(subID, "active", time.Now().AddDate(0, 1, 0))
	if code := e.webhook("subscription.activated", subID, true); code != 400 {
		t.Errorf("mis-signed webhook: want 400, got %d", code)
	}
	if b := e.billing(hid); b.Status == controlplane.StatusActive {
		t.Fatal("a mis-signed webhook changed billing state")
	}
	if code := e.webhook("subscription.activated", subID, false); code != 200 {
		t.Fatalf("webhook: %d", code)
	}
	if b := e.billing(hid); b.Status != controlplane.StatusActive {
		t.Fatalf("after activated: %+v", b)
	}

	// Out of order: a stale "charged" arrives after the subscription was
	// halted. State comes from Razorpay, not the event, so it stays lapsed.
	e.rzp.SetStatus(subID, "halted", time.Time{})
	e.webhook("subscription.halted", subID, false)
	e.webhook("subscription.charged", subID, false)
	if b := e.billing(hid); b.Status != controlplane.StatusLapsed {
		t.Errorf("out-of-order webhook regressed state: %+v", b)
	}

	// A webhook for a subscription Kosh doesn't know is acknowledged, not
	// retried forever.
	if code := e.webhook("subscription.activated", "sub_unknown", false); code != 200 {
		t.Errorf("unknown subscription: want 200, got %d", code)
	}
}

func TestBilling_AbandonedSubscriptionCantAffectTheHousehold(t *testing.T) {
	e := newHostedEnv(t)
	alice, hid := e.onboarded("alice@x.com")
	_, monthly := alice.checkout("monthly")
	_, annual := alice.checkout("annual") // switched plans before paying
	if monthly == annual {
		t.Fatal("switching plans should create a new subscription")
	}
	// The abandoned monthly one later goes active somehow: ignored.
	e.rzp.SetStatus(monthly, "active", time.Now().AddDate(0, 1, 0))
	e.webhook("subscription.activated", monthly, false)
	if b := e.billing(hid); b.Status == controlplane.StatusActive || b.SubscriptionID != annual {
		t.Errorf("abandoned subscription changed billing: %+v", b)
	}
}

func TestBilling_OnlyTheOwnerManagesTheSubscription(t *testing.T) {
	e := newHostedEnv(t)
	alice, _ := e.onboarded("alice@x.com")
	var inv struct {
		LinkPath string `json:"link_path"`
	}
	alice.json("POST", "/api/hosted/invites", map[string]string{"email": "bob@x.com"}, &inv)
	bob := e.browser()
	bob.login("bob@x.com")
	bob.do("POST", "/api/hosted/invites/accept", map[string]string{"token": strings.TrimPrefix(inv.LinkPath, "/join#")})
	for _, p := range []string{"/api/billing/checkout", "/api/billing/cancel", "/api/billing/verify"} {
		if code, _ := bob.do("POST", p, map[string]string{"plan": "monthly"}); code != 403 {
			t.Errorf("member POST %s: want 403, got %d", p, code)
		}
	}
	// Another household's payment can't be attached to yours.
	carol, _ := e.onboarded("carol@x.com")
	_, carolSub := carol.checkout("monthly")
	if code, _ := alice.do("POST", "/api/billing/verify", map[string]string{
		"razorpay_payment_id": "pay_9", "razorpay_subscription_id": carolSub,
		"razorpay_signature": hostedtest.PaymentSignature("pay_9", carolSub),
	}); code != 400 {
		t.Errorf("verifying another household's subscription: want 400, got %d", code)
	}
}

func TestBilling_CSPAllowsCheckoutOnlyWhenHosted(t *testing.T) {
	e := newHostedEnv(t)
	resp, _ := e.srv.Client().Get(e.srv.URL + "/api/health")
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "https://checkout.razorpay.com") {
		t.Errorf("hosted CSP missing Razorpay Checkout: %s", csp)
	}
}
