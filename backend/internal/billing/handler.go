package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"kosh/internal/controlplane"
)

// Plan keys customers choose between.
const (
	PlanMonthly = "monthly"
	PlanAnnual  = "annual"
)

// totalCounts is how many billing cycles a subscription is created for —
// effectively open-ended; customers cancel whenever they like.
var totalCounts = map[string]int{PlanMonthly: 120, PlanAnnual: 10}

// minDeferredStart: subscribing in the last few minutes of a trial charges
// immediately rather than scheduling a start that's already past.
const minDeferredStart = 10 * time.Minute

const maxWebhookBody = 256 << 10

// errStaleSubscription: an event for a subscription that is no longer the
// household's current one (e.g. an abandoned checkout) — ignored.
var errStaleSubscription = errors.New("stale subscription")

// Handler serves /api/billing/* and enforces read-only access for lapsed
// households.
type Handler struct {
	store        controlplane.Store
	gw           Gateway
	planIDs      map[string]string // plan key -> Razorpay plan ID
	plans        map[string]Plan
	sessionEmail func(*http.Request) (string, bool)
	isDemo       func(*http.Request) bool
	now          func() time.Time
}

// NewHandler loads the configured plans from the gateway up front, so a
// misconfigured plan ID fails startup rather than a customer's checkout.
func NewHandler(ctx context.Context, store controlplane.Store, gw Gateway, planIDs map[string]string,
	sessionEmail func(*http.Request) (string, bool), isDemo func(*http.Request) bool) (*Handler, error) {
	h := &Handler{store: store, gw: gw, planIDs: planIDs, plans: map[string]Plan{}, sessionEmail: sessionEmail, isDemo: isDemo, now: time.Now}
	for key, id := range planIDs {
		p, err := gw.FetchPlan(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("loading %s plan %s: %w", key, id, err)
		}
		h.plans[key] = p
	}
	return h, nil
}

func (h *Handler) planKey(planID string) string {
	for k, id := range h.planIDs {
		if id == planID {
			return k
		}
	}
	return ""
}

// Routes serves the session-protected /api/billing/* endpoints.
func (h *Handler) Routes(w http.ResponseWriter, r *http.Request) {
	if h.isDemo(r) {
		http.Error(w, "Not available in the demo.", http.StatusForbidden)
		return
	}
	email, ok := h.sessionEmail(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	email = controlplane.NormalizeEmail(email)
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	switch path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/billing/"), "/"); {
	case path == "plans" && r.Method == http.MethodGet:
		h.listPlans(w)
	case path == "checkout" && r.Method == http.MethodPost:
		h.checkout(w, r, email)
	case path == "verify" && r.Method == http.MethodPost:
		h.verify(w, r, email)
	case path == "cancel" && r.Method == http.MethodPost:
		h.cancel(w, r, email)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) listPlans(w http.ResponseWriter) {
	type planView struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Amount int64  `json:"amount"` // paise
		Period string `json:"period"`
	}
	out := []planView{}
	for _, k := range []string{PlanMonthly, PlanAnnual} {
		if p, ok := h.plans[k]; ok {
			out = append(out, planView{ID: k, Name: p.Name, Amount: p.Amount, Period: p.Period})
		}
	}
	writeJSON(w, map[string]any{"key_id": h.gw.KeyID(), "plans": out})
}

func (h *Handler) owner(w http.ResponseWriter, r *http.Request, email string) (controlplane.Household, bool) {
	m, err := h.store.GetMember(r.Context(), email)
	if errors.Is(err, controlplane.ErrNotFound) {
		http.Error(w, "no_household", http.StatusForbidden)
		return controlplane.Household{}, false
	}
	if err != nil {
		serverError(w, "looking up membership", err)
		return controlplane.Household{}, false
	}
	if m.Role != controlplane.RoleOwner {
		http.Error(w, "Only the household owner manages the subscription.", http.StatusForbidden)
		return controlplane.Household{}, false
	}
	hh, err := h.store.GetHousehold(r.Context(), m.HouseholdID)
	if err != nil {
		serverError(w, "loading household", err)
		return hh, false
	}
	return hh, true
}

