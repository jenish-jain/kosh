// Package billing handles hosted-mode subscriptions: plans, checkout via
// Razorpay, webhook-driven subscription state, and what each state lets a
// household do.
package billing

import (
	"time"

	"kosh/internal/controlplane"
)

// Access is what a household's billing state allows.
type Access int

const (
	// ReadOnly: view and export everything, but no edits, AI, or new share
	// links. The data is always the customer's — it stays readable here and
	// in their own Google Sheet.
	ReadOnly Access = iota
	// Full: everything.
	Full
)

// AccessFor decides a household's access from its billing state at now.
func AccessFor(b controlplane.Billing, now time.Time) Access {
	switch b.Status {
	case controlplane.StatusTrialing:
		if now.Before(b.TrialEndsAt) {
			return Full
		}
	case controlplane.StatusActive, controlplane.StatusPastDue:
		// past_due: Razorpay is still retrying the charge — keep the
		// household working while that plays out.
		return Full
	case controlplane.StatusCancelled:
		// Cancelled at period end: paid-for time is honoured.
		if now.Before(b.CurrentPeriodEnd) {
			return Full
		}
	}
	return ReadOnly
}
