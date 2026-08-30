package report

import (
	"strconv"
	"time"

	"kosh/internal/models"
)

// Table is one section of a report — already formatted, so both renderers
// (PDF and XLSX) just lay these strings out rather than reimplementing
// "what columns/values go in a report" each.
type Table struct {
	Title   string     // PDF section heading / XLSX sheet name
	Headers []string
	Rows    [][]string
	Totals  []string // optional; same length as Headers, "" where not applicable
}

// ReportMeta carries the cover/header info shared by both renderers.
type ReportMeta struct {
	Scope     string // "Whole family" or the member's name — mirrors aggregate.js's scope()
	Generated string // formatted generation date
}

// sectionOrder is the canonical, fixed order tables appear in regardless of
// the order sections were requested in.
var sectionOrder = []string{"summary", "mf", "stocks", "metals", "fixed", "insurance", "nps", "loans", "sips", "income"}

var sectionLabels = map[string]string{
	"summary":   "Net Worth Summary",
	"mf":        "Mutual Funds",
	"stocks":    "Stocks",
	"metals":    "Gold & Silver",
	"fixed":     "Fixed Deposits",
	"insurance": "Insurance",
	"nps":       "NPS",
	"loans":     "Loans / Liabilities",
	"sips":      "SIPs",
	"income":    "Income",
}

// ValidSections returns the canonical section keys, in report order.
func ValidSections() []string {
	out := make([]string, len(sectionOrder))
	copy(out, sectionOrder)
	return out
}

// SectionLabel returns the human label for a section key, or "" if unknown.
func SectionLabel(key string) string { return sectionLabels[key] }

// BuildMeta resolves the report's scope label and generation timestamp.
func BuildMeta(d *models.Data, member string, now time.Time) ReportMeta {
	scope := "Whole family"
	if member != "" {
		if name := memberName(d.Members, member); name != "" {
			scope = name
		}
	}
	return ReportMeta{Scope: scope, Generated: now.Format("2 Jan 2006")}
}

// BuildTables builds one Table per requested section, in canonical order,
// scoped to `member` (empty = whole family) — mirrors
// frontend/src/data/aggregate.js's holdingsFor/classTotals in Go so the
// server can build a report independently of the client.
func BuildTables(d *models.Data, member string, sections []string) []Table {
	want := map[string]bool{}
	for _, s := range sections {
		want[s] = true
	}

	var tables []Table
	for _, key := range sectionOrder {
		if !want[key] {
			continue
		}
		switch key {
		case "summary":
			tables = append(tables, buildSummary(d, member))
		case "mf":
			tables = append(tables, buildMF(d.MF, member, d.Members))
		case "stocks":
			tables = append(tables, buildStocks(d.Stocks, member, d.Members))
		case "metals":
			tables = append(tables, buildMetals(d.Metals, member, d.Members))
		case "fixed":
			tables = append(tables, buildFixed(d.Fixed, member, d.Members))
		case "insurance":
			tables = append(tables, buildInsurance(d.Insurance, member, d.Members))
		case "nps":
			tables = append(tables, buildNPS(d.NPS, member, d.Members))
		case "loans":
			tables = append(tables, buildLoans(d.Loans, member, d.Members))
		case "sips":
			tables = append(tables, buildSIPs(d.SIPs, member, d.Members))
			tables = append(tables, buildLumpsums(d.Lumpsums, member, d.Members))
		case "income":
			tables = append(tables, buildIncome(d.Income))
		}
	}
	return tables
}

// filterByMember returns rows unchanged if member is "" (whole family),
// else only the rows belonging to that member — the same convention every
// screen already uses via aggregate.js's holdingsFor.
func filterByMember[T any](rows []T, member string, getMember func(T) string) []T {
	if member == "" {
		return rows
	}
	var out []T
	for _, r := range rows {
		if getMember(r) == member {
			out = append(out, r)
		}
	}
	return out
}

