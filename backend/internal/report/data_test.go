package report_test

import (
	"testing"
	"time"

	"kosh/internal/models"
	"kosh/internal/report"
)

func exampleNow() time.Time {
	return time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
}

func sampleData() *models.Data {
	return &models.Data{
		Members: []models.Member{
			{ID: "you", Name: "You"},
			{ID: "mom", Name: "Mom"},
		},
		MF: []models.MFRow{
			{ID: "mf1", Name: "Fund A", Member: "you", Invested: 1000, Current: 1200, SIP: 100},
			{ID: "mf2", Name: "Fund B", Member: "mom", Invested: 2000, Current: 1900, SIP: 0},
		},
		Stocks: []models.Stock{
			{ID: "st1", Name: "Acme", Member: "you", Qty: 10, AvgPrice: 100, LastPrice: 120},
		},
		Fixed: []models.Fixed{
			{ID: "fd1", Name: "Bank FD", Member: "you", Principal: 50000, CurrentValue: 53000, Opened: "2025-01-01", Matures: "2026-01-01"},
		},
		Income: []models.Income{
			{ID: "inc1", Period: "Jul 2026", Source: "Employer", Gross: 100000, Net: 80000, PFDeduction: 5000, TaxDeduction: 10000, OtherDeductions: 5000},
		},
		Loans: []models.Loan{
			{ID: "ln1", Lender: "Bank", Member: "you", Principal: 100000, Outstanding: 60000, EMI: 5000},
		},
	}
}

func TestBuildTables_MemberFiltering(t *testing.T) {
	d := sampleData()

	whole := report.BuildTables(d, "", []string{"mf"})
	if len(whole) != 1 || len(whole[0].Rows) != 2 {
		t.Fatalf("whole-family MF: got %+v, want 2 rows", whole)
	}
	// Whole-family scope includes an Owner column.
	if whole[0].Headers[2] != "Owner" {
		t.Errorf("expected an Owner column in whole-family scope, headers = %v", whole[0].Headers)
	}

	you := report.BuildTables(d, "you", []string{"mf"})
	if len(you) != 1 || len(you[0].Rows) != 1 {
		t.Fatalf("you-scoped MF: got %+v, want 1 row", you)
	}
	if you[0].Headers[2] == "Owner" {
		t.Errorf("single-member scope should not include an Owner column, headers = %v", you[0].Headers)
	}
	if you[0].Rows[0][0] != "Fund A" {
		t.Errorf("you-scoped MF row = %v, want Fund A", you[0].Rows[0])
	}
}

func TestBuildTables_StocksComputedValues(t *testing.T) {
	d := sampleData()
	tables := report.BuildTables(d, "", []string{"stocks"})
	row := tables[0].Rows[0]
	// Headers: Stock, Ticker, Owner, Qty, Avg Price, Last Price, Invested, Current Value, Gain
	if row[6] != "₹1,000" { // 10 * 100
		t.Errorf("invested = %q, want ₹1,000", row[6])
	}
	if row[7] != "₹1,200" { // 10 * 120
		t.Errorf("current = %q, want ₹1,200", row[7])
	}
	if row[8] != "₹200" {
		t.Errorf("gain = %q, want ₹200", row[8])
	}
}

func TestBuildTables_IncomeIgnoresMemberFilter(t *testing.T) {
	d := sampleData()
	// Income has no Member field in the data model at all — it must always
	// return the same rows regardless of the member scope, matching how
	// Income.jsx already behaves (not a gap to "fix").
	whole := report.BuildTables(d, "", []string{"income"})
	you := report.BuildTables(d, "you", []string{"income"})
	if len(whole[0].Rows) != 1 || len(you[0].Rows) != 1 {
		t.Fatalf("expected income rows unaffected by member scope: whole=%v you=%v", whole[0].Rows, you[0].Rows)
	}
}

func TestBuildTables_SummaryComputesNetWorth(t *testing.T) {
	d := sampleData()
	tables := report.BuildTables(d, "", []string{"summary"})
	if len(tables) != 1 {
		t.Fatalf("expected exactly one summary table, got %d", len(tables))
	}
	rows := tables[0].Rows
	var netWorthRow []string
	for _, r := range rows {
		if r[0] == "Net Worth" {
			netWorthRow = r
		}
	}
	if netWorthRow == nil {
		t.Fatalf("no Net Worth row found in %v", rows)
	}
	// Assets: MF 1200+1900=3100, Stocks 1200, FD 53000 = 57300. Liabilities: 60000.
	// Net worth = 57300 - 60000 = -2700.
	want := "−₹2,700"
	if netWorthRow[2] != want {
		t.Errorf("Net Worth = %q, want %q", netWorthRow[2], want)
	}
}

func TestBuildTables_RespectsRequestedSectionsAndOrder(t *testing.T) {
	d := sampleData()
	// Request in a scrambled order — output must follow the canonical order.
	tables := report.BuildTables(d, "", []string{"income", "mf", "summary"})
	if len(tables) != 3 {
		t.Fatalf("got %d tables, want 3", len(tables))
	}
	got := []string{tables[0].Title, tables[1].Title, tables[2].Title}
	want := []string{"Net Worth Summary", "Mutual Funds", "Income"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("table[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildTables_SIPsSectionAlsoIncludesLumpsums(t *testing.T) {
	d := sampleData()
	d.SIPs = []models.SIP{{ID: "s1", Fund: "Fund A", Member: "you", Amount: 500, Status: "active"}}
	d.Lumpsums = []models.Lumpsum{{ID: "l1", Fund: "Fund A", Member: "you", Amount: 2000, Date: "2026-01-01"}}

	tables := report.BuildTables(d, "", []string{"sips"})
	if len(tables) != 2 {
		t.Fatalf("expected SIPs + Lumpsums tables, got %d: %+v", len(tables), tables)
	}
	if tables[0].Title != "SIPs" || tables[1].Title != "Lumpsum Top-ups" {
		t.Errorf("unexpected titles: %q, %q", tables[0].Title, tables[1].Title)
	}
}

func TestBuildMeta_ScopeLabel(t *testing.T) {
	d := sampleData()
	if got := report.BuildMeta(d, "", exampleNow()).Scope; got != "Whole family" {
		t.Errorf("Scope = %q, want %q", got, "Whole family")
	}
	if got := report.BuildMeta(d, "mom", exampleNow()).Scope; got != "Mom" {
		t.Errorf("Scope = %q, want %q", got, "Mom")
	}
}
