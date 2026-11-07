package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Collection names.
const (
	colHouseholds    = "households"
	colMembers       = "members"
	colInvites       = "invites"
	colShareTokens   = "shareTokens"
	colSubscriptions = "subscriptions"
	colUsage         = "usage"
	colEvents        = "webhookEvents"
)

// eventRetention is how long processed webhook IDs are kept. Razorpay
// retries for up to 24h; a Firestore TTL policy on expireAt does cleanup.
const eventRetention = 7 * 24 * time.Hour

// Firestore is the production Store.
type Firestore struct {
	c *firestore.Client
}

// NewFirestore connects to the named Firestore database in project.
// Honours FIRESTORE_EMULATOR_HOST for local testing.
func NewFirestore(ctx context.Context, project, database string) (*Firestore, error) {
	c, err := firestore.NewClientWithDatabase(ctx, project, database)
	if err != nil {
		return nil, fmt.Errorf("connecting to Firestore: %w", err)
	}
	return &Firestore{c: c}, nil
}

// Close releases the client.
func (f *Firestore) Close() error { return f.c.Close() }

// Document shapes. Kept separate from the public types so storage field
// names are explicit and stable.
type fsHousehold struct {
	Name            string    `firestore:"name"`
	OwnerEmail      string    `firestore:"ownerEmail"`
	SpreadsheetID   string    `firestore:"spreadsheetId"`
	RefreshTokenEnc string    `firestore:"refreshTokenEnc"`
	NeedsReconnect  bool      `firestore:"needsReconnect"`
	CreatedAt       time.Time `firestore:"createdAt"`
	Billing         fsBilling `firestore:"billing"`
}

type fsBilling struct {
	Status            string    `firestore:"status"`
	Plan              string    `firestore:"plan"`
	TrialEndsAt       time.Time `firestore:"trialEndsAt"`
	CurrentPeriodEnd  time.Time `firestore:"currentPeriodEnd"`
	SubscriptionID    string    `firestore:"subscriptionId"`
	CancelAtPeriodEnd bool      `firestore:"cancelAtPeriodEnd"`
	UpdatedAt         time.Time `firestore:"updatedAt"`
}

type fsMember struct {
	Email       string    `firestore:"email"`
	HouseholdID string    `firestore:"householdId"`
	Role        string    `firestore:"role"`
	JoinedAt    time.Time `firestore:"joinedAt"`
}

type fsInvite struct {
	HouseholdID string    `firestore:"householdId"`
	Email       string    `firestore:"email"`
	InvitedBy   string    `firestore:"invitedBy"`
	CreatedAt   time.Time `firestore:"createdAt"`
	ExpiresAt   time.Time `firestore:"expiresAt"`
}

type fsIndex struct {
	HouseholdID string `firestore:"householdId"`
}

type fsUsage struct {
	HouseholdID string         `firestore:"householdId"`
	Period      string         `firestore:"period"`
	Counts      map[string]int `firestore:"counts"`
}

func toFSHousehold(h Household) fsHousehold {
	b := h.Billing
	return fsHousehold{
		Name: h.Name, OwnerEmail: h.OwnerEmail, SpreadsheetID: h.SpreadsheetID,
		RefreshTokenEnc: h.RefreshTokenEnc, NeedsReconnect: h.NeedsReconnect, CreatedAt: h.CreatedAt,
		Billing: fsBilling{
			Status: b.Status, Plan: b.Plan, TrialEndsAt: b.TrialEndsAt, CurrentPeriodEnd: b.CurrentPeriodEnd,
			SubscriptionID: b.SubscriptionID, CancelAtPeriodEnd: b.CancelAtPeriodEnd, UpdatedAt: b.UpdatedAt,
		},
	}
}