// live reports whether a Razorpay subscription status still bills (or is
// about to) — such a subscription must not be duplicated.
func live(status string) bool {
	return status == "active" || status == "authenticated" || status == "pending"
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.owner(w, r, email)
	if !ok {
		return
	}
	var body struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	planID, ok := h.planIDs[body.Plan]
	if !ok {
		http.Error(w, "unknown plan", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	if id := hh.Billing.SubscriptionID; id != "" {
		existing, err := h.gw.FetchSubscription(ctx, id)
		if err != nil {
			serverError(w, "checking your current subscription", err)
			return
		}
		switch {
		case live(existing.Status):
			http.Error(w, "You already have an active subscription.", http.StatusConflict)
			return
		case existing.Status == "created" && existing.PlanID == planID:
			// An unfinished checkout for the same plan: reuse it.
			writeJSON(w, h.checkoutView(existing.ID, body.Plan, email))
			return
		case existing.Status == "created":
			if _, err := h.gw.CancelSubscription(ctx, existing.ID, false); err != nil {
				log.Printf("⚠ billing: cancelling abandoned subscription %s: %v", existing.ID, err)
			}
		}
	}

	// Subscribing mid-trial: the first charge waits for the trial to end,
	// so no paid-for trial days are lost.
	var startAt time.Time
	if hh.Billing.Status == controlplane.StatusTrialing && hh.Billing.TrialEndsAt.After(h.now().Add(minDeferredStart)) {
		startAt = hh.Billing.TrialEndsAt
	}
	sub, err := h.gw.CreateSubscription(ctx, planID, totalCounts[body.Plan], startAt, map[string]string{"household_id": hh.ID})
	if err != nil {
		serverError(w, "starting checkout", err)
		return
	}
	// Index before recording, so a webhook arriving immediately can be routed.
	if err := h.store.PutSubscription(ctx, sub.ID, hh.ID); err != nil {
		serverError(w, "starting checkout", err)
		return
	}
	err = h.store.UpdateHousehold(ctx, hh.ID, func(x *controlplane.Household) error {
		x.Billing.SubscriptionID = sub.ID
		x.Billing.CancelAtPeriodEnd = false
		x.Billing.UpdatedAt = h.now().UTC()
		return nil
	})
	if err != nil {
		serverError(w, "starting checkout", err)
		return
	}
	writeJSON(w, h.checkoutView(sub.ID, body.Plan, email))
}

func (h *Handler) checkoutView(subID, plan, email string) map[string]any {
	return map[string]any{"key_id": h.gw.KeyID(), "subscription_id": subID, "plan": plan, "email": email}
}

// verify confirms a Checkout success callback and syncs state immediately,
// so the customer doesn't wait for the webhook. The webhook remains the
// authority; this is the same idempotent sync.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.owner(w, r, email)
	if !ok {
		return
	}
	var body struct {
		PaymentID      string `json:"razorpay_payment_id"`
		SubscriptionID string `json:"razorpay_subscription_id"`
		Signature      string `json:"razorpay_signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.SubscriptionID == "" || body.SubscriptionID != hh.Billing.SubscriptionID {
		http.Error(w, "That payment isn't for this household's subscription.", http.StatusBadRequest)
		return
	}
	if !h.gw.VerifyPayment(body.PaymentID, body.SubscriptionID, body.Signature) {
		http.Error(w, "Payment signature didn't verify.", http.StatusBadRequest)
		return
	}
	billing, err := h.sync(r.Context(), hh.ID, body.SubscriptionID)
	if err != nil {
		serverError(w, "confirming your subscription", err)
		return
	}
	writeJSON(w, map[string]any{"billing": billingView(billing)})
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.owner(w, r, email)
	if !ok {
		return
	}
	id := hh.Billing.SubscriptionID
	if id == "" {
		http.Error(w, "There's no subscription to cancel.", http.StatusBadRequest)
		return
	}
	if _, err := h.gw.CancelSubscription(r.Context(), id, true); err != nil {
		serverError(w, "cancelling your subscription", err)
		return
	}
	err := h.store.UpdateHousehold(r.Context(), hh.ID, func(x *controlplane.Household) error {
		x.Billing.CancelAtPeriodEnd = true
		return nil
	})
	if err != nil {
		serverError(w, "cancelling your subscription", err)
		return
	}
	billing, err := h.sync(r.Context(), hh.ID, id)
	if err != nil {
		serverError(w, "cancelling your subscription", err)
		return
	}
	writeJSON(w, map[string]any{"billing": billingView(billing)})
}

// Webhook serves POST /api/billing/webhook — unauthenticated; trusted only
// once its signature verifies. The event body is used solely to find which
// subscription changed: state is then re-fetched from Razorpay, so events
// arriving out of order (or replayed) can't move a household backwards.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if !h.gw.VerifyWebhook(raw, r.Header.Get("X-Razorpay-Signature")) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	eventID := r.Header.Get("X-Razorpay-Event-Id")
	if eventID == "" {
		sum := sha256.Sum256(raw)
		eventID = "body-" + hex.EncodeToString(sum[:])
	}
	var ev struct {
		Event   string `json:"event"`
		Payload struct {
			Subscription struct {
				Entity struct {
					ID string `json:"id"`
				} `json:"entity"`
			} `json:"subscription"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	subID := ev.Payload.Subscription.Entity.ID
	if !strings.HasPrefix(ev.Event, "subscription.") || subID == "" {
		w.WriteHeader(http.StatusOK) // not an event Kosh acts on
		return
	}
	ctx := r.Context()
	hid, err := h.store.SubscriptionHousehold(ctx, subID)
	if errors.Is(err, controlplane.ErrNotFound) {
		log.Printf("⚠ billing webhook %s: unknown subscription %s", ev.Event, subID)
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		serverError(w, "routing webhook", err)
		return
	}
	if _, err := h.sync(ctx, hid, subID); err != nil && !errors.Is(err, errStaleSubscription) {
		// 5xx: Razorpay retries, and nothing was marked processed.
		serverError(w, "applying webhook", err)
		return
	}
	// Recorded only after success; a duplicate arriving meanwhile just
	// repeats the same idempotent sync.
	if _, err := h.store.MarkEventProcessed(ctx, eventID); err != nil {
		log.Printf("⚠ billing webhook: recording event %s: %v", eventID, err)
	}
	w.WriteHeader(http.StatusOK)
}

// sync fetches the subscription from Razorpay and applies its state to the
// household, if it's still the household's current subscription.
func (h *Handler) sync(ctx context.Context, householdID, subID string) (controlplane.Billing, error) {
	sub, err := h.gw.FetchSubscription(ctx, subID)
	if err != nil {
		return controlplane.Billing{}, err
	}
	var out controlplane.Billing
	err = h.store.UpdateHousehold(ctx, householdID, func(x *controlplane.Household) error {
		if x.Billing.SubscriptionID != sub.ID {
			return errStaleSubscription
		}
		Apply(&x.Billing, sub, h.planKey(sub.PlanID), h.now())
		out = x.Billing
		return nil
	})
	return out, err
}

// Apply maps a Razorpay subscription onto billing state.
func Apply(b *controlplane.Billing, sub Subscription, planKey string, now time.Time) {
	switch sub.Status {
	case "created":
		return // checkout not completed — nothing has changed
	case "active", "authenticated":
		b.Status = controlplane.StatusActive
	case "pending":
		b.Status = controlplane.StatusPastDue
	case "cancelled":
		b.Status = controlplane.StatusCancelled
	default: // halted, completed, expired, paused
		b.Status = controlplane.StatusLapsed
	}
	if planKey != "" {
		b.Plan = planKey
	}
	switch {
	case sub.CurrentEnd > 0:
		b.CurrentPeriodEnd = time.Unix(sub.CurrentEnd, 0).UTC()
	case sub.ChargeAt > 0:
		// Authenticated during a trial: paid-up until the first charge.
		b.CurrentPeriodEnd = time.Unix(sub.ChargeAt, 0).UTC()
	}
	b.UpdatedAt = now.UTC()
}

// CancelNow stops a household's subscription immediately (account
// deletion). A subscription that has already ended is fine.
func (h *Handler) CancelNow(ctx context.Context, subID string) error {
	sub, err := h.gw.FetchSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if sub.Status == "cancelled" || sub.Status == "completed" || sub.Status == "expired" {
		return nil
	}
	_, err = h.gw.CancelSubscription(ctx, subID, false)
	return err
}

// Enforce wraps the server: a household without Full access (trial over,
// subscription lapsed) can still read and export everything, and revoke or
// delete family-share links — but not edit data, use AI, or create new
// share links. Answers 402 subscription_required.
func (h *Handler) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !needsFullAccess(r) || h.isDemo(r) {
			next.ServeHTTP(w, r)
			return
		}
		email, ok := h.sessionEmail(r)
		if !ok {
			next.ServeHTTP(w, r) // the session check downstream rejects it
			return
		}
		m, err := h.store.GetMember(r.Context(), email)
		if err != nil {
			next.ServeHTTP(w, r) // no household: the resolver answers
			return
		}
		hh, err := h.store.GetHousehold(r.Context(), m.HouseholdID)
		if err != nil {
			serverError(w, "checking your subscription", err)
			return
		}
		if AccessFor(hh.Billing, h.now()) != Full {
			http.Error(w, "subscription_required", http.StatusPaymentRequired)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// needsFullAccess: state-changing API calls, minus the ones a lapsed
// household must always keep (account, billing, recipients' own views,
// and revoking or deleting share links).
func needsFullAccess(r *http.Request) bool {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/api/") {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	for _, prefix := range []string{"/api/auth/", "/api/hosted/", "/api/billing/", "/api/share/", "/api/dev/"} {
		if strings.HasPrefix(p, prefix) {
			return false
		}
	}
	if strings.HasPrefix(p, "/api/shares/") && (r.Method == http.MethodDelete || strings.HasSuffix(p, "/revoke")) {
		return false
	}
	return true
}

type billingJSON struct {
	Status            string    `json:"status"`
	Plan              string    `json:"plan"`
	TrialEndsAt       time.Time `json:"trial_ends_at"`
	CurrentPeriodEnd  time.Time `json:"current_period_end"`
	CancelAtPeriodEnd bool      `json:"cancel_at_period_end"`
}

func billingView(b controlplane.Billing) billingJSON {
	return billingJSON{Status: b.Status, Plan: b.Plan, TrialEndsAt: b.TrialEndsAt, CurrentPeriodEnd: b.CurrentPeriodEnd, CancelAtPeriodEnd: b.CancelAtPeriodEnd}
}

func serverError(w http.ResponseWriter, doing string, err error) {
	log.Printf("⚠ billing: %s: %v", doing, err)
	http.Error(w, "Something went wrong "+doing+". Please try again.", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
