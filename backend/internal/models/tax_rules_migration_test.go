package models_test

import (
	"math"
	"testing"

	"kosh/internal/models"
)

// Reference implementations of the pre-rules-engine hardcoded tax functions
// that used to live in tax_facts.go, before the generic models.ComputeTax
// engine replaced them. Kept here ONLY to regression-test that the
// migration to data-driven rules preserves behavior — except for one
// documented, intentional fix (see TestComputeTax_MatchesOldEngine below).
func oldTaxOldRegime(income float64) float64 {
	if income <= 250000 {
		return 0
	}
	t := 0.0
	if income > 1000000 {
		t += (income - 1000000) * 0.30
	}
	if income > 500000 {
		t += (math.Min(income, 1000000) - 500000) * 0.20
	}
	if income > 250000 {
		t += (math.Min(income, 500000) - 250000) * 0.05
	}
	surcharge := 0.0
	if income > 10000000 {
		surcharge = t * 0.15
	} else if income > 5000000 {
		surcharge = t * 0.10
	}
	cess := (t + surcharge) * 0.04
	return math.Round(t + surcharge + cess)
}

func oldTaxNewRegime(gross float64) float64 {
	taxable := math.Max(0, gross-75000)
	t := 0.0
	if taxable > 2400000 {
		t += (taxable - 2400000) * 0.30
	}
	if taxable > 2000000 {
		t += (math.Min(taxable, 2400000) - 2000000) * 0.25
	}
	if taxable > 1600000 {
		t += (math.Min(taxable, 2000000) - 1600000) * 0.20
	}
	if taxable > 1200000 {
		t += (math.Min(taxable, 1600000) - 1200000) * 0.15
	}
	if taxable > 800000 {
		t += (math.Min(taxable, 1200000) - 800000) * 0.10
	}
	if taxable > 400000 {
		t += (math.Min(taxable, 800000) - 400000) * 0.05
	}
	if taxable <= 1200000 {
		t = 0 // Section 87A rebate
	}
	surcharge := 0.0
	if gross > 10000000 {
		surcharge = t * 0.15
	} else if gross > 5000000 {
		surcharge = t * 0.10
	}
	cess := (t + surcharge) * 0.04
	return math.Round(t + surcharge + cess)
}

// incomeSamples spans every slab boundary (just under/at/over) in both
// regimes, the 87A rebate thresholds, and the surcharge thresholds.
func incomeSamples() []float64 {
	return []float64{
		0, 1, 100000,
		249999, 250000, 250001,
		399999, 400000, 400001,
		499999, 500000, 500001,
		799999, 800000, 800001,
		999999, 1000000, 1000001,
		1199999, 1200000, 1200001,
		1599999, 1600000, 1600001,
		1999999, 2000000, 2000001,
		2399999, 2400000, 2400001,
		4999999, 5000000, 5000001,
		9999999, 10000000, 10000001,
		1234567, 3456789, 7654321, 15000000,
	}
}

func TestComputeTax_NewRegime_MatchesOldEngine(t *testing.T) {
	rules, err := models.DefaultTaxRuleSet("new").ParseRules()
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}
	for _, income := range incomeSamples() {
		want := oldTaxNewRegime(income)
		got := models.ComputeTax(rules, income)
		if got != want {
			t.Errorf("income=%.0f: ComputeTax(new) = %v, old oldTaxNewRegime = %v", income, got, want)
		}
	}
}

// New regime behavior is unchanged by the migration (it already had a full
// 87A rebate), so this is a strict byte-identical comparison. The old
// regime is NOT — see TestComputeTax_OldRegime_FixesMissing87ARebate.
func TestComputeTax_OldRegime_MatchesOldEngine_AboveRebateBand(t *testing.T) {
	rules, err := models.DefaultTaxRuleSet("old").ParseRules()
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}
	for _, income := range incomeSamples() {
		if income <= 500000 {
			continue // covered separately below — this is the documented behavior change
		}
		want := oldTaxOldRegime(income)
		got := models.ComputeTax(rules, income)
		if got != want {
			t.Errorf("income=%.0f: ComputeTax(old) = %v, old oldTaxOldRegime = %v", income, got, want)
		}
	}
}

// The pre-rules-engine oldTaxOldRegime() never implemented a Section 87A
// rebate for the old regime at all (real law zeroes tax for taxable income
// <= Rs 5,00,000). The new generic engine's default rule set includes it —
// an intentional bug fix that ships as a side effect of this migration, not
// a byte-identical port. This test documents that difference explicitly so
// it can never be mistaken for a regression.
func TestComputeTax_OldRegime_FixesMissing87ARebate(t *testing.T) {
	rules, err := models.DefaultTaxRuleSet("old").ParseRules()
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}
	for _, income := range []float64{300000, 400000, 499999, 500000} {
		oldBuggy := oldTaxOldRegime(income)
		fixed := models.ComputeTax(rules, income)
		if oldBuggy == 0 {
			t.Fatalf("income=%.0f: expected the old buggy function to show nonzero tax here to make this test meaningful, got 0", income)
		}
		if fixed != 0 {
			t.Errorf("income=%.0f: ComputeTax(old) = %v, want 0 (Section 87A rebate should zero tax for taxable income <= 5L)", income, fixed)
		}
	}
	// Just above the rebate threshold, both should agree tax is nonzero.
	if got := models.ComputeTax(rules, 500001); got <= 0 {
		t.Errorf("income=500001: ComputeTax(old) = %v, want > 0 (just above the rebate threshold)", got)
	}
}
