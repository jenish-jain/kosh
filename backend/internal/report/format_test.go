package report_test

import (
	"testing"

	"kosh/internal/report"
)

func TestFormatINR(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "₹0"},
		{100, "₹100"},
		{1000, "₹1,000"},
		{123456, "₹1,23,456"},
		{1234567, "₹12,34,567"},
		{12345678, "₹1,23,45,678"},
		{-2700, "−₹2,700"},
		{999.6, "₹1,000"}, // rounds
	}
	for _, c := range cases {
		if got := report.FormatINR(c.in); got != c.want {
			t.Errorf("FormatINR(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"2026-08-29", "29 Aug 2026"},
		{"2026-01-01", "1 Jan 2026"},
		{"not-a-date", "not-a-date"}, // falls back to input rather than crashing
	}
	for _, c := range cases {
		if got := report.FormatDate(c.in); got != c.want {
			t.Errorf("FormatDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
