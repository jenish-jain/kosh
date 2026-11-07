package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// stores returns every Store implementation to run the contract against:
// Memory always, Firestore when FIRESTORE_EMULATOR_HOST points at an
// emulator (docker run -p 8085:8085 gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators
// gcloud emulators firestore start --host-port=0.0.0.0:8085).
func stores(t *testing.T) map[string]Store {
	t.Helper()
	out := map[string]Store{"memory": NewMemory()}
	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		fs, err := NewFirestore(context.Background(), "kosh-test", "(default)")
		if err != nil {
			t.Fatalf("emulator: %v", err)
		}
		t.Cleanup(func() { fs.Close() })
		out["firestore"] = fs
	}
	return out
}

// uniq makes IDs unique per test run, since the emulator is shared state.
func uniq(s string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return s + "-" + hex.EncodeToString(b)
}

func forEachStore(t *testing.T, fn func(t *testing.T, s Store)) {
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) { fn(t, s) })
	}
}

var ctx = context.Background()

func newHousehold(t *testing.T, s Store, owner string) Household {
	t.Helper()
	h := Household{
		ID: uniq("hh"), Name: "The Jains", OwnerEmail: owner, SpreadsheetID: "sheet-1",
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Billing:   Billing{Status: StatusTrialing, TrialEndsAt: time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Millisecond)},
	}
	if err := s.CreateHousehold(ctx, h, Member{Email: owner, HouseholdID: h.ID, Role: RoleOwner, JoinedAt: h.CreatedAt}); err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	return h
}

func TestStore_CreateAndGet(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		owner := uniq("Owner") + "@Example.com"
		h := newHousehold(t, s, owner)

		got, err := s.GetHousehold(ctx, h.ID)
		if err != nil || got.Name != "The Jains" || got.Billing.Status != StatusTrialing || !got.Billing.TrialEndsAt.Equal(h.Billing.TrialEndsAt) {
			t.Fatalf("GetHousehold: %+v, %v", got, err)
		}
		m, err := s.GetMember(ctx, strings.ToUpper(owner)) // lookups are case-insensitive
		if err != nil || m.HouseholdID != h.ID || m.Role != RoleOwner || m.Email != NormalizeEmail(owner) {
			t.Fatalf("GetMember: %+v, %v", m, err)
		}
		if _, err := s.GetHousehold(ctx, uniq("missing")); !errors.Is(err, ErrNotFound) {
			t.Errorf("missing household: want ErrNotFound, got %v", err)
		}
	})
}

func TestStore_OwnerCanOnlyHaveOneHousehold(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		owner := uniq("o") + "@x.com"
		newHousehold(t, s, owner)
		h2 := Household{ID: uniq("hh")}
		err := s.CreateHousehold(ctx, h2, Member{Email: owner, HouseholdID: h2.ID, Role: RoleOwner})
		if !errors.Is(err, ErrExists) {
			t.Fatalf("want ErrExists, got %v", err)
		}
		if _, err := s.GetHousehold(ctx, h2.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("failed create must not leave a household behind: %v", err)
		}
	})
}

func TestStore_UpdateHousehold(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		h := newHousehold(t, s, uniq("o")+"@x.com")
		err := s.UpdateHousehold(ctx, h.ID, func(h *Household) error {
			h.Billing.Status = StatusActive
			h.Billing.SubscriptionID = "sub_1"
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		boom := errors.New("boom")
		if err := s.UpdateHousehold(ctx, h.ID, func(h *Household) error { h.Name = "changed"; return boom }); !errors.Is(err, boom) {
			t.Fatalf("fn error should be returned, got %v", err)
		}
		got, _ := s.GetHousehold(ctx, h.ID)
		if got.Billing.Status != StatusActive || got.Billing.SubscriptionID != "sub_1" || got.Name != "The Jains" {
			t.Fatalf("unexpected state: %+v", got)
		}
		if err := s.UpdateHousehold(ctx, uniq("missing"), func(*Household) error { return nil }); !errors.Is(err, ErrNotFound) {
			t.Errorf("missing: want ErrNotFound, got %v", err)
		}
	})
}

