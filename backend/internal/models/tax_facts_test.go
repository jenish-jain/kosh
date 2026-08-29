package models_test

import (
	"testing"
	"time"

	"kosh/internal/models"
)

func TestComputeTaxFacts_NextFYInflows_ExcludesAlreadyMaturedFDs(t *testing.T) {
	at := time.Now()
	fy := models.CurrentFY(at)

	d := &models.Data{
		Fixed: []models.Fixed{
			// Matured before "today" — must NOT count as a future inflow.
			{ID: "past", Name: "Past FD", Principal: 100000, Rate: 7, Opened: dateMonthsAgo(24), Matures: dateMonthsAgo(1)},
			// Matures within the FY window, after today — must count.
			{ID: "future", Name: "Future FD", Principal: 100000, Rate: 7, Opened: dateMonthsAgo(12), Matures: dateMonthsLater(2)},
		},
	}

	facts := models.ComputeTaxFacts(d, "old", at)

	if facts.NextFYTotalTaxable <= 0 {
		t.Fatalf("NextFYTotalTaxable = %v, want > 0 (the future-maturing FD's interest)", facts.NextFYTotalTaxable)
	}
	for _, f := range facts.NextFYTaxableInterest {
		if f.Name == "Past FD" {
			t.Errorf("already-matured FD %q must not appear in NextFYTaxableInterest", f.Name)
		}
	}
	if len(facts.NextFYTaxableInterest) != 1 || facts.NextFYTaxableInterest[0].Name != "Future FD" {
		t.Errorf("NextFYTaxableInterest = %+v, want exactly the future-maturing FD", facts.NextFYTaxableInterest)
	}
	_ = fy
}

func TestComputeTaxFacts_ExemptMaturity_MatchesFYCalendarYears(t *testing.T) {
	at := time.Now()
	fy := models.CurrentFY(at)

	d := &models.Data{
		Insurance: []models.Insurance{
			{ID: "in-fy", Name: "In FY", Value: 260000, Maturity: fy.End.Year()},
			{ID: "far-future", Name: "Far future", Value: 300000, Maturity: fy.End.Year() + 10},
		},
	}

	facts := models.ComputeTaxFacts(d, "old", at)

	if facts.NextFYTotalExempt != 260000 {
		t.Errorf("NextFYTotalExempt = %v, want 260000 (only the in-FY policy)", facts.NextFYTotalExempt)
	}
	if len(facts.NextFYExemptMaturity) != 1 || facts.NextFYExemptMaturity[0].Name != "In FY" {
		t.Errorf("NextFYExemptMaturity = %+v, want exactly the in-FY policy", facts.NextFYExemptMaturity)
	}
}

func TestComputeTaxFacts_Deduction80C_SumsPFInsuranceAndELSS(t *testing.T) {
	at := time.Now()

	d := &models.Data{
		Income: []models.Income{
			{Period: at.Format("Jan 2006"), Gross: 400000, PFDeduction: 14400},
		},
		Insurance: []models.Insurance{
			{Type: "Term", Premium: 14500, Freq: "annual"}, // life insurance -> counts toward 80C
			{Type: "Health", Premium: 20000, Freq: "annual"}, // health -> counts toward 80D, not 80C
		},
	}

	facts := models.ComputeTaxFacts(d, "old", at)

	wantPF := 14400.0 * 12
	if facts.PFAnnualized != wantPF {
		t.Errorf("PFAnnualized = %v, want %v", facts.PFAnnualized, wantPF)
	}
	if facts.LifeInsPremium != 14500 {
		t.Errorf("LifeInsPremium = %v, want 14500", facts.LifeInsPremium)
	}
	if facts.HealthInsPremium != 20000 {
		t.Errorf("HealthInsPremium = %v, want 20000 (must not be double-counted into 80C)", facts.HealthInsPremium)
	}
	wantDeduction80C := wantPF + 14500 // PF + life insurance; no ELSS in this fixture
	if facts.Deduction80CUsed != wantDeduction80C {
		t.Errorf("Deduction80CUsed = %v, want %v", facts.Deduction80CUsed, wantDeduction80C)
	}
}

func TestComputeTaxFacts_NewRegime_UsesNewSlabs(t *testing.T) {
	d := &models.Data{Config: models.Config{GrossIncome: 4800000}}
	facts := models.ComputeTaxFacts(d, "new", time.Now())
	if facts.Regime != "new" {
		t.Errorf("Regime = %q, want %q", facts.Regime, "new")
	}
	if facts.TaxPayable <= 0 {
		t.Errorf("TaxPayable = %v, want > 0 for ₹48L gross income", facts.TaxPayable)
	}
}
