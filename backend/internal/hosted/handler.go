package hosted

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"kosh/internal/controlplane"
	"kosh/internal/share"
	"kosh/internal/tenant"
)

const (
	// TrialLength is how long a new household gets the full plan for free.
	TrialLength = 14 * 24 * time.Hour
	// MaxMembers caps a household (owner included), counting pending
	// invites, so one account can't fan out unlimited invites.
	MaxMembers  = 6
	inviteTTL   = 7 * 24 * time.Hour
	maxNameLen  = 60
	maxBodySize = 16 << 10
)

// Handler serves /api/hosted/*: the signed-in account's status,
// onboarding, Drive reconnect, and household membership.
type Handler struct {
	store        controlplane.Store
	sealer       *controlplane.TokenSealer
	google       Google
	resolver     *Resolver
	sessionEmail func(*http.Request) (string, bool)
	isDemo       func(*http.Request) bool
	now          func() time.Time

	// cancelSubscription stops billing immediately (account deletion).
	cancelSubscription func(ctx context.Context, subscriptionID string) error
	// entitlements reports a household's tier and features (billing).
	entitlements func(ctx context.Context, b controlplane.Billing) (string, map[string]bool, error)
	// isOperator reports whether an account runs the service.
	isOperator func(email string) bool
}

// SetEntitlements wires billing's tiers into the household views, so the
// UI can hide what the household's plan doesn't include. (The billing
// middleware enforces it regardless.)
func (h *Handler) SetEntitlements(fn func(ctx context.Context, b controlplane.Billing) (string, map[string]bool, error)) {
	h.entitlements = fn
}

// SetOperatorCheck lets /me tell the UI to show the operator settings.
func (h *Handler) SetOperatorCheck(fn func(email string) bool) {
	h.isOperator = fn
}

// SetSubscriptionCanceller wires billing into account deletion.
func (h *Handler) SetSubscriptionCanceller(fn func(ctx context.Context, subscriptionID string) error) {
	h.cancelSubscription = fn
}

// NewHandler constructs a Handler.
func NewHandler(store controlplane.Store, sealer *controlplane.TokenSealer, google Google, resolver *Resolver,
	sessionEmail func(*http.Request) (string, bool), isDemo func(*http.Request) bool) *Handler {
	return &Handler{store: store, sealer: sealer, google: google, resolver: resolver,
		sessionEmail: sessionEmail, isDemo: isDemo, now: time.Now}
}

// Routes is mounted at /api/hosted/ behind the normal session check.
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
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/hosted/"), "/")
	parts := strings.Split(path, "/")
	switch {
	case path == "me" && r.Method == http.MethodGet:
		h.me(w, r, email)
	case path == "households" && r.Method == http.MethodPost:
		h.onboard(w, r, email)
	case path == "reconnect" && r.Method == http.MethodPost:
		h.reconnect(w, r, email)
	case path == "household" && r.Method == http.MethodGet:
		h.household(w, r, email)
	case path == "household" && r.Method == http.MethodDelete:
		h.deleteHousehold(w, r, email)
	case path == "usage" && r.Method == http.MethodGet:
		h.usage(w, r, email)
	case path == "invites" && r.Method == http.MethodPost:
		h.invite(w, r, email)
	case path == "invites/accept" && r.Method == http.MethodPost:
		h.acceptInvite(w, r, email)
	case len(parts) == 2 && parts[0] == "invites" && r.Method == http.MethodDelete:
		h.cancelInvite(w, r, email, parts[1])
	case len(parts) == 2 && parts[0] == "members" && r.Method == http.MethodDelete:
		h.removeMember(w, r, email, parts[1])
	case path == "leave" && r.Method == http.MethodPost:
		h.leave(w, r, email)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// ── views ───────────────────────────────────────────────────────────────────

type billingView struct {
	Status            string    `json:"status"`
	Plan              string    `json:"plan"`
	TrialEndsAt       time.Time `json:"trial_ends_at"`
	CurrentPeriodEnd  time.Time `json:"current_period_end"`
	CancelAtPeriodEnd bool      `json:"cancel_at_period_end"`
}

type householdView struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Role           string          `json:"role"`
	OwnerEmail     string          `json:"owner_email"`
	NeedsReconnect bool            `json:"needs_reconnect"`
	SheetURL       string          `json:"sheet_url,omitempty"` // owner only — it's their Drive file
	Billing        billingView     `json:"billing"`
	Tier           string          `json:"tier,omitempty"`
	Features       map[string]bool `json:"features,omitempty"`
}