func memberName(members []models.Member, id string) string {
	for _, m := range members {
		if m.ID == id {
			return m.Name
		}
	}
	return id
}

func buildSummary(d *models.Data, member string) Table {
	mf := filterByMember(d.MF, member, func(r models.MFRow) string { return r.Member })
	stocks := filterByMember(d.Stocks, member, func(r models.Stock) string { return r.Member })
	metals := filterByMember(d.Metals, member, func(r models.Metal) string { return r.Member })
	fixed := filterByMember(d.Fixed, member, func(r models.Fixed) string { return r.Member })
	insurance := filterByMember(d.Insurance, member, func(r models.Insurance) string { return r.Member })
	nps := filterByMember(d.NPS, member, func(r models.NPS) string { return r.Member })
	loans := filterByMember(d.Loans, member, func(r models.Loan) string { return r.Member })

	var mfInv, mfCur float64
	for _, r := range mf {
		mfInv += r.Invested
		mfCur += r.Current
	}
	var stInv, stCur float64
	for _, r := range stocks {
		stInv += r.Qty * r.AvgPrice
		stCur += r.Qty * r.LastPrice
	}
	var meInv, meCur float64
	for _, r := range metals {
		meInv += r.Grams * r.BuyRate
		meCur += r.Grams * r.TodayPrice
	}
	var fdInv, fdCur float64
	for _, r := range fixed {
		fdInv += r.Principal
		fdCur += r.CurrentValue
	}
	var inInv, inCur float64
	for _, r := range insurance {
		inInv += r.Paid
		inCur += r.Value
	}
	var npsInv, npsCur float64
	for _, r := range nps {
		npsInv += r.Invested
		npsCur += r.Units * r.NAV
	}
	var liab float64
	for _, r := range loans {
		liab += r.Outstanding
	}

	row := func(label string, inv, cur float64) []string {
		return []string{label, FormatINR(inv), FormatINR(cur), FormatINR(cur - inv)}
	}

	totalInv := mfInv + stInv + meInv + fdInv + inInv + npsInv
	totalCur := mfCur + stCur + meCur + fdCur + inCur + npsCur
	netWorth := totalCur - liab

	rows := [][]string{
		row("Mutual Funds", mfInv, mfCur),
		row("Stocks", stInv, stCur),
		row("Gold & Silver", meInv, meCur),
		row("Fixed Deposits", fdInv, fdCur),
		row("Insurance (paid vs. value)", inInv, inCur),
		row("NPS", npsInv, npsCur),
		{"Total Assets", FormatINR(totalInv), FormatINR(totalCur), FormatINR(totalCur - totalInv)},
		{"Loans / Liabilities", "", FormatINR(liab), ""},
		{"Net Worth", "", FormatINR(netWorth), ""},
	}
	return Table{Title: "Net Worth Summary", Headers: []string{"Asset Class", "Invested", "Current Value", "Gain"}, Rows: rows}
}

func buildMF(rows []models.MFRow, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.MFRow) string { return r.Member })
	all := member == ""
	headers := []string{"Fund", "Platform"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Monthly SIP", "Invested", "Current Value", "Gain")

	var out [][]string
	var totSIP, totInv, totCur float64
	for _, r := range rows {
		row := []string{r.Name, r.Platform}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatINR(r.SIP), FormatINR(r.Invested), FormatINR(r.Current), FormatINR(r.Current-r.Invested))
		out = append(out, row)
		totSIP += r.SIP
		totInv += r.Invested
		totCur += r.Current
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-4], totals[n-3], totals[n-2], totals[n-1] = FormatINR(totSIP), FormatINR(totInv), FormatINR(totCur), FormatINR(totCur-totInv)
	return Table{Title: "Mutual Funds", Headers: headers, Rows: out, Totals: totals}
}

