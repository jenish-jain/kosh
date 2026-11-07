package hosted

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"kosh/internal/controlplane"
	"kosh/internal/share"
	"kosh/internal/tenant"
)

// statusError is an expected resolver failure with its HTTP response; the
// message is a stable code the frontend switches on.
type statusError struct {
	msg    string
	status int
}

func (e *statusError) Error() string   { return e.msg }
func (e *statusError) HTTPStatus() int { return e.status }

var (
	// ErrNoHousehold means the signed-in account belongs to no household —
	// a new sign-up that hasn't onboarded yet.
	ErrNoHousehold error = &statusError{"no_household", http.StatusForbidden}
	// ErrReconnect means Google rejected the household's stored grant and
	// the owner must reconnect Drive.
	ErrReconnect error = &statusError{"reconnect_required", http.StatusConflict}
	// errUnknownShare is a share link no household owns. Same 401 and text as
	// the share handler's own rejections, so it reveals nothing more.
	errUnknownShare error = &statusError{share.ErrGeneric, http.StatusUnauthorized}
)

// tenantTTL bounds how long a built tenant (its Sheets client and cached
// access token) is reused before being rebuilt from the control plane.
const tenantTTL = 10 * time.Minute

// maxCachedTenants caps memory use on a busy instance.
const maxCachedTenants = 2000

// Resolver maps a request to its household's Sheet. App routes resolve by
// the signed-in account's membership; family-share recipient routes (which
// carry no session) by the share-link token.
type Resolver struct {
	store        controlplane.Store
	sealer       *controlplane.TokenSealer
	google       Google
	sessionEmail func(*http.Request) (string, bool)
	now          func() time.Time

	mu    sync.Mutex
	cache map[string]cachedTenant // by household ID
}

type cachedTenant struct {
	t       *tenant.Tenant
	key     string // spreadsheet + sealed token; a change invalidates
	expires time.Time
}

// NewResolver constructs a Resolver.
func NewResolver(store controlplane.Store, sealer *controlplane.TokenSealer, google Google, sessionEmail func(*http.Request) (string, bool)) *Resolver {
	return &Resolver{store: store, sealer: sealer, google: google, sessionEmail: sessionEmail, now: time.Now, cache: map[string]cachedTenant{}}
}

// Resolve implements tenant.Resolver.
func (res *Resolver) Resolve(r *http.Request) (*tenant.Tenant, error) {
	ctx := r.Context()
	hid, err := res.householdID(ctx, r)
	if err != nil {
		return nil, err
	}
	h, err := res.store.GetHousehold(ctx, hid)
	if errors.Is(err, controlplane.ErrNotFound) {
		if strings.HasPrefix(r.URL.Path, "/api/share/") {
			return nil, errUnknownShare
		}
		return nil, ErrNoHousehold
	}
	if err != nil {
		return nil, err
	}
	return res.tenantFor(ctx, h)
}

func (res *Resolver) householdID(ctx context.Context, r *http.Request) (string, error) {
	if strings.HasPrefix(r.URL.Path, "/api/share/") {
		tok := r.Header.Get(share.TokenHeader)
		if tok == "" {
			return "", errUnknownShare
		}
		id, err := res.store.ShareTokenHousehold(ctx, share.HashToken(tok))
		if errors.Is(err, controlplane.ErrNotFound) {
			return "", errUnknownShare
		}
		return id, err
	}
	email, ok := res.sessionEmail(r)
	if !ok {
		return "", ErrNoHousehold
	}
	m, err := res.store.GetMember(ctx, email)
	if errors.Is(err, controlplane.ErrNotFound) {
		return "", ErrNoHousehold
	}
	if err != nil {
		return "", err
	}
	return m.HouseholdID, nil
}

func (res *Resolver) tenantFor(ctx context.Context, h controlplane.Household) (*tenant.Tenant, error) {
	if h.NeedsReconnect {
		return nil, ErrReconnect
	}
	key := h.SpreadsheetID + "|" + h.RefreshTokenEnc
	now := res.now()

	res.mu.Lock()
	if c, ok := res.cache[h.ID]; ok && c.key == key && now.Before(c.expires) {
		res.mu.Unlock()
		return c.t, nil
	}
	res.mu.Unlock()

	rt, err := res.sealer.Open(h.RefreshTokenEnc, h.ID)
	if err != nil {
		return nil, fmt.Errorf("unsealing household token: %w", err)
	}
	ts := oauth2.ReuseTokenSource(nil, res.google.TokenSource(rt))
	api, err := res.google.Sheets(ctx, ts, h.SpreadsheetID)
	if err != nil {
		return nil, err
	}
	t := tenant.New(h.ID, &guardedSheets{inner: api, onRevoked: func() { res.markReconnect(h.ID) }})

	res.mu.Lock()
	defer res.mu.Unlock()
	if len(res.cache) >= maxCachedTenants {
		res.evictLocked(now)
	}
	res.cache[h.ID] = cachedTenant{t: t, key: key, expires: now.Add(tenantTTL)}
	return t, nil
}

func (res *Resolver) evictLocked(now time.Time) {
	for id, c := range res.cache {
		if !now.Before(c.expires) {
			delete(res.cache, id)
		}
	}
	if len(res.cache) >= maxCachedTenants {
		res.cache = map[string]cachedTenant{}
	}
}

// Invalidate drops the cached tenant for a household (after reconnecting,
// or deleting it).
func (res *Resolver) Invalidate(householdID string) {
	res.mu.Lock()
	defer res.mu.Unlock()
	delete(res.cache, householdID)
}

func (res *Resolver) markReconnect(householdID string) {
	log.Printf("⚠ household %s: Google rejected the stored grant; marking for reconnect", householdID)
	err := res.store.UpdateHousehold(context.Background(), householdID, func(h *controlplane.Household) error {
		h.NeedsReconnect = true
		return nil
	})
	if err != nil {
		log.Printf("⚠ household %s: recording reconnect: %v", householdID, err)
	}
	res.Invalidate(householdID)
}
