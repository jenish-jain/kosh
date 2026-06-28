package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kosh/internal/models"
	"kosh/internal/store"
	sh "kosh/sheets"
)

// Handler serves API data using typed repositories.
type Handler struct {
	repos     *store.Repositories
	configAPI store.SheetsAPI // for reading the Config key-value sheet
	devData   *models.Data
	demoData  *models.Data             // lazily loaded; served to demo sessions in live mode
	isDemo    func(*http.Request) bool // nil when auth (and therefore demo sessions) is disabled
}

// NewHandler constructs a Handler. When repos is backed by a real sheets client,
// configAPI should be the same client (passed separately so Config reading can
// use the SheetsAPI abstraction without coupling to *sh.Client). Pass nil for
// configAPI to skip Config reads (dev/demo mode never reaches that path anyway).
func NewHandler(repos *store.Repositories, configAPI store.SheetsAPI, isDemo func(*http.Request) bool) *Handler {
	h := &Handler{repos: repos, configAPI: configAPI, isDemo: isDemo}
	if configAPI == nil {
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
	d := h.fetchFromRepos()
	models.ComputeFixedValues(d.Fixed)
	return d
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
		d = h.fetchFromRepos()
	}
	models.ComputeFixedValues(d.Fixed)
	if !sample {
		h.maybeSnapshotHistory(d)
	}
	writeJSON(w, d)
}

// fetchFromRepos reads all sheet tabs via the typed repositories and returns a
// fully-populated Data struct. Sheet read errors are logged as warnings (a
// missing/unshared tab should not crash the whole app).
func (h *Handler) fetchFromRepos() *models.Data {
	d := &models.Data{}

	tabLog := func(tab string, e error) {
		log.Printf("⚠  sheet %q unavailable: %v", tab, e)
	}

	if rows, e := h.repos.Members.All(); e == nil {
		d.Members = rows
	} else {
		tabLog("Members", e)
	}
	if rows, e := h.repos.MF.All(); e == nil {
		d.MF = rows
	} else {
		tabLog("MF", e)
	}
	if rows, e := h.repos.Stocks.All(); e == nil {
		d.Stocks = rows
	} else {
		tabLog("Stocks", e)
	}
	if rows, e := h.repos.Metals.All(); e == nil {
		d.Metals = rows
	} else {
		tabLog("Metals", e)
	}
	if rows, e := h.repos.Fixed.All(); e == nil {
		// CurrentValue (col index 6) is computed at request time by
		// ComputeFixedValues — the repository reads it from the sheet but it will
		// be overwritten, so this is harmless.
		d.Fixed = rows
	} else {
		tabLog("Fixed", e)
	}
	if rows, e := h.repos.Insurance.All(); e == nil {
		d.Insurance = rows
	} else {
		tabLog("Insurance", e)
	}
	if rows, e := h.repos.Loans.All(); e == nil {
		d.Loans = rows
	} else {
		tabLog("Loans", e)
	}
	if rows, e := h.repos.NPS.All(); e == nil {
		d.NPS = rows
	} else {
		tabLog("NPS", e)
	}
	if rows, e := h.repos.SIPs.All(); e == nil {
		d.SIPs = rows
	} else {
		tabLog("SIPs", e)
	}
	if rows, e := h.repos.Lumpsums.All(); e == nil {
		d.Lumpsums = rows
	} else {
		tabLog("Lumpsums", e)
	}
	if rows, e := h.repos.History.All(); e == nil {
		d.History = rows
	} else {
		tabLog("History", e)
	}
	if rows, e := h.repos.Income.All(); e == nil {
		d.Income = rows
	} else {
		tabLog("Income", e)
	}

	// Config is a key-value sheet with no typed repository — read it directly.
	if h.configAPI != nil {
		if rows, e := h.configAPI.ReadSheet("Config"); e == nil {
			cfg := map[string]string{}
			for _, row := range rows[1:] {
				if len(row) >= 2 {
					cfg[strings.TrimSpace(sh.ColStr(row, 0))] = sh.ColStr(row, 1)
				}
			}
			d.Config = parseConfig(cfg)
		} else {
			tabLog("Config", e)
		}
	}

	return d
}

// maybeSnapshotHistory appends a History row for the current month if one
// isn't already there, so the net-worth sparkline stays current without
// manual monthly upkeep. No-op in dev mode or if net worth is still zero
// (e.g. sheets not yet populated).
func (h *Handler) maybeSnapshotHistory(d *models.Data) {
	if h.configAPI == nil {
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
	if _, err := h.repos.History.Add(models.History{Month: month, Value: total}); err != nil {
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