func buildStocks(rows []models.Stock, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Stock) string { return r.Member })
	all := member == ""
	headers := []string{"Stock", "Ticker"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Qty", "Avg Price", "Last Price", "Invested", "Current Value", "Gain")

	var out [][]string
	var totInv, totCur float64
	for _, r := range rows {
		inv, cur := r.Qty*r.AvgPrice, r.Qty*r.LastPrice
		row := []string{r.Name, r.Ticker}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatNum(r.Qty), FormatINR(r.AvgPrice), FormatINR(r.LastPrice), FormatINR(inv), FormatINR(cur), FormatINR(cur-inv))
		out = append(out, row)
		totInv += inv
		totCur += cur
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-3], totals[n-2], totals[n-1] = FormatINR(totInv), FormatINR(totCur), FormatINR(totCur-totInv)
	return Table{Title: "Stocks", Headers: headers, Rows: out, Totals: totals}
}

func buildMetals(rows []models.Metal, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Metal) string { return r.Member })
	all := member == ""
	headers := []string{"Metal"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Purchased", "Grams", "Buy Rate/g", "Today Rate/g", "Invested", "Current Value", "Gain")

	var out [][]string
	var totInv, totCur float64
	for _, r := range rows {
		inv, cur := r.Grams*r.BuyRate, r.Grams*r.TodayPrice
		row := []string{r.Type}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatDate(r.DatePurchased), FormatNum(r.Grams), FormatINR(r.BuyRate), FormatINR(r.TodayPrice), FormatINR(inv), FormatINR(cur), FormatINR(cur-inv))
		out = append(out, row)
		totInv += inv
		totCur += cur
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-3], totals[n-2], totals[n-1] = FormatINR(totInv), FormatINR(totCur), FormatINR(totCur-totInv)
	return Table{Title: "Gold & Silver", Headers: headers, Rows: out, Totals: totals}
}

func buildFixed(rows []models.Fixed, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Fixed) string { return r.Member })
	all := member == ""
	headers := []string{"Deposit", "Kind"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Opened", "Matures", "Rate %", "Principal", "Current Value", "Gain")

	var out [][]string
	var totPrin, totCur float64
	for _, r := range rows {
		row := []string{r.Name, r.Kind}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatDate(r.Opened), FormatDate(r.Matures), FormatNum(r.Rate), FormatINR(r.Principal), FormatINR(r.CurrentValue), FormatINR(r.CurrentValue-r.Principal))
		out = append(out, row)
		totPrin += r.Principal
		totCur += r.CurrentValue
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-3], totals[n-2], totals[n-1] = FormatINR(totPrin), FormatINR(totCur), FormatINR(totCur-totPrin)
	return Table{Title: "Fixed Deposits", Headers: headers, Rows: out, Totals: totals}
}

func buildInsurance(rows []models.Insurance, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Insurance) string { return r.Member })
	all := member == ""
	headers := []string{"Plan", "Type"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Frequency", "Maturity Year", "Premium", "Paid So Far", "Cover", "Value")

	var out [][]string
	var totPremium, totPaid, totCover, totValue float64
	for _, r := range rows {
		row := []string{r.Name, r.Type}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		maturity := ""
		if r.Maturity > 0 {
			maturity = strconv.Itoa(r.Maturity)
		}
		row = append(row, r.Freq, maturity, FormatINR(r.Premium), FormatINR(r.Paid), FormatINR(r.Cover), FormatINR(r.Value))
		out = append(out, row)
		totPremium += r.Premium
		totPaid += r.Paid
		totCover += r.Cover
		totValue += r.Value
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-4], totals[n-3], totals[n-2], totals[n-1] = FormatINR(totPremium), FormatINR(totPaid), FormatINR(totCover), FormatINR(totValue)
	return Table{Title: "Insurance", Headers: headers, Rows: out, Totals: totals}
}

