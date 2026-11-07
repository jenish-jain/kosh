package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Subscription is the subset of a Razorpay subscription Kosh uses.
type Subscription struct {
	ID           string `json:"id"`
	PlanID       string `json:"plan_id"`
	Status       string `json:"status"`
	CurrentStart int64  `json:"current_start"`
	CurrentEnd   int64  `json:"current_end"`
	ChargeAt     int64  `json:"charge_at"`
	StartAt      int64  `json:"start_at"`
	ShortURL     string `json:"short_url"`
}

// Plan is the subset of a Razorpay plan Kosh shows customers.
type Plan struct {
	ID     string
	Period string // "monthly" | "yearly"
	Amount int64  // in paise
	Name   string
}

// Gateway is the payment provider, behind an interface for tests and the
// hosteddev build.
type Gateway interface {
	KeyID() string
	FetchPlan(ctx context.Context, planID string) (Plan, error)
	CreateSubscription(ctx context.Context, planID string, totalCount int, startAt time.Time, notes map[string]string) (Subscription, error)
	FetchSubscription(ctx context.Context, id string) (Subscription, error)
	// CancelSubscription cancels at the end of the current billing cycle.
	CancelSubscription(ctx context.Context, id string) (Subscription, error)
	// VerifyPayment checks the signature Checkout returns after a
	// subscription's authorization payment.
	VerifyPayment(paymentID, subscriptionID, signature string) bool
	// VerifyWebhook checks a webhook body's X-Razorpay-Signature.
	VerifyWebhook(body []byte, signature string) bool
}

// Razorpay is the real Gateway.
type Razorpay struct {
	keyID, keySecret, webhookSecret string
	base                            string
	http                            *http.Client
}

// NewRazorpay returns a client for the Razorpay REST API.
func NewRazorpay(keyID, keySecret, webhookSecret string, client *http.Client) *Razorpay {
	return &Razorpay{keyID: keyID, keySecret: keySecret, webhookSecret: webhookSecret, base: "https://api.razorpay.com/v1", http: client}
}

func (r *Razorpay) KeyID() string { return r.keyID }

func (r *Razorpay) call(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.base+path, rd)
	if err != nil {
		return err
	}
	req.SetBasicAuth(r.keyID, r.keySecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return fmt.Errorf("razorpay %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Description string `json:"description"`
			} `json:"error"`
		}
		json.Unmarshal(raw, &e)
		return fmt.Errorf("razorpay %s %s: HTTP %d: %s", method, path, resp.StatusCode, e.Error.Description)
	}
	return json.Unmarshal(raw, out)
}

func (r *Razorpay) FetchPlan(ctx context.Context, planID string) (Plan, error) {
	var p struct {
		ID     string `json:"id"`
		Period string `json:"period"`
		Item   struct {
			Name   string `json:"name"`
			Amount int64  `json:"amount"`
		} `json:"item"`
	}
	if err := r.call(ctx, http.MethodGet, "/plans/"+url.PathEscape(planID), nil, &p); err != nil {
		return Plan{}, err
	}
	return Plan{ID: p.ID, Period: p.Period, Amount: p.Item.Amount, Name: p.Item.Name}, nil
}

func (r *Razorpay) CreateSubscription(ctx context.Context, planID string, totalCount int, startAt time.Time, notes map[string]string) (Subscription, error) {
	body := map[string]any{"plan_id": planID, "total_count": totalCount, "quantity": 1, "customer_notify": 1, "notes": notes}
	if !startAt.IsZero() {
		body["start_at"] = startAt.Unix()
	}
	var s Subscription
	err := r.call(ctx, http.MethodPost, "/subscriptions", body, &s)
	return s, err
}

func (r *Razorpay) FetchSubscription(ctx context.Context, id string) (Subscription, error) {
	var s Subscription
	err := r.call(ctx, http.MethodGet, "/subscriptions/"+url.PathEscape(id), nil, &s)
	return s, err
}

func (r *Razorpay) CancelSubscription(ctx context.Context, id string) (Subscription, error) {
	var s Subscription
	err := r.call(ctx, http.MethodPost, "/subscriptions/"+url.PathEscape(id)+"/cancel", map[string]int{"cancel_at_cycle_end": 1}, &s)
	return s, err
}

func (r *Razorpay) VerifyPayment(paymentID, subscriptionID, signature string) bool {
	return verifyHMAC(r.keySecret, paymentID+"|"+subscriptionID, signature)
}

func (r *Razorpay) VerifyWebhook(body []byte, signature string) bool {
	return verifyHMAC(r.webhookSecret, string(body), signature)
}

// Sign returns the hex HMAC-SHA256 Razorpay uses for both checkout and
// webhook signatures. Exported for tests and fakes.
func Sign(secret, msg string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func verifyHMAC(secret, msg, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	want, err := hex.DecodeString(Sign(secret, msg))
	if err != nil {
		return false
	}
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(want, got)
}

// ErrAlreadySubscribed means the household already has a live subscription.
var ErrAlreadySubscribed = errors.New("already subscribed")
