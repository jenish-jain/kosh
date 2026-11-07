package hostedtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"kosh/internal/billing"
)

// Fake gateway credentials.
const (
	RazorpayKeyID         = "rzp_test_fake"
	RazorpayKeySecret     = "fake-key-secret"
	RazorpayWebhookSecret = "fake-webhook-secret"
	PlanMonthlyID         = "plan_monthly_fake"
	PlanAnnualID          = "plan_annual_fake"
)

// Razorpay is an in-memory billing.Gateway that signs exactly like the real
// one, so verification code paths run for real.
type Razorpay struct {
	mu   sync.Mutex
	subs map[string]*billing.Subscription
	n    int
}

// NewRazorpay returns an empty fake.
func NewRazorpay() *Razorpay { return &Razorpay{subs: map[string]*billing.Subscription{}} }

func (f *Razorpay) KeyID() string { return RazorpayKeyID }

func (f *Razorpay) FetchPlan(_ context.Context, id string) (billing.Plan, error) {
	switch id {
	case PlanMonthlyID:
		return billing.Plan{ID: id, Period: "monthly", Amount: 14900, Name: "Kosh Family (monthly)"}, nil
	case PlanAnnualID:
		return billing.Plan{ID: id, Period: "yearly", Amount: 149900, Name: "Kosh Family (annual)"}, nil
	}
	return billing.Plan{}, errors.New("The id provided does not exist")
}

func (f *Razorpay) CreateSubscription(_ context.Context, planID string, _ int, startAt time.Time, _ map[string]string) (billing.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	s := &billing.Subscription{ID: fmt.Sprintf("sub_fake%d", f.n), PlanID: planID, Status: "created"}
	if !startAt.IsZero() {
		s.StartAt = startAt.Unix()
	}
	f.subs[s.ID] = s
	return *s, nil
}

func (f *Razorpay) FetchSubscription(_ context.Context, id string) (billing.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.subs[id]
	if !ok {
		return billing.Subscription{}, errors.New("The id provided does not exist")
	}
	return *s, nil
}

func (f *Razorpay) CancelSubscription(_ context.Context, id string, atCycleEnd bool) (billing.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.subs[id]
	if !ok {
		return billing.Subscription{}, errors.New("The id provided does not exist")
	}
	// Like Razorpay: an unstarted subscription cancels now; a running one
	// stays active until its cycle ends.
	if !atCycleEnd || s.Status == "created" || s.Status == "authenticated" {
		s.Status = "cancelled"
	}
	return *s, nil
}

func (f *Razorpay) VerifyPayment(paymentID, subscriptionID, signature string) bool {
	return signature == billing.Sign(RazorpayKeySecret, paymentID+"|"+subscriptionID)
}

func (f *Razorpay) VerifyWebhook(body []byte, signature string) bool {
	return signature == billing.Sign(RazorpayWebhookSecret, string(body))
}

// SetStatus moves a subscription to status (as Razorpay would on payment
// events) with a period ending at periodEnd.
func (f *Razorpay) SetStatus(id, status string, periodEnd time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.subs[id]; ok {
		s.Status = status
		if !periodEnd.IsZero() {
			s.CurrentStart = time.Now().Unix()
			s.CurrentEnd = periodEnd.Unix()
		}
	}
}

// PaymentSignature is what Checkout would hand the browser after paying.
func PaymentSignature(paymentID, subscriptionID string) string {
	return billing.Sign(RazorpayKeySecret, paymentID+"|"+subscriptionID)
}

// WebhookBody builds a signed subscription webhook, as Razorpay sends it.
func WebhookBody(event, subscriptionID string) (body []byte, signature string) {
	body, _ = json.Marshal(map[string]any{
		"entity": "event", "event": event,
		"payload": map[string]any{"subscription": map[string]any{"entity": map[string]any{"id": subscriptionID}}},
	})
	return body, billing.Sign(RazorpayWebhookSecret, string(body))
}

var _ billing.Gateway = (*Razorpay)(nil)
