package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kosh/internal/models"
	"kosh/internal/store"
	"kosh/internal/tenant"
	sh "kosh/sheets"
)

// Handler serves API data using typed repositories.
type Handler struct {
	tenants  tenant.Resolver
	devData  *models.Data
	demoData *models.Data             // lazily loaded; served to demo sessions in live mode
	isDemo   func(*http.Request) bool // nil when auth (and therefore demo sessions) is disabled
}

// NewHandler constructs a Handler. tenants maps each request to the
// household it reads/writes. devMode means there's no real spreadsheet:
// every request is served dev_data.json and mutations no-op.
func NewHandler(tenants tenant.Resolver, devMode bool, isDemo func(*http.Request) bool) *Handler {
	h := &Handler{tenants: tenants, isDemo: isDemo}
	if devMode {
		h.devData = loadDevData()
	}
	return h
}

// servingSampleData reports whether this request should be served sample data
// (and have mutations no-op) — either because the whole deployment runs in
// dev mode, or because this specific request carries a demo session.
func (h *Handler) servingSampleData(r *http.Request) bool {
	return h.devData != nil || (h.isDemo != nil && h.isDemo(r))
}

// IsSampleData exposes servingSampleData to other packages (e.g. ai.TaxHandler)
// that need to decide whether to persist generated data for this request.
func (h *Handler) IsSampleData(r *http.Request) bool {
	return h.servingSampleData(r)
}

// sampleData returns the sample dataset to serve for this request — the
// dev-mode dataset if the deployment itself runs in dev mode, otherwise the
// lazily-loaded demo dataset for an individual demo session.
func (h *Handler) sampleData() *models.Data {
	if h.devData != nil {
		return h.devData
	}
	if h.demoData == nil {
		h.demoData = loadDevData()
	}
	return h.demoData
}

func loadDevData() *models.Data {
	b, err := os.ReadFile("dev_data.json")
	if err != nil {
		fmt.Println("[dev] dev_data.json not found, returning empty data")
		return &models.Data{}
	}
	var d models.Data
	if err := json.Unmarshal(b, &d); err != nil {
		fmt.Println("[dev] failed to parse dev_data.json:", err)
		return &models.Data{}
	}
	fmt.Println("[dev] loaded dev_data.json")
	return &d
}

// LoadData returns the appropriate dataset for the request — used by the AI handler.
func (h *Handler) LoadData(r *http.Request) *models.Data {
	if h.servingSampleData(r) {
		return h.sampleData()
	}
	t, err := h.tenants.Resolve(r)
	if err != nil {
		log.Printf("⚠ resolving tenant: %v", err)
		return &models.Data{}
	}
	d, err := h.fetchFromRepos(t)
	if err != nil {
		log.Printf("⚠ loading data: %v", err)
		return &models.Data{}
	}
	models.ComputeFixedValues(d.Fixed)
	return d
}

// FamilyData returns the tenant's dataset regardless of who is asking — dev
// data in dev mode, live Sheets otherwise. Used by family-share views, whose
// requests carry no owner session (and must not be switched to demo data by
// a stray demo cookie in the recipient's browser).
func (h *Handler) FamilyData(t *tenant.Tenant) (*models.Data, error) {
	var d *models.Data
	if h.devData != nil {
		cp := *h.devData
		cp.Fixed = append([]models.Fixed(nil), cp.Fixed...)
		d = &cp
	} else {
		var err error
		if d, err = h.fetchFromRepos(t); err != nil {
			return nil, err
		}
	}
	models.ComputeFixedValues(d.Fixed)
	return d, nil
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	var d *models.Data
	sample := h.servingSampleData(r)
	if sample {
		// Shallow-copy so we don't mutate the cached struct across requests.
		src := h.sampleData()
		cp := *src
		fixed := make([]models.Fixed, len(cp.Fixed))
		copy(fixed, cp.Fixed)
		cp.Fixed = fixed
		d = &cp
	} else {
		t, ok := tenant.FromRequest(h.tenants, w, r)
		if !ok {
			return
		}
		d, err := h.fetchFromRepos(t)
		if err != nil {
			writeTenantError(w, err)
			return
		}
		models.ComputeFixedValues(d.Fixed)
		h.maybeSnapshotHistory(t, d)
		writeJSON(w, d)
		return
	}
	models.ComputeFixedValues(d.Fixed)
	writeJSON(w, d)
}