func fromFSHousehold(id string, d fsHousehold) Household {
	b := d.Billing
	return Household{
		ID: id, Name: d.Name, OwnerEmail: d.OwnerEmail, SpreadsheetID: d.SpreadsheetID,
		RefreshTokenEnc: d.RefreshTokenEnc, NeedsReconnect: d.NeedsReconnect, CreatedAt: d.CreatedAt,
		Billing: Billing{
			Status: b.Status, Plan: b.Plan, TrialEndsAt: b.TrialEndsAt, CurrentPeriodEnd: b.CurrentPeriodEnd,
			SubscriptionID: b.SubscriptionID, CancelAtPeriodEnd: b.CancelAtPeriodEnd, UpdatedAt: b.UpdatedAt,
		},
	}
}

func fromFSMember(d fsMember) Member {
	return Member{Email: d.Email, HouseholdID: d.HouseholdID, Role: d.Role, JoinedAt: d.JoinedAt}
}

// memberDocID keys member docs by a hash of the email: emails can legally
// contain characters Firestore document IDs can't ('/').
func memberDocID(email string) string {
	h := sha256.Sum256([]byte(NormalizeEmail(email)))
	return hex.EncodeToString(h[:])
}

func isCode(err error, c codes.Code) bool { return status.Code(err) == c }

func (f *Firestore) member(email string) *firestore.DocumentRef {
	return f.c.Collection(colMembers).Doc(memberDocID(email))
}

func (f *Firestore) CreateHousehold(ctx context.Context, h Household, owner Member) error {
	owner.Email = NormalizeEmail(owner.Email)
	return f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(f.member(owner.Email)); err == nil {
			return ErrExists
		} else if !isCode(err, codes.NotFound) {
			return err
		}
		hRef := f.c.Collection(colHouseholds).Doc(h.ID)
		if err := tx.Create(hRef, toFSHousehold(h)); err != nil {
			return err
		}
		return tx.Create(f.member(owner.Email), fsMember{
			Email: owner.Email, HouseholdID: owner.HouseholdID, Role: owner.Role, JoinedAt: owner.JoinedAt,
		})
	})
}

func (f *Firestore) GetHousehold(ctx context.Context, id string) (Household, error) {
	snap, err := f.c.Collection(colHouseholds).Doc(id).Get(ctx)
	if isCode(err, codes.NotFound) {
		return Household{}, ErrNotFound
	}
	if err != nil {
		return Household{}, err
	}
	var d fsHousehold
	if err := snap.DataTo(&d); err != nil {
		return Household{}, err
	}
	return fromFSHousehold(id, d), nil
}

func (f *Firestore) UpdateHousehold(ctx context.Context, id string, fn func(*Household) error) error {
	ref := f.c.Collection(colHouseholds).Doc(id)
	return f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if isCode(err, codes.NotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var d fsHousehold
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		h := fromFSHousehold(id, d)
		if err := fn(&h); err != nil {
			return err
		}
		return tx.Set(ref, toFSHousehold(h))
	})
}

func (f *Firestore) DeleteHousehold(ctx context.Context, id string) error {
	var refs []*firestore.DocumentRef
	for _, col := range []string{colMembers, colInvites, colShareTokens, colSubscriptions, colUsage} {
		docs, err := f.c.Collection(col).Where("householdId", "==", id).Documents(ctx).GetAll()
		if err != nil {
			return fmt.Errorf("listing %s for deletion: %w", col, err)
		}
		for _, d := range docs {
			refs = append(refs, d.Ref)
		}
	}
	// The household doc goes last: if a delete fails part-way, the
	// household still exists and a retry finds the rest.
	refs = append(refs, f.c.Collection(colHouseholds).Doc(id))
	bw := f.c.BulkWriter(ctx)
	jobs := make([]*firestore.BulkWriterJob, 0, len(refs))
	for _, ref := range refs {
		j, err := bw.Delete(ref)
		if err != nil {
			bw.End()
			return err
		}
		jobs = append(jobs, j)
	}
	bw.End()
	for _, j := range jobs {
		if _, err := j.Results(); err != nil && !isCode(err, codes.NotFound) {
			return err
		}
	}
	return nil
}

