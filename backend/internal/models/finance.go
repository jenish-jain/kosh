package models

import (
	"math"
	"time"
)

// FDCurrentValue computes the current value of a fixed deposit using annual
// compounding: P × (1 + r)^t.
func FDCurrentValue(principal, ratePercent float64, opened string) float64 {
	if principal == 0 {
		return 0
	}
	t := YearsSince(opened)
	if t <= 0 {
		return principal
	}
	r := ratePercent / 100.0
	return principal * math.Pow(1+r, t)
}

// RDCurrentValue computes the current value of a recurring deposit. Each
// monthly installment compounds annually from its deposit date. Installments
// are capped at the tenure derived from opened→matures.
func RDCurrentValue(monthly, ratePercent float64, opened, matures string) float64 {
	if monthly == 0 {
		return 0
	}
	r := ratePercent / 100.0
	tenure := MonthsBetween(opened, matures)
	elapsed := MonthsBetween(opened, time.Now().Format("2006-01-02"))
	if elapsed <= 0 {
		return 0
	}
	made := elapsed
	if tenure > 0 && made > tenure {
		made = tenure
	}
	total := 0.0
	for i := 0; i < made; i++ {
		years := float64(elapsed-i) / 12.0
		total += monthly * math.Pow(1+r, years)
	}
	return total
}

// YearsSince returns the number of years elapsed since dateStr (YYYY-MM-DD).
// Returns 0 on parse error.
func YearsSince(dateStr string) float64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	return time.Since(t).Hours() / (24 * 365.25)
}

// MonthsBetween returns the number of whole calendar months from fromStr to
// toStr (both YYYY-MM-DD). Returns 0 if either date is unparseable or if
// toStr is before fromStr.
func MonthsBetween(fromStr, toStr string) int {
	from, e1 := time.Parse("2006-01-02", fromStr)
	to, e2 := time.Parse("2006-01-02", toStr)
	if e1 != nil || e2 != nil {
		return 0
	}
	m := (to.Year()-from.Year())*12 + int(to.Month()-from.Month())
	if m < 0 {
		return 0
	}
	return m
}

// YearsBetween returns the number of years from fromStr to toStr (both
// YYYY-MM-DD). Returns 0 on parse error or if toStr is before fromStr.
func YearsBetween(fromStr, toStr string) float64 {
	from, e1 := time.Parse("2006-01-02", fromStr)
	to, e2 := time.Parse("2006-01-02", toStr)
	if e1 != nil || e2 != nil || to.Before(from) {
		return 0
	}
	return to.Sub(from).Hours() / (24 * 365.25)
}

// RDValueAtMaturity returns a recurring deposit's value once every
// installment through `matures` has been made and compounded to that date
// (as opposed to RDCurrentValue, which projects only installments made so far).
func RDValueAtMaturity(monthly, ratePercent float64, opened, matures string) float64 {
	if monthly == 0 {
		return 0
	}
	tenure := MonthsBetween(opened, matures)
	if tenure <= 0 {
		return 0
	}
	r := ratePercent / 100.0
	total := 0.0
	for i := 0; i < tenure; i++ {
		years := float64(tenure-i) / 12.0
		total += monthly * math.Pow(1+r, years)
	}
	return total
}

// FDValueAtMaturity returns a Fixed entry's projected value at its own
// `matures` date (as opposed to ComputeFixedValue, which projects to now).
func FDValueAtMaturity(f Fixed) float64 {
	if f.Kind == "RD" {
		return RDValueAtMaturity(f.Monthly, f.Rate, f.Opened, f.Matures)
	}
	if f.Principal == 0 || f.Opened == "" || f.Matures == "" {
		return f.Principal
	}
	years := YearsBetween(f.Opened, f.Matures)
	if years <= 0 {
		return f.Principal
	}
	r := f.Rate / 100.0
	return f.Principal * math.Pow(1+r, years)
}

// ComputeFixedValue returns the computed current value for a single Fixed entry.
func ComputeFixedValue(f Fixed) float64 {
	if f.Rate == 0 || f.Opened == "" {
		return f.Principal
	}
	if f.Kind == "RD" {
		return RDCurrentValue(f.Monthly, f.Rate, f.Opened, f.Matures)
	}
	return FDCurrentValue(f.Principal, f.Rate, f.Opened)
}

// ComputeFixedValues fills CurrentValue for every entry in the slice in place.
func ComputeFixedValues(fixed []Fixed) {
	for i := range fixed {
		fixed[i].CurrentValue = ComputeFixedValue(fixed[i])
	}
}
