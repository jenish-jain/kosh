package controlplane

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Memory is an in-process Store for tests and local development. State is
// lost on restart.
type Memory struct {
	mu            sync.Mutex
	households    map[string]Household
	members       map[string]Member                    // by normalized email
	invites       map[string]Invite                    // by token hash
	shareTokens   map[string]string                    // token hash -> household
	subscriptions map[string]string                    // subscription ID -> household
	usage         map[string]map[string]map[string]int // household -> period -> kind -> count
	events        map[string]bool
	policy        *FeaturePolicy
}

// NewMemory returns an empty in-memory Store.
func NewMemory() *Memory {
	return &Memory{
		households:    map[string]Household{},
		members:       map[string]Member{},
		invites:       map[string]Invite{},
		shareTokens:   map[string]string{},
		subscriptions: map[string]string{},
		usage:         map[string]map[string]map[string]int{},
		events:        map[string]bool{},
	}
}

func (m *Memory) CreateHousehold(_ context.Context, h Household, owner Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	owner.Email = NormalizeEmail(owner.Email)
	if _, ok := m.members[owner.Email]; ok {
		return ErrExists
	}
	if _, ok := m.households[h.ID]; ok {
		return ErrExists
	}
	m.households[h.ID] = h
	m.members[owner.Email] = owner
	return nil
}

func (m *Memory) GetHousehold(_ context.Context, id string) (Household, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.households[id]
	if !ok {
		return Household{}, ErrNotFound
	}
	return h, nil
}

func (m *Memory) UpdateHousehold(_ context.Context, id string, fn func(*Household) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.households[id]
	if !ok {
		return ErrNotFound
	}
	if err := fn(&h); err != nil {
		return err
	}
	h.ID = id
	m.households[id] = h
	return nil
}

func (m *Memory) DeleteHousehold(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.households, id)
	for k, v := range m.members {
		if v.HouseholdID == id {
			delete(m.members, k)
		}
	}
	for k, v := range m.invites {
		if v.HouseholdID == id {
			delete(m.invites, k)
		}
	}
	for k, v := range m.shareTokens {
		if v == id {
			delete(m.shareTokens, k)
		}
	}
	for k, v := range m.subscriptions {
		if v == id {
			delete(m.subscriptions, k)
		}
	}
	delete(m.usage, id)
	return nil
}

func (m *Memory) GetMember(_ context.Context, email string) (Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mem, ok := m.members[NormalizeEmail(email)]
	if !ok {
		return Member{}, ErrNotFound
	}
	return mem, nil
}

func (m *Memory) ListMembers(_ context.Context, householdID string) ([]Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Member
	for _, v := range m.members {
		if v.HouseholdID == householdID {
			out = append(out, v)
		}
	}
	sortMembers(out)
	return out, nil
}

func (m *Memory) AddMember(_ context.Context, mem Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addMemberLocked(mem)
}

func (m *Memory) addMemberLocked(mem Member) error {
	mem.Email = NormalizeEmail(mem.Email)
	if _, ok := m.members[mem.Email]; ok {
		return ErrExists
	}
	m.members[mem.Email] = mem
	return nil
}

func (m *Memory) RemoveMember(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.members, NormalizeEmail(email))
	return nil
}

func (m *Memory) CreateInvite(_ context.Context, inv Invite) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv.Email = NormalizeEmail(inv.Email)
	if _, ok := m.invites[inv.TokenHash]; ok {
		return ErrExists
	}
	m.invites[inv.TokenHash] = inv
	return nil
}

func (m *Memory) ListInvites(_ context.Context, householdID string) ([]Invite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Invite
	for _, v := range m.invites {
		if v.HouseholdID == householdID {
			out = append(out, v)
		}
	}
	sortInvites(out)
	return out, nil
}

func (m *Memory) AcceptInvite(_ context.Context, tokenHash, email string, now time.Time) (Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	email = NormalizeEmail(email)
	inv, ok := m.invites[tokenHash]
	if !ok || inv.Email != email || !now.Before(inv.ExpiresAt) {
		return Member{}, ErrNotFound
	}
	mem := Member{Email: email, HouseholdID: inv.HouseholdID, Role: RoleMember, JoinedAt: now}
	if err := m.addMemberLocked(mem); err != nil {
		return Member{}, err
	}
	delete(m.invites, tokenHash)
	return mem, nil
}

func (m *Memory) DeleteInvite(_ context.Context, householdID, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inv, ok := m.invites[tokenHash]; ok && inv.HouseholdID == householdID {
		delete(m.invites, tokenHash)
	}
	return nil
}

func (m *Memory) PutShareToken(_ context.Context, tokenHash, householdID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shareTokens[tokenHash] = householdID
	return nil
}

func (m *Memory) ShareTokenHousehold(_ context.Context, tokenHash string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.shareTokens[tokenHash]
	if !ok {
		return "", ErrNotFound
	}
	return id, nil
}

func (m *Memory) DeleteShareToken(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.shareTokens, tokenHash)
	return nil
}

func (m *Memory) PutSubscription(_ context.Context, subscriptionID, householdID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[subscriptionID] = householdID
	return nil
}

func (m *Memory) SubscriptionHousehold(_ context.Context, subscriptionID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.subscriptions[subscriptionID]
	if !ok {
		return "", ErrNotFound
	}
	return id, nil
}

func (m *Memory) ConsumeQuota(_ context.Context, householdID, kind, period string, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.usage[householdID] == nil {
		m.usage[householdID] = map[string]map[string]int{}
	}
	counts := m.usage[householdID][period]
	if counts == nil {
		counts = map[string]int{}
		m.usage[householdID][period] = counts
	}
	if counts[kind] >= limit {
		return counts[kind], ErrQuotaExceeded
	}
	counts[kind]++
	return counts[kind], nil
}

func (m *Memory) ReleaseQuota(_ context.Context, householdID, kind, period string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.usage[householdID][period]; c != nil && c[kind] > 0 {
		c[kind]--
	}
	return nil
}

func (m *Memory) Usage(_ context.Context, householdID, period string) (map[string]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]int{}
	for k, v := range m.usage[householdID][period] {
		out[k] = v
	}
	return out, nil
}

func (m *Memory) MarkEventProcessed(_ context.Context, eventID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.events[eventID] {
		return false, nil
	}
	m.events[eventID] = true
	return true, nil
}

func (m *Memory) GetFeaturePolicy(_ context.Context) (FeaturePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.policy == nil {
		return FeaturePolicy{}, ErrNotFound
	}
	return copyPolicy(*m.policy), nil
}

func (m *Memory) PutFeaturePolicy(_ context.Context, p FeaturePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := copyPolicy(p)
	m.policy = &c
	return nil
}

// copyPolicy deep-copies p so callers can't mutate stored state.
func copyPolicy(p FeaturePolicy) FeaturePolicy {
	tiers := make(map[string]map[string]bool, len(p.Tiers))
	for tier, fs := range p.Tiers {
		tiers[tier] = make(map[string]bool, len(fs))
		for f, on := range fs {
			tiers[tier][f] = on
		}
	}
	p.Tiers = tiers
	return p
}

func sortMembers(ms []Member) {
	sort.Slice(ms, func(i, j int) bool { return ms[i].JoinedAt.Before(ms[j].JoinedAt) })
}

func sortInvites(is []Invite) {
	sort.Slice(is, func(i, j int) bool { return is[i].CreatedAt.Before(is[j].CreatedAt) })
}
