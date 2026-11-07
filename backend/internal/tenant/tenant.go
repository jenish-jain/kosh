// Package tenant decides whose household data a request reads and writes.
//
// Self-hosted Kosh has exactly one household, configured by env vars, so
// Single resolves every request to it. The hosted service (see
// HOSTED_PLAN.md) will resolve each request to the signed-in customer's own
// Google Sheet instead — handlers only ever see a *Tenant, so they don't
// change when that lands.
package tenant

import (
	"errors"
	"log"
	"net/http"

	"kosh/internal/store"
)

// Tenant is one household's data source.
type Tenant struct {
	ID string
	// API reads and writes the household's spreadsheet. Repos is built on
	// it; it's exposed directly for the key-value Config tab, which has no
	// typed repository.
	API   store.SheetsAPI
	Repos *store.Repositories
}

// New returns a Tenant whose repositories are backed by api.
func New(id string, api store.SheetsAPI) *Tenant {
	return &Tenant{ID: id, API: api, Repos: store.NewRepositories(api)}
}

// Resolver maps a request to the tenant it acts on.
type Resolver interface {
	Resolve(r *http.Request) (*Tenant, error)
}

type single struct{ t *Tenant }

// Single returns a Resolver that maps every request to t — self-host mode.
func Single(t *Tenant) Resolver { return single{t} }

func (s single) Resolve(*http.Request) (*Tenant, error) { return s.t, nil }

// HTTPError lets a Resolver choose the response for an expected failure
// (e.g. "this account hasn't set up a household yet") instead of the
// generic 401.
type HTTPError interface {
	error
	HTTPStatus() int
}

// FromRequest resolves r's tenant, writing an error response and returning
// false if that fails, so handlers can bail out with a single check.
func FromRequest(res Resolver, w http.ResponseWriter, r *http.Request) (*Tenant, bool) {
	t, err := res.Resolve(r)
	if err != nil {
		var he HTTPError
		if errors.As(err, &he) {
			http.Error(w, he.Error(), he.HTTPStatus())
			return nil, false
		}
		log.Printf("⚠ resolving tenant: %v", err)
		http.Error(w, "couldn't load your household", http.StatusUnauthorized)
		return nil, false
	}
	return t, true
}
