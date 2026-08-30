// Package report builds CA-shareable exports (PDF/XLSX) from the same
// financial data every screen renders — a data-shaping step (BuildTables)
// feeds two renderers so "what columns/values go in a report" is defined
// once, not twice.
package report

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// FormatINR formats n as Indian-grouped rupees, e.g. 1234567 -> "₹12,34,567".
// Mirrors frontend/src/data/format.js's fmtINR.
func FormatINR(n float64) string {
	if n == 0 {
		return "₹0"
	}
	neg := n < 0
	v := int64(math.Round(math.Abs(n)))
	s := "₹" + groupIndian(strconv.FormatInt(v, 10))
	if neg {
		return "−" + s
	}
	return s
}

// groupIndian groups a digit string as ...,XX,XXX (last 3 together, then
// pairs of 2 to the left) — the Indian lakh/crore comma convention.
func groupIndian(s string) string {
	if len(s) <= 3 {
		return s
	}
	last3 := s[len(s)-3:]
	other := s[:len(s)-3]
	var parts []string
	for len(other) > 2 {
		parts = append([]string{other[len(other)-2:]}, parts...)
		other = other[:len(other)-2]
	}
	if len(other) > 0 {
		parts = append([]string{other}, parts...)
	}
	return strings.Join(parts, ",") + "," + last3
}

// FormatDate renders a YYYY-MM-DD date string as "29 Aug 2026", matching
// frontend/src/data/format.js's fmtDate. Returns the input unchanged if it
// doesn't parse (so a malformed/empty date never crashes report generation).
func FormatDate(s string) string {
	if s == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("2 Jan 2006")
}

// FormatNum renders a plain number with no currency symbol, trimming
// trailing zeros after the decimal point (e.g. quantities, rates).
func FormatNum(n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	return s
}