func (f *Firestore) GetMember(ctx context.Context, email string) (Member, error) {
	snap, err := f.member(email).Get(ctx)
	if isCode(err, codes.NotFound) {
		return Member{}, ErrNotFound
	}
	if err != nil {
		return Member{}, err
	}
	var d fsMember
	if err := snap.DataTo(&d); err != nil {
		return Member{}, err
	}
	return fromFSMember(d), nil
}

func (f *Firestore) ListMembers(ctx context.Context, householdID string) ([]Member, error) {
	docs, err := f.c.Collection(colMembers).Where("householdId", "==", householdID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(docs))
	for _, d := range docs {
		var m fsMember
		if err := d.DataTo(&m); err != nil {
			return nil, err
		}
		out = append(out, fromFSMember(m))
	}
	sortMembers(out)
	return out, nil
}

func (f *Firestore) AddMember(ctx context.Context, m Member) error {
	m.Email = NormalizeEmail(m.Email)
	_, err := f.member(m.Email).Create(ctx, fsMember{Email: m.Email, HouseholdID: m.HouseholdID, Role: m.Role, JoinedAt: m.JoinedAt})
	if isCode(err, codes.AlreadyExists) {
		return ErrExists
	}
	return err
}

func (f *Firestore) RemoveMember(ctx context.Context, email string) error {
	_, err := f.member(email).Delete(ctx)
	return err
}

func (f *Firestore) CreateInvite(ctx context.Context, inv Invite) error {
	_, err := f.c.Collection(colInvites).Doc(inv.TokenHash).Create(ctx, fsInvite{
		HouseholdID: inv.HouseholdID, Email: NormalizeEmail(inv.Email), InvitedBy: inv.InvitedBy,
		CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
	})
	if isCode(err, codes.AlreadyExists) {
		return ErrExists
	}
	return err
}

func (f *Firestore) ListInvites(ctx context.Context, householdID string) ([]Invite, error) {
	docs, err := f.c.Collection(colInvites).Where("householdId", "==", householdID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]Invite, 0, len(docs))
	for _, d := range docs {
		var inv fsInvite
		if err := d.DataTo(&inv); err != nil {
			return nil, err
		}
		out = append(out, Invite{
			TokenHash: d.Ref.ID, HouseholdID: inv.HouseholdID, Email: inv.Email, InvitedBy: inv.InvitedBy,
			CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
		})
	}
	sortInvites(out)
	return out, nil
}

func (f *Firestore) AcceptInvite(ctx context.Context, tokenHash, email string, now time.Time) (Member, error) {
	email = NormalizeEmail(email)
	invRef := f.c.Collection(colInvites).Doc(tokenHash)
	var mem Member
	err := f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(invRef)
		if isCode(err, codes.NotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var inv fsInvite
		if err := snap.DataTo(&inv); err != nil {
			return err
		}
		if inv.Email != email || !now.Before(inv.ExpiresAt) {
			return ErrNotFound
		}
		if _, err := tx.Get(f.member(email)); err == nil {
			return ErrExists
		} else if !isCode(err, codes.NotFound) {
			return err
		}
		mem = Member{Email: email, HouseholdID: inv.HouseholdID, Role: RoleMember, JoinedAt: now}
		if err := tx.Create(f.member(email), fsMember{Email: email, HouseholdID: mem.HouseholdID, Role: mem.Role, JoinedAt: now}); err != nil {
			return err
		}
		return tx.Delete(invRef)
	})
	if err != nil {
		return Member{}, err
	}
	return mem, nil
}

func (f *Firestore) DeleteInvite(ctx context.Context, householdID, tokenHash string) error {
	ref := f.c.Collection(colInvites).Doc(tokenHash)
	return f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if isCode(err, codes.NotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var inv fsInvite
		if err := snap.DataTo(&inv); err != nil {
			return err
		}
		if inv.HouseholdID != householdID {
			return nil // not this household's invite — never delete across households
		}
		return tx.Delete(ref)
	})
}