func (h *Handler) viewHousehold(ctx context.Context, hh controlplane.Household, role string) householdView {
	v := householdView{
		ID: hh.ID, Name: hh.Name, Role: role, OwnerEmail: hh.OwnerEmail, NeedsReconnect: hh.NeedsReconnect,
		Billing: billingView{
			Status: hh.Billing.Status, Plan: hh.Billing.Plan, TrialEndsAt: hh.Billing.TrialEndsAt,
			CurrentPeriodEnd: hh.Billing.CurrentPeriodEnd, CancelAtPeriodEnd: hh.Billing.CancelAtPeriodEnd,
		},
	}
	if role == controlplane.RoleOwner {
		v.SheetURL = "https://docs.google.com/spreadsheets/d/" + hh.SpreadsheetID
	}
	if h.entitlements != nil {
		var err error
		if v.Tier, v.Features, err = h.entitlements(ctx, hh.Billing); err != nil {
			// Only a display hint — the server still enforces the plan.
			log.Printf("⚠ hosted: loading features for %s: %v", hh.ID, err)
		}
	}
	return v
}

// membership returns the caller's member record and household.
func (h *Handler) membership(w http.ResponseWriter, r *http.Request, email string) (controlplane.Member, controlplane.Household, bool) {
	m, err := h.store.GetMember(r.Context(), email)
	if errors.Is(err, controlplane.ErrNotFound) {
		http.Error(w, "no_household", http.StatusForbidden)
		return m, controlplane.Household{}, false
	}
	if err != nil {
		serverError(w, "looking up membership", err)
		return m, controlplane.Household{}, false
	}
	hh, err := h.store.GetHousehold(r.Context(), m.HouseholdID)
	if err != nil {
		serverError(w, "loading household", err)
		return m, hh, false
	}
	return m, hh, true
}

func (h *Handler) ownerOnly(w http.ResponseWriter, r *http.Request, email string) (controlplane.Household, bool) {
	m, hh, ok := h.membership(w, r, email)
	if !ok {
		return hh, false
	}
	if m.Role != controlplane.RoleOwner {
		http.Error(w, "Only the household owner can do that.", http.StatusForbidden)
		return hh, false
	}
	return hh, true
}

// ── endpoints ───────────────────────────────────────────────────────────────

func (h *Handler) me(w http.ResponseWriter, r *http.Request, email string) {
	out := map[string]any{"email": email, "household": nil, "operator": h.isOperator != nil && h.isOperator(email)}
	m, err := h.store.GetMember(r.Context(), email)
	switch {
	case errors.Is(err, controlplane.ErrNotFound):
	case err != nil:
		serverError(w, "looking up membership", err)
		return
	default:
		hh, err := h.store.GetHousehold(r.Context(), m.HouseholdID)
		if err != nil {
			serverError(w, "loading household", err)
			return
		}
		out["household"] = h.viewHousehold(r.Context(), hh, m.Role)
	}
	writeJSON(w, out)
}