func TestStore_MemberInAtMostOneHousehold(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		a := newHousehold(t, s, uniq("a")+"@x.com")
		b := newHousehold(t, s, uniq("b")+"@x.com")
		spouse := uniq("spouse") + "@x.com"
		if err := s.AddMember(ctx, Member{Email: spouse, HouseholdID: a.ID, Role: RoleMember}); err != nil {
			t.Fatal(err)
		}
		if err := s.AddMember(ctx, Member{Email: spouse, HouseholdID: b.ID, Role: RoleMember}); !errors.Is(err, ErrExists) {
			t.Fatalf("want ErrExists, got %v", err)
		}
		ms, _ := s.ListMembers(ctx, a.ID)
		if len(ms) != 2 {
			t.Fatalf("want 2 members, got %+v", ms)
		}
		s.RemoveMember(ctx, spouse)
		if _, err := s.GetMember(ctx, spouse); !errors.Is(err, ErrNotFound) {
			t.Errorf("removed member still resolves: %v", err)
		}
	})
}

func TestStore_Invites(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		h := newHousehold(t, s, uniq("o")+"@x.com")
		now := time.Now()
		invitee := uniq("Invitee") + "@x.com"
		inv := Invite{TokenHash: uniq("inv"), HouseholdID: h.ID, Email: invitee, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
		if err := s.CreateInvite(ctx, inv); err != nil {
			t.Fatal(err)
		}
		if list, _ := s.ListInvites(ctx, h.ID); len(list) != 1 || list[0].Email != NormalizeEmail(invitee) {
			t.Fatalf("ListInvites: %+v", list)
		}

		if _, err := s.AcceptInvite(ctx, inv.TokenHash, uniq("someone-else")+"@x.com", now); !errors.Is(err, ErrNotFound) {
			t.Errorf("wrong email: want ErrNotFound, got %v", err)
		}
		if _, err := s.AcceptInvite(ctx, inv.TokenHash, invitee, now.Add(2*time.Hour)); !errors.Is(err, ErrNotFound) {
			t.Errorf("expired: want ErrNotFound, got %v", err)
		}
		m, err := s.AcceptInvite(ctx, inv.TokenHash, strings.ToUpper(invitee), now)
		if err != nil || m.HouseholdID != h.ID || m.Role != RoleMember {
			t.Fatalf("accept: %+v, %v", m, err)
		}
		if _, err := s.AcceptInvite(ctx, inv.TokenHash, invitee, now); !errors.Is(err, ErrNotFound) {
			t.Errorf("invite must be single-use, got %v", err)
		}
	})
}

func TestStore_InviteForExistingMemberIsKept(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		a := newHousehold(t, s, uniq("a")+"@x.com")
		bOwner := uniq("b") + "@x.com"
		newHousehold(t, s, bOwner)
		now := time.Now()
		inv := Invite{TokenHash: uniq("inv"), HouseholdID: a.ID, Email: bOwner, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
		s.CreateInvite(ctx, inv)
		if _, err := s.AcceptInvite(ctx, inv.TokenHash, bOwner, now); !errors.Is(err, ErrExists) {
			t.Fatalf("want ErrExists, got %v", err)
		}
		if list, _ := s.ListInvites(ctx, a.ID); len(list) != 1 {
			t.Errorf("invite should survive a failed accept, got %+v", list)
		}
	})
}

func TestStore_DeleteInviteIsScopedToHousehold(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		a := newHousehold(t, s, uniq("a")+"@x.com")
		b := newHousehold(t, s, uniq("b")+"@x.com")
		inv := Invite{TokenHash: uniq("inv"), HouseholdID: a.ID, Email: "x@x.com", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
		s.CreateInvite(ctx, inv)
		s.DeleteInvite(ctx, b.ID, inv.TokenHash)
		if list, _ := s.ListInvites(ctx, a.ID); len(list) != 1 {
			t.Fatal("another household deleted this household's invite")
		}
		s.DeleteInvite(ctx, a.ID, inv.TokenHash)
		if list, _ := s.ListInvites(ctx, a.ID); len(list) != 0 {
			t.Fatal("own-household delete failed")
		}
	})
}

func TestStore_Indexes(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		tok, sub := uniq("tok"), uniq("sub")
		s.PutShareToken(ctx, tok, "hh-1")
		s.PutSubscription(ctx, sub, "hh-2")
		if id, err := s.ShareTokenHousehold(ctx, tok); err != nil || id != "hh-1" {
			t.Fatalf("share index: %q %v", id, err)
		}
		if id, err := s.SubscriptionHousehold(ctx, sub); err != nil || id != "hh-2" {
			t.Fatalf("subscription index: %q %v", id, err)
		}
		s.DeleteShareToken(ctx, tok)
		if _, err := s.ShareTokenHousehold(ctx, tok); !errors.Is(err, ErrNotFound) {
			t.Errorf("deleted share token still resolves: %v", err)
		}
		if _, err := s.SubscriptionHousehold(ctx, uniq("nope")); !errors.Is(err, ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})
}

