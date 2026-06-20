package models_test

import (
	"testing"
	"time"

	"kosh/internal/models"
)

// dateYearsAgo returns a YYYY-MM-DD string for a date approximately n years
// before today. Used to construct test inputs without hard-coding a date that
// ages out of range.
func dateYearsAgo(n int) string {
	return time.Now().AddDate(-n, 0, 0).Format("2006-01-02")
}

// dateMonthsAgo returns a YYYY-MM-DD string n months before today.
func dateMonthsAgo(n int) string {
	return time.Now().AddDate(0, -n, 0).Format("2006-01-02")
}

// dateMonthsLater returns a YYYY-MM-DD string n months after today.
func dateMonthsLater(n int) string {
	return time.Now().AddDate(0, n, 0).Format("2006-01-02")
}

// ── YearsSince ────────────────────────────────────────────────────────────────

func TestYearsSince_ValidDate_ReturnsPositive(t *testing.T) {
	date := dateYearsAgo(3)
	got := models.YearsSince(date)
	if got <= 0 {
		t.Errorf("YearsSince(%q) = %v; want > 0", date, got)
	}
}

func TestYearsSince_FutureDate_ReturnsNonPositive(t *testing.T) {
	future := dateMonthsLater(6)
	got := models.YearsSince(future)
	if got > 0 {
		t.Errorf("YearsSince(%q) = %v; want <= 0 for a future date", future, got)
	}
}

func TestYearsSince_BadDate_ReturnsZero(t *testing.T) {
	got := models.YearsSince("not-a-date")
	if got != 0 {
		t.Errorf("YearsSince(%q) = %v; want 0", "not-a-date", got)
	}
}

func TestYearsSince_ApproximatelyCorrect(t *testing.T) {
	date := dateYearsAgo(3)
	got := models.YearsSince(date)
	// Allow ±0.1 year tolerance for leap-year and sub-day rounding.
	if got < 2.9 || got > 3.1 {
		t.Errorf("YearsSince(%q) = %v; want ~3.0", date, got)
	}
}

// ── MonthsBetween ─────────────────────────────────────────────────────────────

func TestMonthsBetween_BasicCalculation(t *testing.T) {
	got := models.MonthsBetween("2023-01-01", "2023-07-01")
	if got != 6 {
		t.Errorf("MonthsBetween(2023-01-01, 2023-07-01) = %d; want 6", got)
	}
}

func TestMonthsBetween_ReversedDates_ReturnsZero(t *testing.T) {
	got := models.MonthsBetween("2024-06-01", "2023-01-01")
	if got != 0 {
		t.Errorf("MonthsBetween reversed = %d; want 0", got)
	}
}

func TestMonthsBetween_BadFromDate_ReturnsZero(t *testing.T) {
	got := models.MonthsBetween("bad", "2024-01-01")
	if got != 0 {
		t.Errorf("MonthsBetween(bad, ...) = %d; want 0", got)
	}
}

func TestMonthsBetween_BadToDate_ReturnsZero(t *testing.T) {
	got := models.MonthsBetween("2024-01-01", "bad")
	if got != 0 {
		t.Errorf("MonthsBetween(..., bad) = %d; want 0", got)
	}
}

func TestMonthsBetween_SameDate_ReturnsZero(t *testing.T) {
	got := models.MonthsBetween("2024-03-15", "2024-03-15")
	if got != 0 {
		t.Errorf("MonthsBetween same date = %d; want 0", got)
	}
}

func TestMonthsBetween_CrossYear(t *testing.T) {
	got := models.MonthsBetween("2022-11-01", "2023-03-01")
	if got != 4 {
		t.Errorf("MonthsBetween cross-year = %d; want 4", got)
	}
}

// ── FDCurrentValue ────────────────────────────────────────────────────────────

func TestFDCurrentValue_ZeroPrincipal_ReturnsZero(t *testing.T) {
	got := models.FDCurrentValue(0, 7.0, dateYearsAgo(2))
	if got != 0 {
		t.Errorf("FDCurrentValue(0, ...) = %v; want 0", got)
	}
}

func TestFDCurrentValue_ZeroRate_ReturnsPrincipal(t *testing.T) {
	got := models.FDCurrentValue(100000, 0, dateYearsAgo(2))
	// Rate 0 is handled by ComputeFixedValue; FDCurrentValue with r=0 gives P*(1+0)^t = P.
	if got != 100000 {
		t.Errorf("FDCurrentValue(100000, 0, ...) = %v; want 100000", got)
	}
}

func TestFDCurrentValue_BasicCompoundInterest(t *testing.T) {
	// P=100000, rate=7%, opened ~3 years ago.
	// Expected: ~122504 (100000 * 1.07^3).
	opened := dateYearsAgo(3)
	got := models.FDCurrentValue(100000, 7.0, opened)
	if got < 120000 || got > 126000 {
		t.Errorf("FDCurrentValue(100000, 7.0, ~3yr) = %v; want ~122504", got)
	}
}

func TestFDCurrentValue_FutureOpenDate_ReturnsPrincipal(t *testing.T) {
	// YearsSince returns ≤0 for a future date; function should return principal.
	got := models.FDCurrentValue(50000, 8.0, dateMonthsLater(3))
	if got != 50000 {
		t.Errorf("FDCurrentValue with future open date = %v; want 50000", got)
	}
}

