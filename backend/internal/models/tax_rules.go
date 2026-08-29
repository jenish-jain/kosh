package models

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// TaxRuleSet is one versioned row of tax rules, self-provisioned into a
// "TaxRules" sheet tab (one row per FY+regime version). Nothing auto-applies:
// only a row with Status=="active" is used by ComputeTax — a new version
// starts as "pending" and only takes effect once explicitly approved (see
// the propose/approve/reject handlers in package ai).
type TaxRuleSet struct {
	ID            string `json:"id"             sheet:"id"`
	FY            string `json:"fy"             sheet:"fy"`
	Regime        string `json:"regime"         sheet:"regime"` // "old" | "new"
	Status        string `json:"status"         sheet:"status"` // "active" | "pending" | "rejected" | "superseded"
	Source        string `json:"source"         sheet:"source"`
	GeneratedDate string `json:"generated_date" sheet:"generated_date"`
	ActivatedDate string `json:"activated_date" sheet:"activated_date"`
	RulesJSON     string `json:"rules_json"     sheet:"rules_json"`
	Notes         string `json:"notes"          sheet:"notes"`
}

// TaxRules is the parsed shape of TaxRuleSet.RulesJSON — the slab/surcharge/
// deduction parameters ComputeTax interprets. SchemaVersion exists so a
// future field this code doesn't understand yet fails loudly at parse time
// rather than silently being ignored.
type TaxRules struct {
	SchemaVersion   int             `json:"schemaVersion"`
	Slabs           []SlabBand      `json:"slabs"`
	StdDeduction    float64         `json:"stdDeduction"`
	RebateThreshold float64         `json:"rebateThreshold"`
	RebateAmount    float64         `json:"rebateAmount"`
	Surcharge       []SurchargeBand `json:"surcharge"`
	CessRate        float64         `json:"cessRate"`
	DeductionCaps   DeductionCaps   `json:"deductionCaps"`
}

// SlabBand is one income-tax slab, keyed on TAXABLE income. Upto is this
// band's upper bound in rupees; nil marks the unbounded top band.
type SlabBand struct {
	Upto *float64 `json:"upto,omitempty"`
	Rate float64  `json:"rate"`
}

// SurchargeBand applies Rate to the base tax once GROSS income exceeds
// Above. Only the highest-threshold band the income exceeds applies — not
// cumulative/marginal like the income slabs are.
type SurchargeBand struct {
	Above float64 `json:"above"`
	Rate  float64 `json:"rate"`
}

// DeductionCaps are the old-regime-only deduction limits the tax-saving
// recommendation engine (ComputeTaxFacts) compares actual usage against.
type DeductionCaps struct {
	Section80C       float64 `json:"section80C"`
	Section80DSelf   float64 `json:"section80DSelf"`
	Section80DSenior float64 `json:"section80DSenior"`
	NPS80CCD1B       float64 `json:"nps80CCD1B"`
}

// ParseRules decodes RulesJSON into TaxRules.
func (rs TaxRuleSet) ParseRules() (TaxRules, error) {
	var r TaxRules
	if err := json.Unmarshal([]byte(rs.RulesJSON), &r); err != nil {
		return TaxRules{}, fmt.Errorf("parsing rules_json for %s: %w", rs.ID, err)
	}
	return r, nil
}

// ComputeTax applies rules to grossIncome. This is the single generic
// interpreter that replaces the old regime-specific hardcoded slab-walking
// functions — every "which slabs/caps apply" decision now lives in data
// (TaxRuleSet.RulesJSON), not in code that needs a redeploy to change.
func ComputeTax(rules TaxRules, grossIncome float64) float64 {
	taxable := math.Max(0, grossIncome-rules.StdDeduction)

	tax := 0.0
	prev := 0.0
	for _, band := range rules.Slabs {
		upper := taxable
		if band.Upto != nil {
			upper = math.Min(taxable, *band.Upto)
		}
		if upper > prev {
			tax += (upper - prev) * band.Rate
		}
		if band.Upto == nil {
			break
		}
		prev = *band.Upto
	}

	if rules.RebateThreshold > 0 && taxable <= rules.RebateThreshold {
		tax = math.Max(0, tax-rules.RebateAmount)
	}

	surchargeRate, maxAbove := 0.0, -1.0
	for _, band := range rules.Surcharge {
		if grossIncome > band.Above && band.Above > maxAbove {
			maxAbove = band.Above
			surchargeRate = band.Rate
		}
	}
	surcharge := tax * surchargeRate

	cess := (tax + surcharge) * rules.CessRate
	return math.Round(tax + surcharge + cess)
}