func buildNPS(rows []models.NPS, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.NPS) string { return r.Member })
	all := member == ""
	headers := []string{"Scheme", "Fund Manager"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "PRAN", "Tier", "Asset Class", "Units", "NAV", "Invested", "Current Value", "Gain")

	var out [][]string
	var totInv, totCur float64
	for _, r := range rows {
		cur := r.Units * r.NAV
		row := []string{r.Scheme, r.FundManager}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, r.PRAN, r.Tier, r.AssetClass, FormatNum(r.Units), FormatINR(r.NAV), FormatINR(r.Invested), FormatINR(cur), FormatINR(cur-r.Invested))
		out = append(out, row)
		totInv += r.Invested
		totCur += cur
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-3], totals[n-2], totals[n-1] = FormatINR(totInv), FormatINR(totCur), FormatINR(totCur-totInv)
	return Table{Title: "NPS", Headers: headers, Rows: out, Totals: totals}
}

func buildLoans(rows []models.Loan, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Loan) string { return r.Member })
	all := member == ""
	headers := []string{"Lender", "Type"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Started", "Rate %", "Principal", "Outstanding", "EMI")

	var out [][]string
	var totPrin, totOut, totEMI float64
	for _, r := range rows {
		row := []string{r.Lender, r.Type}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatDate(r.Started), FormatNum(r.Rate), FormatINR(r.Principal), FormatINR(r.Outstanding), FormatINR(r.EMI))
		out = append(out, row)
		totPrin += r.Principal
		totOut += r.Outstanding
		totEMI += r.EMI
	}
	totals := make([]string, len(headers))
	n := len(headers)
	totals[0] = "Total"
	totals[n-3], totals[n-2], totals[n-1] = FormatINR(totPrin), FormatINR(totOut), FormatINR(totEMI)
	return Table{Title: "Loans / Liabilities", Headers: headers, Rows: out, Totals: totals}
}

func buildSIPs(rows []models.SIP, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.SIP) string { return r.Member })
	all := member == ""
	headers := []string{"Fund", "Kind"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Platform", "Debit Day", "Status", "Amount")

	var out [][]string
	var totAmt float64
	for _, r := range rows {
		row := []string{r.Fund, r.Kind}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, r.Platform, strconv.Itoa(r.Day), r.Status, FormatINR(r.Amount))
		out = append(out, row)
		if r.Status == "active" {
			totAmt += r.Amount
		}
	}
	totals := make([]string, len(headers))
	totals[0] = "Total (active)"
	totals[len(headers)-1] = FormatINR(totAmt)
	return Table{Title: "SIPs", Headers: headers, Rows: out, Totals: totals}
}

func buildLumpsums(rows []models.Lumpsum, member string, members []models.Member) Table {
	rows = filterByMember(rows, member, func(r models.Lumpsum) string { return r.Member })
	all := member == ""
	headers := []string{"Fund"}
	if all {
		headers = append(headers, "Owner")
	}
	headers = append(headers, "Date", "Amount")

	var out [][]string
	var totAmt float64
	for _, r := range rows {
		row := []string{r.Fund}
		if all {
			row = append(row, memberName(members, r.Member))
		}
		row = append(row, FormatDate(r.Date), FormatINR(r.Amount))
		out = append(out, row)
		totAmt += r.Amount
	}
	totals := make([]string, len(headers))
	totals[0] = "Total"
	totals[len(headers)-1] = FormatINR(totAmt)
	return Table{Title: "Lumpsum Top-ups", Headers: headers, Rows: out, Totals: totals}
}

func buildIncome(rows []models.Income) Table {
	headers := []string{"Period", "Source", "Type", "Gross", "Net", "Deductions"}
	var out [][]string
	var totGross, totNet, totDed float64
	for _, r := range rows {
		ded := r.PFDeduction + r.TaxDeduction + r.OtherDeductions
		out = append(out, []string{r.Period, r.Source, r.Type, FormatINR(r.Gross), FormatINR(r.Net), FormatINR(ded)})
		totGross += r.Gross
		totNet += r.Net
		totDed += ded
	}
	totals := []string{"Total", "", "", FormatINR(totGross), FormatINR(totNet), FormatINR(totDed)}
	return Table{Title: "Income", Headers: headers, Rows: out, Totals: totals}
}