func TestFDCurrentValue_EmptyDate_ReturnsZero(t *testing.T) {
	// YearsSince("") returns 0, so t<=0 → return principal.
	// Empty string is handled: yearsSince returns 0 → t<=0 → principal returned.
	got := models.FDCurrentValue(80000, 6.5, "")
	if got != 80000 {
		t.Errorf("FDCurrentValue with empty date = %v; want 80000", got)
	}
}

// ── RDCurrentValue ────────────────────────────────────────────────────────────

func TestRDCurrentValue_ZeroMonthly_ReturnsZero(t *testing.T) {
	got := models.RDCurrentValue(0, 7.0, dateMonthsAgo(12), dateMonthsLater(12))
	if got != 0 {
		t.Errorf("RDCurrentValue(0, ...) = %v; want 0", got)
	}
}

func TestRDCurrentValue_BasicSmoke(t *testing.T) {
	// 12 monthly installments of 5000 at 7%; result should be > 12*5000 = 60000.
	opened := dateMonthsAgo(12)
	matures := dateMonthsLater(12)
	got := models.RDCurrentValue(5000, 7.0, opened, matures)
	if got <= 60000 {
		t.Errorf("RDCurrentValue smoke = %v; want > 60000 (12 × 5000 + interest)", got)
	}
}

func TestRDCurrentValue_FutureOpenDate_ReturnsZero(t *testing.T) {
	// elapsed = MonthsBetween(future, now) = 0 → return 0.
	future := dateMonthsLater(6)
	got := models.RDCurrentValue(5000, 7.0, future, dateMonthsLater(18))
	if got != 0 {
		t.Errorf("RDCurrentValue with future open date = %v; want 0", got)
	}
}

// ── ComputeFixedValue ─────────────────────────────────────────────────────────

func TestComputeFixedValue_FDPath(t *testing.T) {
	f := models.Fixed{
		Kind:      "FD",
		Principal: 100000,
		Rate:      7.0,
		Opened:    dateYearsAgo(2),
	}
	got := models.ComputeFixedValue(f)
	// Should be roughly 100000 * 1.07^2 ≈ 114490.
	if got < 112000 || got > 118000 {
		t.Errorf("ComputeFixedValue(FD) = %v; want ~114490", got)
	}
}

func TestComputeFixedValue_RDPath(t *testing.T) {
	f := models.Fixed{
		Kind:    "RD",
		Rate:    7.0,
		Monthly: 5000,
		Opened:  dateMonthsAgo(6),
		Matures: dateMonthsLater(18),
	}
	got := models.ComputeFixedValue(f)
	// 6 installments of 5000 + interest; should be > 30000.
	if got <= 30000 {
		t.Errorf("ComputeFixedValue(RD) = %v; want > 30000", got)
	}
}

func TestComputeFixedValue_ZeroRate_ReturnsPrincipal(t *testing.T) {
	f := models.Fixed{
		Kind:      "FD",
		Principal: 75000,
		Rate:      0,
		Opened:    dateYearsAgo(1),
	}
	got := models.ComputeFixedValue(f)
	if got != 75000 {
		t.Errorf("ComputeFixedValue(rate=0) = %v; want 75000", got)
	}
}

func TestComputeFixedValue_EmptyOpened_ReturnsPrincipal(t *testing.T) {
	f := models.Fixed{
		Kind:      "FD",
		Principal: 50000,
		Rate:      8.0,
		Opened:    "",
	}
	got := models.ComputeFixedValue(f)
	if got != 50000 {
		t.Errorf("ComputeFixedValue(opened='') = %v; want 50000", got)
	}
}

// ── ComputeFixedValues ────────────────────────────────────────────────────────

func TestComputeFixedValues_ModifiesSliceInPlace(t *testing.T) {
	opened := dateYearsAgo(1)
	fixed := []models.Fixed{
		{Kind: "FD", Principal: 100000, Rate: 8.0, Opened: opened},
		{Kind: "FD", Principal: 50000, Rate: 0, Opened: opened},
	}
	models.ComputeFixedValues(fixed)

	if fixed[0].CurrentValue <= 100000 {
		t.Errorf("fixed[0].CurrentValue = %v; want > 100000 after 1yr at 8%%", fixed[0].CurrentValue)
	}
	if fixed[1].CurrentValue != 50000 {
		t.Errorf("fixed[1].CurrentValue = %v; want 50000 (rate=0)", fixed[1].CurrentValue)
	}
}

func TestComputeFixedValues_EmptySlice_NoOp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ComputeFixedValues(nil) panicked: %v", r)
		}
	}()
	models.ComputeFixedValues(nil)
	models.ComputeFixedValues([]models.Fixed{})
}

// ── NetWorthTotal (quick sanity via models package) ───────────────────────────

func TestNetWorthTotal_SimpleData(t *testing.T) {
	d := &models.Data{
		MF:    []models.MFRow{{Current: 100000}},
		Loans: []models.Loan{{Outstanding: 20000}},
	}
	got := models.NetWorthTotal(d)
	want := 80000.0
	if got != want {
		t.Errorf("NetWorthTotal = %v; want %v", got, want)
	}
}

func TestNetWorthTotal_EmptyData_ReturnsZero(t *testing.T) {
	got := models.NetWorthTotal(&models.Data{})
	if got != 0 {
		t.Errorf("NetWorthTotal(empty) = %v; want 0", got)
	}
}