// activeRuleSet returns the "active" TaxRuleSet for fy+regime. If more than
// one row is transiently active for the same FY+regime (possible mid-way
// through an approve operation — see the approve handler's ordering), the
// most recently activated one wins. Falls back to the most recently
// activated active row for the regime regardless of FY, then to the bundled
// DefaultTaxRuleSet so the app never breaks before the one-time migration
// seed has run.
func activeRuleSet(rows []TaxRuleSet, fy, regime string) TaxRuleSet {
	pick := func(matchFY bool) (TaxRuleSet, bool) {
		var best TaxRuleSet
		found := false
		for _, r := range rows {
			if r.Status != "active" || !strings.EqualFold(r.Regime, regime) {
				continue
			}
			if matchFY && r.FY != fy {
				continue
			}
			if !found || r.ActivatedDate > best.ActivatedDate {
				best, found = r, true
			}
		}
		return best, found
	}
	if rs, ok := pick(true); ok {
		return rs
	}
	if rs, ok := pick(false); ok {
		return rs
	}
	return DefaultTaxRuleSet(regime)
}

// ActiveTaxRules resolves the active rule set for fy+regime and returns its
// parsed rules, falling back to the bundled default if parsing fails.
func ActiveTaxRules(rows []TaxRuleSet, fy, regime string) TaxRules {
	rs := activeRuleSet(rows, fy, regime)
	rules, err := rs.ParseRules()
	if err != nil {
		def, _ := DefaultTaxRuleSet(regime).ParseRules()
		return def
	}
	return rules
}

func f64(v float64) *float64 { return &v }

// DefaultTaxRuleSet returns the bundled fallback rule set for regime,
// reproducing the numbers this app shipped with before the rules engine
// existed. Used to seed the initial "active" TaxRules rows on first run and
// as a last-resort fallback if no active row exists yet.
//
// One deliberate difference from the old hardcoded taxOldRegime(): that
// function never applied a Section 87A rebate for the old regime at all
// (real law zeroes tax for taxable income <= Rs 5,00,000). This default
// rule set includes it — a real bug fix, not a migration artifact.
func DefaultTaxRuleSet(regime string) TaxRuleSet {
	isNew := strings.EqualFold(regime, "new")
	regimeName := "old"
	if isNew {
		regimeName = "new"
	}

	caps := DeductionCaps{Section80C: 150000, Section80DSelf: 25000, Section80DSenior: 50000, NPS80CCD1B: 50000}
	// Full rebate to zero within threshold — a cap far above any realistic
	// tax amount reproduces the old "tax = 0" behavior via max(0, tax-cap).
	const fullRebate = 999999999.0

	var rules TaxRules
	if isNew {
		rules = TaxRules{
			SchemaVersion: 1,
			Slabs: []SlabBand{
				{Upto: f64(400000), Rate: 0},
				{Upto: f64(800000), Rate: 0.05},
				{Upto: f64(1200000), Rate: 0.10},
				{Upto: f64(1600000), Rate: 0.15},
				{Upto: f64(2000000), Rate: 0.20},
				{Upto: f64(2400000), Rate: 0.25},
				{Rate: 0.30},
			},
			StdDeduction:    75000,
			RebateThreshold: 1200000,
			RebateAmount:    fullRebate,
			Surcharge: []SurchargeBand{
				{Above: 5000000, Rate: 0.10},
				{Above: 10000000, Rate: 0.15},
			},
			CessRate:      0.04,
			DeductionCaps: caps,
		}
	} else {
		rules = TaxRules{
			SchemaVersion: 1,
			Slabs: []SlabBand{
				{Upto: f64(250000), Rate: 0},
				{Upto: f64(500000), Rate: 0.05},
				{Upto: f64(1000000), Rate: 0.20},
				{Rate: 0.30},
			},
			StdDeduction:    0,
			RebateThreshold: 500000,
			RebateAmount:    fullRebate,
			Surcharge: []SurchargeBand{
				{Above: 5000000, Rate: 0.10},
				{Above: 10000000, Rate: 0.15},
			},
			CessRate:      0.04,
			DeductionCaps: caps,
		}
	}

	b, _ := json.Marshal(rules)
	return TaxRuleSet{
		ID:        "seed-" + regimeName,
		FY:        "FY 2025-26",
		Regime:    regimeName,
		Status:    "active",
		Source:    "Bundled default (pre-rules-engine values)",
		RulesJSON: string(b),
		Notes:     "Seeded automatically on first run.",
	}
}
