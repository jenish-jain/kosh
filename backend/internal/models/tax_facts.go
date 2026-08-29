package models

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// ── Tax slab math (Go mirror of frontend/src/screens/tax/taxMath.js) ───────
// Kept in sync manually — the frontend copy renders instantly with no round
// trip; this copy lets the AI recommendation facts be computed server-side
// from data the server just read, without trusting client-submitted numbers.

const newRegimeStdDeduction = 75000.0

func taxOldRegime(income float64) float64 {
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

func taxNewRegime(gross float64) float64 {
	taxable := math.Max(0, gross-newRegimeStdDeduction)
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

func taxPayable(regime string, gross float64) float64 {
	if strings.EqualFold(regime, "new") {
		return taxNewRegime(gross)
	}
	return taxOldRegime(gross)
}

// ── Tax facts ────────────────────────────────────────────────────────────

// MFCategoryTotals sums invested/current value for one MF category.
type MFCategoryTotals struct {
	Invested float64
	Current  float64
}

// FYInflow is one instrument contributing to a next-FY inflow projection.
type FYInflow struct {
	Name   string
	When   string
	Amount float64
}

// TaxFacts is the deterministic set of numbers handed to the AI
// recommendation prompt. All figures here are computed in Go from real data
// — the AI only prioritizes, explains trade-offs, and drafts recommendation
// text/amounts from these facts; it never computes tax numbers itself.
type TaxFacts struct {
	FY     string
	Regime string
	Today  string

	GrossIncome float64
	TaxPayable  float64
	EffRate     float64

	Deduction80CUsed float64
	Deduction80CCap  float64
	NPSInvestedTotal float64 // lifetime NPS contributions — no per-transaction dates to scope to this FY
	NPS80CCD1BCap    float64
	Deduction80DUsed float64
	Deduction80DCap  float64

	PFAnnualized     float64
	ELSSThisFYApprox float64
	LifeInsPremium   float64
	HealthInsPremium float64

	MFByCategory map[string]MFCategoryTotals

	NextFYTaxableInterest []FYInflow
	NextFYExemptMaturity  []FYInflow
	NextFYTotalTaxable    float64
	NextFYTotalExempt     float64

	// Caveats are known approximations, surfaced so the AI hedges in its
	// rationale rather than overclaiming precision.
	Caveats []string
}

// ComputeTaxFacts derives TaxFacts for `regime` as of `at`.
func ComputeTaxFacts(d *Data, regime string, at time.Time) TaxFacts {
	fy := CurrentFY(at)
	regime = strings.ToLower(regime)
	if regime != "new" {
		regime = "old"
	}

	gross := d.Config.GrossIncome
	if period, monthlyGross := latestIncomeGross(d.Income); period != "" && monthlyGross > 0 {
		gross = monthlyGross * 12
	}
	payable := taxPayable(regime, gross)
	effRate := 0.0
	if gross > 0 {
		effRate = payable / gross * 100
	}

	pfAnnual := latestIncomePF(d.Income) * 12

	lifeIns, healthIns := 0.0, 0.0
	for _, ins := range d.Insurance {
		annual := annualizePremium(ins.Premium, ins.Freq)
		if strings.Contains(strings.ToLower(ins.Type), "health") || strings.Contains(strings.ToLower(ins.Type), "mediclaim") {
			healthIns += annual
		} else {
			lifeIns += annual
		}
	}

	npsTotal := 0.0
	for _, n := range d.NPS {
		npsTotal += n.Invested
	}

	mfByCategory := map[string]MFCategoryTotals{}
	for _, mf := range d.MF {
		cat := mf.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		tot := mfByCategory[cat]
		tot.Invested += mf.Invested
		tot.Current += mf.Current
		mfByCategory[cat] = tot
	}

	elssApprox := elssThisFYApprox(d, fy)
	deduction80C := pfAnnual + lifeIns + elssApprox

	inTaxable, inExempt, totalTaxable, totalExempt := nextFYInflows(d, fy, at)

	caveats := []string{
		"ELSS this-FY contribution is estimated from ELSS-tagged active SIPs (× months elapsed this FY) plus lumpsums dated this FY against ELSS-tagged funds — MF invested totals are lifetime-cumulative, not FY-scoped, so treat this as an estimate.",
		"NPS invested total is lifetime-cumulative — there are no per-contribution dates, so this-FY NPS contribution toward the ₹50,000 80CCD(1B) limit cannot be computed precisely; verify against actual NPS statements.",
		"Insurance maturity year is year-only in the data — exempt-inflow timing is approximate to the calendar year, not an exact date. Sec 10(10D) exemption eligibility is not verified (depends on premium-to-cover ratio and policy terms).",
	}

	return TaxFacts{
		FY:     fy.Label,
		Regime: regime,
		Today:  at.Format("2006-01-02"),

		GrossIncome: gross,
		TaxPayable:  payable,
		EffRate:     effRate,

		Deduction80CUsed: deduction80C,
		Deduction80CCap:  150000,
		NPSInvestedTotal: npsTotal,
		NPS80CCD1BCap:    50000,
		Deduction80DUsed: healthIns,
		Deduction80DCap:  deduction80DCap(d),

		PFAnnualized:     pfAnnual,
		ELSSThisFYApprox: elssApprox,
		LifeInsPremium:   lifeIns,
		HealthInsPremium: healthIns,

		MFByCategory: mfByCategory,

		NextFYTaxableInterest: inTaxable,
		NextFYExemptMaturity:  inExempt,
		NextFYTotalTaxable:    totalTaxable,
		NextFYTotalExempt:     totalExempt,

		Caveats: caveats,
	}
}

func annualizePremium(premium float64, freq string) float64 {
	switch strings.ToLower(freq) {
	case "monthly":
		return premium * 12
	case "quarterly":
		return premium * 4
	default: // annual
		return premium
	}
}

func deduction80DCap(d *Data) float64 {
	cap := 25000.0 // self + family, non-senior
	for _, m := range d.Members {
		if m.ID != "you" && strings.Contains(strings.ToLower(m.Relation), "sr") {
			cap += 50000.0 // covers a senior-citizen parent
			break
		}
	}
	return cap
}

// latestIncomeGross returns the latest period string and its summed monthly
// gross across all income sources for that period.
func latestIncomeGross(income []Income) (period string, gross float64) {
	var latest time.Time
	for _, r := range income {
		t, err := time.Parse("Jan 2006", r.Period)
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
			period = r.Period
		}
	}
	for _, r := range income {
		if r.Period == period {
			gross += r.Gross
		}
	}
	return
}

// latestIncomePF returns the summed PF deduction for the latest income period.
func latestIncomePF(income []Income) float64 {
	period, _ := latestIncomeGross(income)
	pf := 0.0
	for _, r := range income {
		if r.Period == period {
			pf += r.PFDeduction
		}
	}
	return pf
}

func fuzzyMatch(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// elssThisFYApprox estimates this-FY contribution to ELSS-tagged funds from
// active SIPs (× months elapsed in the FY so far) plus lumpsums dated within
// the FY window, fuzzy-matched by fund name (see the Caveats on TaxFacts).
func elssThisFYApprox(d *Data, fy FYWindow) float64 {
	elssNames := map[string]bool{}
	for _, mf := range d.MF {
		if strings.EqualFold(mf.Category, "ELSS") {
			elssNames[strings.ToLower(mf.Name)] = true
		}
	}
	if len(elssNames) == 0 {
		return 0
	}
	isELSS := func(fund string) bool {
		for name := range elssNames {
			if fuzzyMatch(fund, name) {
				return true
			}
		}
		return false
	}

	now := time.Now()
	monthsElapsed := (now.Year()-fy.Start.Year())*12 + int(now.Month()-fy.Start.Month()) + 1
	if monthsElapsed < 0 {
		monthsElapsed = 0
	}
	if monthsElapsed > 12 {
		monthsElapsed = 12
	}

	total := 0.0
	for _, s := range d.SIPs {
		if s.Status == "active" && isELSS(s.Fund) {
			total += s.Amount * float64(monthsElapsed)
		}
	}
	for _, l := range d.Lumpsums {
		date, err := time.Parse("2006-01-02", l.Date)
		if err != nil {
			continue
		}
		if isELSS(l.Fund) && !date.Before(fy.Start) && !date.After(fy.End) {
			total += l.Amount
		}
	}
	return total
}

// nextFYInflows mirrors PotentialInflows.jsx's computePotentialInflows: FD/RD
// interest accrued to maturity (taxable, not a capital gain) for entries
// maturing between `at` and the FY's end, and insurance maturities whose
// year matches the FY's spanning calendar years (typically exempt).
func nextFYInflows(d *Data, fy FYWindow, at time.Time) (taxable, exempt []FYInflow, totalTaxable, totalExempt float64) {
	for _, f := range d.Fixed {
		if f.Matures == "" {
			continue
		}
		m, err := time.Parse("2006-01-02", f.Matures)
		if err != nil || m.Before(at) || m.After(fy.End) {
			continue
		}
		interest := FDValueAtMaturity(f) - f.Principal
		if interest <= 0 {
			continue
		}
		taxable = append(taxable, FYInflow{Name: f.Name, When: f.Matures, Amount: interest})
		totalTaxable += interest
	}

	fyYears := map[int]bool{fy.Start.Year(): true, fy.End.Year(): true}
	for _, ins := range d.Insurance {
		if !fyYears[ins.Maturity] || ins.Value <= 0 {
			continue
		}
		exempt = append(exempt, FYInflow{Name: ins.Name, When: "Matures " + strconv.Itoa(ins.Maturity), Amount: ins.Value})
		totalExempt += ins.Value
	}
	return
}