// onboard creates a household for a signed-in account that has none: a
// fresh Kosh spreadsheet in their own Drive, with a 14-day trial.
func (h *Handler) onboard(w http.ResponseWriter, r *http.Request, email string) {
	var body struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}
	name, ok := cleanName(body.Name)
	if !ok {
		http.Error(w, "Household name must be 1–60 characters.", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	if _, err := h.store.GetMember(ctx, email); err == nil {
		http.Error(w, "You're already part of a household.", http.StatusConflict)
		return
	} else if !errors.Is(err, controlplane.ErrNotFound) {
		serverError(w, "looking up membership", err)
		return
	}

	grant, ok := h.exchange(w, r, body.Code, email)
	if !ok {
		return
	}
	ts := h.google.TokenSource(grant.RefreshToken)
	sheetID, err := h.google.CreateSpreadsheet(ctx, ts, "Kosh — "+name)
	if err != nil {
		serverError(w, "creating your spreadsheet", err)
		return
	}
	api, err := h.google.Sheets(ctx, ts, sheetID)
	if err != nil {
		serverError(w, "opening your spreadsheet", err)
		return
	}
	id := newID("hh_")
	if err := tenant.Provision(tenant.New(id, api)); err != nil {
		serverError(w, "setting up your spreadsheet", err)
		return
	}
	sealed, err := h.sealer.Seal(grant.RefreshToken, id)
	if err != nil {
		serverError(w, "securing your Google access", err)
		return
	}
	now := h.now().UTC()
	hh := controlplane.Household{
		ID: id, Name: name, OwnerEmail: email, SpreadsheetID: sheetID, RefreshTokenEnc: sealed, CreatedAt: now,
		Billing: controlplane.Billing{Status: controlplane.StatusTrialing, TrialEndsAt: now.Add(TrialLength), UpdatedAt: now},
	}
	err = h.store.CreateHousehold(ctx, hh, controlplane.Member{Email: email, HouseholdID: id, Role: controlplane.RoleOwner, JoinedAt: now})
	if errors.Is(err, controlplane.ErrExists) {
		// Lost a race with another onboarding (or invite) for this account.
		// The new spreadsheet stays in their Drive, unused.
		http.Error(w, "You're already part of a household.", http.StatusConflict)
		return
	}
	if err != nil {
		serverError(w, "saving your household", err)
		return
	}
	writeJSON(w, map[string]any{"household": h.viewHousehold(r.Context(), hh, controlplane.RoleOwner)})
}

// reconnect stores a fresh Drive grant after Google revoked the old one.
func (h *Handler) reconnect(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.ownerOnly(w, r, email)
	if !ok {
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}
	grant, ok := h.exchange(w, r, body.Code, email)
	if !ok {
		return
	}
	sealed, err := h.sealer.Seal(grant.RefreshToken, hh.ID)
	if err != nil {
		serverError(w, "securing your Google access", err)
		return
	}
	var updated controlplane.Household
	err = h.store.UpdateHousehold(r.Context(), hh.ID, func(x *controlplane.Household) error {
		x.RefreshTokenEnc, x.NeedsReconnect = sealed, false
		updated = *x
		return nil
	})
	if err != nil {
		serverError(w, "saving your Google access", err)
		return
	}
	h.resolver.Invalidate(hh.ID)
	writeJSON(w, map[string]any{"household": h.viewHousehold(r.Context(), updated, controlplane.RoleOwner)})
}

// exchange trades the code for a grant and insists it belongs to the
// signed-in account: a Drive grant for some other Google account must never
// be attached to this household.
func (h *Handler) exchange(w http.ResponseWriter, r *http.Request, code, email string) (Grant, bool) {
	grant, err := h.google.Exchange(r.Context(), code)
	switch {
	case errors.Is(err, ErrScopeNotGranted):
		http.Error(w, "Kosh needs permission to create its spreadsheet in your Google Drive. Please try again and leave that box ticked.", http.StatusBadRequest)
		return grant, false
	case errors.Is(err, ErrNoRefreshToken):
		http.Error(w, "Google didn't grant lasting access. Remove Kosh at myaccount.google.com/permissions, then try again.", http.StatusBadRequest)
		return grant, false
	case err != nil:
		log.Printf("⚠ hosted: code exchange: %v", err)
		http.Error(w, "Couldn't connect to Google. Please try again.", http.StatusBadGateway)
		return grant, false
	}
	if controlplane.NormalizeEmail(grant.Email) != email {
		http.Error(w, "Please connect the same Google account you signed in with ("+email+").", http.StatusForbidden)
		return grant, false
	}
	return grant, true
}

type memberView struct {
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type inviteView struct {
	ID        string    `json:"id"` // the token hash — not the token
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handler) household(w http.ResponseWriter, r *http.Request, email string) {
	m, hh, ok := h.membership(w, r, email)
	if !ok {
		return
	}
	members, err := h.store.ListMembers(r.Context(), hh.ID)
	if err != nil {
		serverError(w, "listing members", err)
		return
	}
	out := map[string]any{"household": h.viewHousehold(r.Context(), hh, m.Role)}
	mv := make([]memberView, 0, len(members))
	for _, x := range members {
		mv = append(mv, memberView{Email: x.Email, Role: x.Role, JoinedAt: x.JoinedAt})
	}
	out["members"] = mv
	if m.Role == controlplane.RoleOwner {
		invites, err := h.store.ListInvites(r.Context(), hh.ID)
		if err != nil {
			serverError(w, "listing invites", err)
			return
		}
		iv := make([]inviteView, 0, len(invites))
		for _, x := range invites {
			if h.now().Before(x.ExpiresAt) {
				iv = append(iv, inviteView{ID: x.TokenHash, Email: x.Email, ExpiresAt: x.ExpiresAt})
			}
		}
		out["invites"] = iv
	}
	writeJSON(w, out)
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.ownerOnly(w, r, email)
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	invitee := controlplane.NormalizeEmail(body.Email)
	if addr, err := mail.ParseAddress(invitee); err != nil || addr.Address != invitee || len(invitee) > 254 {
		http.Error(w, "Enter a valid email address.", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	members, err := h.store.ListMembers(ctx, hh.ID)
	if err != nil {
		serverError(w, "listing members", err)
		return
	}
	invites, err := h.store.ListInvites(ctx, hh.ID)
	if err != nil {
		serverError(w, "listing invites", err)
		return
	}
	for _, m := range members {
		if m.Email == invitee {
			http.Error(w, "They're already in your household.", http.StatusConflict)
			return
		}
	}
	pending := 0
	for _, inv := range invites {
		if h.now().Before(inv.ExpiresAt) {
			pending++
		} else {
			h.store.DeleteInvite(ctx, hh.ID, inv.TokenHash) // tidy expired ones
		}
	}
	if len(members)+pending >= MaxMembers {
		http.Error(w, "A household can have up to 6 people, including pending invites.", http.StatusConflict)
		return
	}
	token, err := share.NewLinkToken()
	if err != nil {
		serverError(w, "creating invite", err)
		return
	}
	now := h.now().UTC()
	inv := controlplane.Invite{
		TokenHash: share.HashToken(token), HouseholdID: hh.ID, Email: invitee,
		InvitedBy: email, CreatedAt: now, ExpiresAt: now.Add(inviteTTL),
	}
	if err := h.store.CreateInvite(ctx, inv); err != nil {
		serverError(w, "saving invite", err)
		return
	}
	writeJSON(w, map[string]any{
		// Fragment, not path or query: the token never reaches server logs.
		"link_path": "/join#" + token,
		"invite":    inviteView{ID: inv.TokenHash, Email: inv.Email, ExpiresAt: inv.ExpiresAt},
	})
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request, email string) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	m, err := h.store.AcceptInvite(r.Context(), share.HashToken(body.Token), email, h.now())
	switch {
	case errors.Is(err, controlplane.ErrNotFound):
		http.Error(w, "This invite isn't valid. It may have expired, been cancelled, or be meant for a different Google account than "+email+".", http.StatusNotFound)
		return
	case errors.Is(err, controlplane.ErrExists):
		http.Error(w, "You're already part of a household. Leave it first to join another.", http.StatusConflict)
		return
	case err != nil:
		serverError(w, "accepting invite", err)
		return
	}
	hh, err := h.store.GetHousehold(r.Context(), m.HouseholdID)
	if err != nil {
		serverError(w, "loading household", err)
		return
	}
	writeJSON(w, map[string]any{"household": h.viewHousehold(r.Context(), hh, m.Role)})
}

func (h *Handler) cancelInvite(w http.ResponseWriter, r *http.Request, email, id string) {
	hh, ok := h.ownerOnly(w, r, email)
	if !ok {
		return
	}
	if err := h.store.DeleteInvite(r.Context(), hh.ID, id); err != nil {
		serverError(w, "cancelling invite", err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request, email, target string) {
	hh, ok := h.ownerOnly(w, r, email)
	if !ok {
		return
	}
	target = controlplane.NormalizeEmail(target)
	if target == email {
		http.Error(w, "The owner can't be removed. Delete the household from Account instead.", http.StatusBadRequest)
		return
	}
	m, err := h.store.GetMember(r.Context(), target)
	if errors.Is(err, controlplane.ErrNotFound) || (err == nil && m.HouseholdID != hh.ID) {
		// Never act on (or reveal) accounts outside the caller's household.
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, "looking up member", err)
		return
	}
	if err := h.store.RemoveMember(r.Context(), target); err != nil {
		serverError(w, "removing member", err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request, email string) {
	m, _, ok := h.membership(w, r, email)
	if !ok {
		return
	}
	if m.Role == controlplane.RoleOwner {
		http.Error(w, "The owner can't leave. Delete the household from Account instead.", http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveMember(r.Context(), email); err != nil {
		serverError(w, "leaving household", err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// deleteHousehold closes the account: billing stops, Kosh's Google access
// is revoked, and every control-plane record goes. The spreadsheet stays in
// the owner's Drive — it was always theirs.
func (h *Handler) deleteHousehold(w http.ResponseWriter, r *http.Request, email string) {
	hh, ok := h.ownerOnly(w, r, email)
	if !ok {
		return
	}
	var body struct {
		Confirm string `json:"confirm"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if confirm, _ := cleanName(body.Confirm); confirm != hh.Name {
		http.Error(w, "Type your household's name exactly to confirm.", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	// Billing first, and fatal: a deleted customer must never be charged.
	if id := hh.Billing.SubscriptionID; id != "" && h.cancelSubscription != nil {
		if err := h.cancelSubscription(ctx, id); err != nil {
			serverError(w, "cancelling your subscription", err)
			return
		}
	}
	if rt, err := h.sealer.Open(hh.RefreshTokenEnc, hh.ID); err == nil {
		if err := h.google.Revoke(ctx, rt); err != nil {
			// Not fatal: the token is about to be deleted with the record,
			// and the customer can also remove Kosh in their Google account.
			log.Printf("⚠ hosted: revoking Google access for %s: %v", hh.ID, err)
		}
	}
	if err := h.store.DeleteHousehold(ctx, hh.ID); err != nil {
		serverError(w, "deleting your household", err)
		return
	}
	h.resolver.Invalidate(hh.ID)
	writeJSON(w, map[string]string{"status": "ok", "sheet_url": "https://docs.google.com/spreadsheets/d/" + hh.SpreadsheetID})
}

func (h *Handler) usage(w http.ResponseWriter, r *http.Request, email string) {
	m, _, ok := h.membership(w, r, email)
	if !ok {
		return
	}
	now := h.now()
	used, err := h.store.Usage(r.Context(), m.HouseholdID, period(now))
	if err != nil {
		serverError(w, "loading usage", err)
		return
	}
	writeJSON(w, map[string]any{"period": period(now), "resets_at": nextReset(now), "used": used, "limits": MonthlyQuotas})
}

// ── helpers ─────────────────────────────────────────────────────────────────

func cleanName(s string) (string, bool) {
	s = strings.Join(strings.Fields(s), " ")
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= maxNameLen
}

func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failing is unrecoverable
	}
	return prefix + hex.EncodeToString(b)
}

// serverError logs the detail and returns a generic message, so internal
// errors (which can mention Google API responses) never reach clients.
func serverError(w http.ResponseWriter, doing string, err error) {
	log.Printf("⚠ hosted: %s: %v", doing, err)
	http.Error(w, "Something went wrong "+doing+". Please try again.", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