func TestStore_ConsumeQuotaIsAtomic(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		hh := uniq("hh")
		const limit, attempts = 3, 6
		var wg sync.WaitGroup
		var mu sync.Mutex
		ok, exceeded := 0, 0
		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.ConsumeQuota(ctx, hh, "ask", "2026-11", limit)
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					ok++
				case errors.Is(err, ErrQuotaExceeded):
					exceeded++
				default:
					t.Errorf("unexpected error: %v", err)
				}
			}()
		}
		wg.Wait()
		if ok != limit || exceeded != attempts-limit {
			t.Fatalf("want %d ok / %d exceeded, got %d / %d", limit, attempts-limit, ok, exceeded)
		}
		u, _ := s.Usage(ctx, hh, "2026-11")
		if u["ask"] != limit {
			t.Errorf("usage: %+v", u)
		}
		if u2, _ := s.Usage(ctx, hh, "2026-12"); len(u2) != 0 {
			t.Errorf("new period should start empty: %+v", u2)
		}
	})
}

func TestStore_MarkEventProcessed(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		id := uniq("evt")
		if first, err := s.MarkEventProcessed(ctx, id); !first || err != nil {
			t.Fatalf("first: %v %v", first, err)
		}
		if first, err := s.MarkEventProcessed(ctx, id); first || err != nil {
			t.Fatalf("second: %v %v", first, err)
		}
	})
}

func TestStore_DeleteHouseholdRemovesOnlyItsData(t *testing.T) {
	forEachStore(t, func(t *testing.T, s Store) {
		aOwner, bOwner := uniq("a")+"@x.com", uniq("b")+"@x.com"
		a := newHousehold(t, s, aOwner)
		b := newHousehold(t, s, bOwner)
		aMember := uniq("am") + "@x.com"
		s.AddMember(ctx, Member{Email: aMember, HouseholdID: a.ID, Role: RoleMember})
		aTok, bTok, aSub := uniq("tok"), uniq("tok"), uniq("sub")
		s.PutShareToken(ctx, aTok, a.ID)
		s.PutShareToken(ctx, bTok, b.ID)
		s.PutSubscription(ctx, aSub, a.ID)
		s.CreateInvite(ctx, Invite{TokenHash: uniq("inv"), HouseholdID: a.ID, Email: "i@x.com", ExpiresAt: time.Now().Add(time.Hour)})
		s.ConsumeQuota(ctx, a.ID, "ask", "2026-11", 10)

		if err := s.DeleteHousehold(ctx, a.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetHousehold(ctx, a.ID); !errors.Is(err, ErrNotFound) {
			t.Error("household not deleted")
		}
		for _, e := range []string{aOwner, aMember} {
			if _, err := s.GetMember(ctx, e); !errors.Is(err, ErrNotFound) {
				t.Errorf("member %s survived", e)
			}
		}
		if _, err := s.ShareTokenHousehold(ctx, aTok); !errors.Is(err, ErrNotFound) {
			t.Error("share token survived")
		}
		if _, err := s.SubscriptionHousehold(ctx, aSub); !errors.Is(err, ErrNotFound) {
			t.Error("subscription index survived")
		}
		if list, _ := s.ListInvites(ctx, a.ID); len(list) != 0 {
			t.Error("invite survived")
		}
		if u, _ := s.Usage(ctx, a.ID, "2026-11"); len(u) != 0 {
			t.Error("usage survived")
		}
		// The other household is untouched.
		if _, err := s.GetHousehold(ctx, b.ID); err != nil {
			t.Error("other household deleted")
		}
		if _, err := s.GetMember(ctx, bOwner); err != nil {
			t.Error("other household's owner deleted")
		}
		if id, _ := s.ShareTokenHousehold(ctx, bTok); id != b.ID {
			t.Error("other household's share token deleted")
		}
	})
}

func TestTokenSealer(t *testing.T) {
	s, err := NewTokenSealer(strings.Repeat("k", MinKeyLength))
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := s.Seal("1//refresh-token", "hh-a")
	if strings.Contains(enc, "refresh-token") {
		t.Fatal("ciphertext leaks plaintext")
	}
	if got, err := s.Open(enc, "hh-a"); err != nil || got != "1//refresh-token" {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if _, err := s.Open(enc, "hh-b"); err == nil {
		t.Error("ciphertext must not open under another household's ID")
	}
	other, _ := NewTokenSealer(strings.Repeat("x", MinKeyLength))
	if _, err := other.Open(enc, "hh-a"); err == nil {
		t.Error("ciphertext must not open under another key")
	}
	if _, err := NewTokenSealer("short"); err == nil {
		t.Error("short key accepted")
	}
}