func (f *Firestore) PutShareToken(ctx context.Context, tokenHash, householdID string) error {
	_, err := f.c.Collection(colShareTokens).Doc(tokenHash).Set(ctx, fsIndex{HouseholdID: householdID})
	return err
}

func (f *Firestore) ShareTokenHousehold(ctx context.Context, tokenHash string) (string, error) {
	return f.index(ctx, colShareTokens, tokenHash)
}

func (f *Firestore) DeleteShareToken(ctx context.Context, tokenHash string) error {
	_, err := f.c.Collection(colShareTokens).Doc(tokenHash).Delete(ctx)
	return err
}

func (f *Firestore) PutSubscription(ctx context.Context, subscriptionID, householdID string) error {
	_, err := f.c.Collection(colSubscriptions).Doc(subscriptionID).Set(ctx, fsIndex{HouseholdID: householdID})
	return err
}

func (f *Firestore) SubscriptionHousehold(ctx context.Context, subscriptionID string) (string, error) {
	return f.index(ctx, colSubscriptions, subscriptionID)
}

func (f *Firestore) index(ctx context.Context, col, id string) (string, error) {
	snap, err := f.c.Collection(col).Doc(id).Get(ctx)
	if isCode(err, codes.NotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	var d fsIndex
	if err := snap.DataTo(&d); err != nil {
		return "", err
	}
	return d.HouseholdID, nil
}

// quotaTxAttempts is generous because one household's counter document is
// the contention point when several family members use AI at once; an
// aborted transaction is retried, never double-counted.
const quotaTxAttempts = 15

func (f *Firestore) usageRef(householdID, period string) *firestore.DocumentRef {
	return f.c.Collection(colUsage).Doc(householdID + "_" + period)
}

func (f *Firestore) ConsumeQuota(ctx context.Context, householdID, kind, period string, limit int) (int, error) {
	ref := f.usageRef(householdID, period)
	var count int
	err := f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		u := fsUsage{HouseholdID: householdID, Period: period, Counts: map[string]int{}}
		snap, err := tx.Get(ref)
		if err == nil {
			if err := snap.DataTo(&u); err != nil {
				return err
			}
			if u.Counts == nil {
				u.Counts = map[string]int{}
			}
		} else if !isCode(err, codes.NotFound) {
			return err
		}
		if u.Counts[kind] >= limit {
			count = u.Counts[kind]
			return ErrQuotaExceeded
		}
		u.Counts[kind]++
		count = u.Counts[kind]
		return tx.Set(ref, u)
	}, firestore.MaxAttempts(quotaTxAttempts))
	return count, err
}

func (f *Firestore) ReleaseQuota(ctx context.Context, householdID, kind, period string) error {
	ref := f.usageRef(householdID, period)
	return f.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if isCode(err, codes.NotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var u fsUsage
		if err := snap.DataTo(&u); err != nil {
			return err
		}
		if u.Counts[kind] <= 0 {
			return nil
		}
		u.Counts[kind]--
		return tx.Set(ref, u)
	}, firestore.MaxAttempts(quotaTxAttempts))
}

func (f *Firestore) Usage(ctx context.Context, householdID, period string) (map[string]int, error) {
	snap, err := f.usageRef(householdID, period).Get(ctx)
	if isCode(err, codes.NotFound) {
		return map[string]int{}, nil
	}
	if err != nil {
		return nil, err
	}
	var u fsUsage
	if err := snap.DataTo(&u); err != nil {
		return nil, err
	}
	if u.Counts == nil {
		u.Counts = map[string]int{}
	}
	return u.Counts, nil
}

func (f *Firestore) MarkEventProcessed(ctx context.Context, eventID string) (bool, error) {
	now := time.Now()
	_, err := f.c.Collection(colEvents).Doc(eventID).Create(ctx, map[string]any{
		"at": now, "expireAt": now.Add(eventRetention),
	})
	if isCode(err, codes.AlreadyExists) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

var (
	_ Store = (*Firestore)(nil)
	_ Store = (*Memory)(nil)
)
