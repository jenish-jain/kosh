package billing

import (
	"testing"
	"time"

	"kosh/internal/controlplane"
)

func TestTierFor(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	later, earlier := now.Add(time.Hour), now.Add(-time.Hour)
	cases := []struct {
		name string
		b    controlplane.Billing
		want string
	}{
		{"trial running", controlplane.Billing{Status: controlplane.StatusTrialing, TrialEndsAt: later}, TierTrial},
		{"trial over", controlplane.Billing{Status: controlplane.StatusTrialing, TrialEndsAt: earlier}, TierFree},
		{"active", controlplane.Billing{Status: controlplane.StatusActive}, TierPaid},
		{"payment retrying", controlplane.Billing{Status: controlplane.StatusPastDue}, TierPaid},
		{"cancelled, paid time left", controlplane.Billing{Status: controlplane.StatusCancelled, CurrentPeriodEnd: later}, TierPaid},
		{"cancelled, period over", controlplane.Billing{Status: controlplane.StatusCancelled, CurrentPeriodEnd: earlier}, TierFree},
		{"lapsed", controlplane.Billing{Status: controlplane.StatusLapsed, CurrentPeriodEnd: later}, TierFree},
		{"unknown status", controlplane.Billing{Status: "weird"}, TierFree},
		{"zero value", controlplane.Billing{}, TierFree},
	}
	for _, c := range cases {
		if got := TierFor(c.b, now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestApply(t *testing.T) {
	now := time.Now()
	end := now.Add(30 * 24 * time.Hour).Truncate(time.Second)
	cases := []struct {
		status string
		want   string
	}{
		{"active", controlplane.StatusActive},
		{"authenticated", controlplane.StatusActive},
		{"pending", controlplane.StatusPastDue},
		{"cancelled", controlplane.StatusCancelled},
		{"halted", controlplane.StatusLapsed},
		{"completed", controlplane.StatusLapsed},
		{"expired", controlplane.StatusLapsed},
		{"paused", controlplane.StatusLapsed},
	}
	for _, c := range cases {
		b := controlplane.Billing{Status: controlplane.StatusTrialing}
		Apply(&b, Subscription{Status: c.status, CurrentEnd: end.Unix()}, PlanMonthly, now)
		if b.Status != c.want || b.Plan != PlanMonthly || !b.CurrentPeriodEnd.Equal(end) {
			t.Errorf("%s: got %+v", c.status, b)
		}
	}
	b := controlplane.Billing{Status: controlplane.StatusTrialing}
	Apply(&b, Subscription{Status: "created"}, PlanMonthly, now)
	if b.Status != controlplane.StatusTrialing || b.Plan != "" {
		t.Errorf("an unfinished checkout must change nothing: %+v", b)
	}
	// Authenticated mid-trial: paid-up until the first charge.
	b = controlplane.Billing{}
	Apply(&b, Subscription{Status: "authenticated", ChargeAt: end.Unix()}, PlanAnnual, now)
	if !b.CurrentPeriodEnd.Equal(end) {
		t.Errorf("authenticated should use charge_at: %+v", b)
	}
}

func TestSignatures(t *testing.T) {
	r := NewRazorpay("key", "key-secret", "webhook-secret", nil)
	if !r.VerifyPayment("pay_1", "sub_1", Sign("key-secret", "pay_1|sub_1")) {
		t.Error("valid payment signature rejected")
	}
	for _, bad := range []string{"", "zz", Sign("key-secret", "pay_1|sub_2"), Sign("webhook-secret", "pay_1|sub_1")} {
		if r.VerifyPayment("pay_1", "sub_1", bad) {
			t.Errorf("payment signature %q accepted", bad)
		}
	}
	body := []byte(`{"event":"subscription.charged"}`)
	if !r.VerifyWebhook(body, Sign("webhook-secret", string(body))) {
		t.Error("valid webhook rejected")
	}
	if r.VerifyWebhook(append(body, ' '), Sign("webhook-secret", string(body))) {
		t.Error("tampered webhook body accepted")
	}
	if NewRazorpay("k", "s", "", nil).VerifyWebhook(body, Sign("", string(body))) {
		t.Error("an unset webhook secret must never verify")
	}
}
