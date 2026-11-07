// Package controlplane is the hosted service's own small datastore: who the
// customers are, which household (Google Sheet) each belongs to, billing
// state, and usage counters. It deliberately holds no financial data — that
// stays in each customer's own Sheet (see HOSTED_PLAN.md).
//
// Self-hosted Kosh never uses this package.
package controlplane

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrExists        = errors.New("already exists")
	ErrQuotaExceeded = errors.New("quota exceeded")
)

// Roles a member can hold in a household.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// Billing statuses. Only Store callers interpret them; the store just
// persists whatever the billing code sets.
const (
	StatusTrialing  = "trialing"
	StatusActive    = "active"
	StatusPastDue   = "past_due"
	StatusCancelled = "cancelled"
	StatusLapsed    = "lapsed"
)

// Household is one customer: a family sharing one Kosh Sheet.
type Household struct {
	ID            string
	Name          string
	OwnerEmail    string
	SpreadsheetID string
	// RefreshTokenEnc is the owner's Google OAuth refresh token (drive.file
	// scope), sealed with the control-plane key using the household ID as
	// associated data — so a ciphertext copied onto another household won't
	// decrypt.
	RefreshTokenEnc string
	// NeedsReconnect is set when Google rejects the refresh token (revoked
	// access, password change); the owner must re-grant access.
	NeedsReconnect bool
	CreatedAt      time.Time
	Billing        Billing
}

// Billing is a household's subscription state.
type Billing struct {
	Status            string
	Plan              string // "monthly" | "annual" | "" during trial
	TrialEndsAt       time.Time
	CurrentPeriodEnd  time.Time
	SubscriptionID    string // Razorpay subscription ID
	CancelAtPeriodEnd bool
	UpdatedAt         time.Time
}

// Member links a Google account to the one household it belongs to.
type Member struct {
	Email       string
	HouseholdID string
	Role        string
	JoinedAt    time.Time
}

// Invite lets someone join a household. Only a hash of the invite token is
// stored; the raw token lives in the link sent to the invitee.
type Invite struct {
	TokenHash   string
	HouseholdID string
	Email       string // the only Google account allowed to accept it
	InvitedBy   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// Store is the control-plane persistence interface. Implementations must
// make every method safe for concurrent use and atomic on its own.
type Store interface {
	// CreateHousehold creates h with owner as its first member. Fails with
	// ErrExists if the owner already belongs to a household.
	CreateHousehold(ctx context.Context, h Household, owner Member) error
	GetHousehold(ctx context.Context, id string) (Household, error)
	// UpdateHousehold applies fn to the current record transactionally; if
	// fn returns an error nothing is written and that error is returned.
	UpdateHousehold(ctx context.Context, id string, fn func(*Household) error) error
	// DeleteHousehold removes the household and everything indexed to it
	// (members, invites, share tokens, subscription index, usage).
	DeleteHousehold(ctx context.Context, id string) error

	GetMember(ctx context.Context, email string) (Member, error)
	ListMembers(ctx context.Context, householdID string) ([]Member, error)
	// AddMember fails with ErrExists if the email already belongs to any
	// household — an account is in at most one household.
	AddMember(ctx context.Context, m Member) error
	RemoveMember(ctx context.Context, email string) error

	CreateInvite(ctx context.Context, inv Invite) error
	ListInvites(ctx context.Context, householdID string) ([]Invite, error)
	// AcceptInvite atomically consumes the invite and adds email as a
	// member. Fails with ErrNotFound if the invite doesn't exist, has
	// expired, or was issued to a different email; with ErrExists if the
	// email already belongs to a household (the invite is left intact).
	AcceptInvite(ctx context.Context, tokenHash, email string, now time.Time) (Member, error)
	DeleteInvite(ctx context.Context, householdID, tokenHash string) error

	// Share-link index: which household a family-share link belongs to, so
	// an unauthenticated recipient request can be routed to the right Sheet.
	PutShareToken(ctx context.Context, tokenHash, householdID string) error
	ShareTokenHousehold(ctx context.Context, tokenHash string) (string, error)
	DeleteShareToken(ctx context.Context, tokenHash string) error

	// Subscription index for routing billing webhooks to a household.
	PutSubscription(ctx context.Context, subscriptionID, householdID string) error
	SubscriptionHousehold(ctx context.Context, subscriptionID string) (string, error)

	// ConsumeQuota atomically adds one use of kind in period (e.g.
	// "2026-11") if that keeps the count within limit, returning the new
	// count; otherwise it returns ErrQuotaExceeded and changes nothing.
	ConsumeQuota(ctx context.Context, householdID, kind, period string, limit int) (int, error)
	// Usage returns the counts recorded for period, by kind.
	Usage(ctx context.Context, householdID, period string) (map[string]int, error)

	// MarkEventProcessed records a webhook event ID, returning false if it
	// was already recorded — for idempotent webhook handling.
	MarkEventProcessed(ctx context.Context, eventID string) (bool, error)
}

// NormalizeEmail is the one canonical form emails are stored and looked
// up in.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