// fetchFromRepos reads all of t's sheet tabs and returns a fully-populated
// Data struct — in one batched API call when the backend supports it, else
// tab by tab. A missing/unshared tab is logged and left empty rather than
// failing the whole request — but losing access to the whole spreadsheet
// (a tenant.HTTPError, e.g. a revoked Google grant) is returned, so it's
// never mistaken for an empty one.
func (h *Handler) fetchFromRepos(t *tenant.Tenant) (*models.Data, error) {
	repos := t.Repos
	pre, err := prefetch(t, []string{
		repos.Members.Sheet(), repos.MF.Sheet(), repos.Stocks.Sheet(), repos.Metals.Sheet(),
		repos.Fixed.Sheet(), repos.Insurance.Sheet(), repos.Loans.Sheet(), repos.NPS.Sheet(),
		repos.SIPs.Sheet(), repos.Lumpsums.Sheet(), repos.History.Sheet(), repos.Income.Sheet(),
		repos.TaxRecommendations.Sheet(), repos.TaxRules.Sheet(), configSheet,
	})
	if err != nil {
		return nil, err
	}

	d := &models.Data{
		Members: load(repos.Members, pre),
		MF:      load(repos.MF, pre),
		Stocks:  load(repos.Stocks, pre),
		Metals:  load(repos.Metals, pre),
		// CurrentValue is recomputed at request time by ComputeFixedValues,
		// so the sheet's value being overwritten is harmless.
		Fixed:              load(repos.Fixed, pre),
		Insurance:          load(repos.Insurance, pre),
		Loans:              load(repos.Loans, pre),
		NPS:                load(repos.NPS, pre),
		SIPs:               load(repos.SIPs, pre),
		Lumpsums:           load(repos.Lumpsums, pre),
		History:            load(repos.History, pre),
		Income:             load(repos.Income, pre),
		TaxRecommendations: load(repos.TaxRecommendations, pre),
		TaxRules:           load(repos.TaxRules, pre),
	}

	// Config is a key-value sheet with no typed repository.
	rows, ok := pre[configSheet]
	if !ok {
		var err error
		if rows, err = t.API.ReadSheet(configSheet); err != nil {
			log.Printf("⚠  sheet %q unavailable: %v", configSheet, err)
		}
	}
	if len(rows) > 1 {
		cfg := map[string]string{}
		for _, row := range rows[1:] {
			if len(row) >= 2 {
				k := strings.ToLower(strings.TrimSpace(sh.ColStr(row, 0)))
				v := strings.TrimSpace(sh.ColStr(row, 1))
				if k != "" {
					cfg[k] = v
				}
			}
		}
		d.Config = parseConfig(cfg)
	}
	return d, nil
}

const configSheet = "Config"

// prefetch batch-reads names if t's backend supports it. A nil result (no
// batch support, or the batch failed — e.g. one tab is missing) makes load
// fall back to per-tab reads, which tolerate a single missing tab. A
// tenant.HTTPError means the whole spreadsheet is unreachable and is
// returned instead.
func prefetch(t *tenant.Tenant, names []string) (map[string][][]interface{}, error) {
	br, ok := t.API.(store.BatchReader)
	if !ok {
		return nil, nil
	}
	rows, err := br.ReadSheets(names)
	if err != nil {
		var he tenant.HTTPError
		if errors.As(err, &he) {
			return nil, err
		}
		log.Printf("⚠  batch read failed, falling back to per-tab reads: %v", err)
		return nil, nil
	}
	return rows, nil
}

// writeTenantError responds with a tenant.HTTPError's own status, else 500.
func writeTenantError(w http.ResponseWriter, err error) {
	var he tenant.HTTPError
	if errors.As(err, &he) {
		http.Error(w, he.Error(), he.HTTPStatus())
		return
	}
	log.Printf("⚠ loading data: %v", err)
	http.Error(w, "couldn't load your data", http.StatusInternalServerError)
}

func load[T any](repo *store.Repository[T], pre map[string][][]interface{}) []T {
	if rows, ok := pre[repo.Sheet()]; ok {
		return repo.Parse(rows)
	}
	rows, err := repo.All()
	if err != nil {
		log.Printf("⚠  sheet %q unavailable: %v", repo.Sheet(), err)
	}
	return rows
}

// maybeSnapshotHistory appends a History row for the current month if one
// isn't already there, so the net-worth sparkline stays current without
// manual monthly upkeep. No-op in dev mode or if net worth is still zero
// (e.g. sheets not yet populated).
func (h *Handler) maybeSnapshotHistory(t *tenant.Tenant, d *models.Data) {
	if h.devData != nil {
		return
	}
	month := time.Now().Format("Jan 06")
	for _, row := range d.History {
		if row.Month == month {
			return
		}
	}
	total := models.NetWorthTotal(d)
	if total == 0 {
		return
	}
	if _, err := t.Repos.History.Add(models.History{Month: month, Value: total}); err != nil {
		log.Printf("⚠ History snapshot failed: %v", err)
		return
	}
	d.History = append(d.History, models.History{Month: month, Value: total})
}

func parseConfig(m map[string]string) models.Config {
	pf := func(k string) float64 {
		v, _ := parseFloat64(m[k])
		return v
	}
	return models.Config{
		GrossIncome:          pf("gross_income"),
		Regime:               m["regime"],
		CapitalGainsThisYear: pf("capital_gains_this_year"),
		ShiftedToParents:     pf("shifted_to_parents"),
		SurchargeFrom:        pf("surcharge_from"),
		NextSurchargeAt:      pf("next_surcharge_at"),
		ParentsCapacityMom:   pf("parents_capacity_mom"),
		ParentsCapacityDad:   pf("parents_capacity_dad"),
		Deduction80C:         pf("deduction_80c"),
		Deduction80D:         pf("deduction_80d"),
		OtherDeductions:      pf("other_deductions"),
		SavedByFiling:        pf("saved_by_filing"),
	}
}

func parseFloat64(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
