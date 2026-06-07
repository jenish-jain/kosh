package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	sh "kosh/sheets"
)

// Data is the full payload returned by GET /api/data.
type Data struct {
	Members   []Member   `json:"members"`
	MF        []MFRow    `json:"mf"`
	Stocks    []Stock    `json:"stocks"`
	Metals    []Metal    `json:"metals"`
	Fixed     []Fixed    `json:"fixed"`
	Insurance []Insurance `json:"insurance"`
	SIPs      []SIP      `json:"sips"`
	Lumpsums  []Lumpsum  `json:"lumpsums"`
	History   []History  `json:"history"`
	Config    Config     `json:"config"`
}

type Member struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Relation string `json:"relation"`
	Slab     int    `json:"slab"`
	Color    string `json:"color"`
}

type MFRow struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Plan     string  `json:"plan"`
	Platform string  `json:"platform"`
	Member   string  `json:"member"`
	Invested float64 `json:"invested"`
	Current  float64 `json:"current"`
	SIP      float64 `json:"sip"`
	Notes    string  `json:"notes"`
}

type Stock struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Ticker    string  `json:"ticker"`
	Qty       float64 `json:"qty"`
	AvgPrice  float64 `json:"avg_price"`
	LastPrice float64 `json:"last_price"`
	Member    string  `json:"member"`
}

type Metal struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	DatePurchased string  `json:"date_purchased"`
	Grams         float64 `json:"grams"`
	BuyRate       float64 `json:"buy_rate"`
	TodayPrice    float64 `json:"today_price"`
	Place         string  `json:"place"`
	Member        string  `json:"member"`
}

type Fixed struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Name         string  `json:"name"`
	Member       string  `json:"member"`
	Principal    float64 `json:"principal"`
	Rate         float64 `json:"rate"`
	CurrentValue float64 `json:"current_value"`
	Opened       string  `json:"opened"`
	Matures      string  `json:"matures"`
	Monthly      float64 `json:"monthly"`
}

type Insurance struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Member   string  `json:"member"`
	Premium  float64 `json:"premium"`
	Freq     string  `json:"freq"`
	Paid     float64 `json:"paid"`
	Value    float64 `json:"value"`
	Cover    float64 `json:"cover"`
	Maturity int     `json:"maturity"`
	DueDate  string  `json:"due_date"` // anchor date (YYYY-MM-DD); month/day reused each cycle per Freq
}

type SIP struct {
	ID        string  `json:"id"`
	Fund      string  `json:"fund"`
	Member    string  `json:"member"`
	Amount    float64 `json:"amount"`
	Day       int     `json:"day"`
	Status    string  `json:"status"`
	StartDate string  `json:"start_date"`
	Platform  string  `json:"platform"`
	Kind      string  `json:"kind"`
}

type Lumpsum struct {
	ID     string  `json:"id"`
	Fund   string  `json:"fund"`
	Member string  `json:"member"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}

type History struct {
	Month string  `json:"month"`
	Value float64 `json:"value"`
}

type Config struct {
	GrossIncome          float64 `json:"gross_income"`
	Regime               string  `json:"regime"`
	CapitalGainsThisYear float64 `json:"capital_gains_this_year"`
	ShiftedToParents     float64 `json:"shifted_to_parents"`
	SurchargeFrom        float64 `json:"surcharge_from"`
	NextSurchargeAt      float64 `json:"next_surcharge_at"`
	ParentsCapacityMom   float64 `json:"parents_capacity_mom"`
	ParentsCapacityDad   float64 `json:"parents_capacity_dad"`
	Deduction80C         float64 `json:"deduction_80c"`
	Deduction80D         float64 `json:"deduction_80d"`
	OtherDeductions      float64 `json:"other_deductions"`
	SavedByFiling        float64 `json:"saved_by_filing"`
}

// Handler wraps a Sheets client (may be nil in dev mode).
type Handler struct {
	client  *sh.Client
	devData *Data
}

func NewHandler(client *sh.Client) *Handler {
	h := &Handler{client: client}
	if client == nil {
		h.devData = loadDevData()
	}
	return h
}

func loadDevData() *Data {
	b, err := os.ReadFile("dev_data.json")
	if err != nil {
		fmt.Println("[dev] dev_data.json not found, returning empty data")
		return &Data{}
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		fmt.Println("[dev] failed to parse dev_data.json:", err)
		return &Data{}
	}
	fmt.Println("[dev] loaded dev_data.json")
	return &d
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	var d *Data
	if h.devData != nil {
		// Shallow-copy so we don't mutate the cached struct across requests.
		cp := *h.devData
		fixed := make([]Fixed, len(cp.Fixed))
		copy(fixed, cp.Fixed)
		cp.Fixed = fixed
		d = &cp
	} else {
		var err error
		d, err = h.fetchFromSheets()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	computeFixedValues(d)
	h.maybeSnapshotHistory(d)
	writeJSON(w, d)
}

// maybeSnapshotHistory appends a History row for the current month if one
// isn't already there, so the net-worth sparkline stays current without
// manual monthly upkeep. No-op in dev mode or if net worth is still zero
// (e.g. sheets not yet populated).
func (h *Handler) maybeSnapshotHistory(d *Data) {
	if h.client == nil {
		return
	}
	month := time.Now().Format("Jan 06")
	for _, row := range d.History {
		if row.Month == month {
			return
		}
	}
	total := netWorthTotal(d)
	if total == 0 {
		return
	}
	if _, err := h.client.AppendRow("History", []interface{}{month, fmt.Sprintf("%.0f", total)}); err != nil {
		log.Printf("⚠ History snapshot failed: %v", err)
		return
	}
	d.History = append(d.History, History{Month: month, Value: total})
}

// netWorthTotal sums current values across all asset classes — mirrors
// classTotals().total.cur in frontend/src/data/helpers.js.
func netWorthTotal(d *Data) float64 {
	var total float64
	for _, x := range d.MF {
		total += x.Current
	}
	for _, x := range d.Stocks {
		total += x.Qty * x.LastPrice
	}
	for _, x := range d.Metals {
		total += x.Grams * x.TodayPrice
	}
	for _, x := range d.Fixed {
		total += x.CurrentValue
	}
	for _, x := range d.Insurance {
		total += x.Value
	}
	return total
}

func (h *Handler) fetchFromSheets() (*Data, error) {
	d := &Data{}

	// tabLog logs a warning for a tab read failure but does not abort — a missing
	// or unshared tab should not crash the whole app.
	tabLog := func(tab string, e error) {
		log.Printf("⚠  sheet %q unavailable: %v", tab, e)
	}

	if rows, e := h.client.ReadSheet("Members"); e == nil {
		for _, row := range rows[1:] {
			d.Members = append(d.Members, Member{
				ID: sh.ColStr(row, 0), Name: sh.ColStr(row, 1),
				FullName: sh.ColStr(row, 2), Relation: sh.ColStr(row, 3),
				Slab: sh.ColInt(row, 4), Color: sh.ColStr(row, 5),
			})
		}
	} else {
		tabLog("Members", e)
	}

	if rows, e := h.client.ReadSheet("MF"); e == nil {
		for _, row := range rows[1:] {
			d.MF = append(d.MF, MFRow{
				ID: sh.ColStr(row, 0), Name: sh.ColStr(row, 1), Plan: sh.ColStr(row, 2),
				Platform: sh.ColStr(row, 3), Member: sh.ColStr(row, 4),
				Invested: sh.ColFloat(row, 5), Current: sh.ColFloat(row, 6),
				SIP: sh.ColFloat(row, 7), Notes: sh.ColStr(row, 8),
			})
		}
	} else {
		tabLog("MF", e)
	}

	if rows, e := h.client.ReadSheet("Stocks"); e == nil {
		for _, row := range rows[1:] {
			d.Stocks = append(d.Stocks, Stock{
				ID: sh.ColStr(row, 0), Name: sh.ColStr(row, 1), Ticker: sh.ColStr(row, 2),
				Qty: sh.ColFloat(row, 3), AvgPrice: sh.ColFloat(row, 4),
				LastPrice: sh.ColFloat(row, 5), Member: sh.ColStr(row, 6),
			})
		}
	} else {
		tabLog("Stocks", e)
	}

	if rows, e := h.client.ReadSheet("Metals"); e == nil {
		for _, row := range rows[1:] {
			d.Metals = append(d.Metals, Metal{
				ID: sh.ColStr(row, 0), Type: sh.ColStr(row, 1), DatePurchased: sh.ColStr(row, 2),
				Grams: sh.ColFloat(row, 3), BuyRate: sh.ColFloat(row, 4),
				TodayPrice: sh.ColFloat(row, 5), Place: sh.ColStr(row, 6), Member: sh.ColStr(row, 7),
			})
		}
	} else {
		tabLog("Metals", e)
	}

	if rows, e := h.client.ReadSheet("Fixed"); e == nil {
		for _, row := range rows[1:] {
			// CurrentValue is intentionally not read from column 6 — it is
			// computed at request time by computeFixedValues so it is always
			// accurate without requiring manual sheet updates.
			d.Fixed = append(d.Fixed, Fixed{
				ID: sh.ColStr(row, 0), Kind: sh.ColStr(row, 1), Name: sh.ColStr(row, 2),
				Member: sh.ColStr(row, 3), Principal: sh.ColFloat(row, 4),
				Rate: sh.ColFloat(row, 5),
				Opened: sh.ColStr(row, 7), Matures: sh.ColStr(row, 8), Monthly: sh.ColFloat(row, 9),
			})
		}
	} else {
		tabLog("Fixed", e)
	}

	if rows, e := h.client.ReadSheet("Insurance"); e == nil {
		for _, row := range rows[1:] {
			d.Insurance = append(d.Insurance, Insurance{
				ID: sh.ColStr(row, 0), Name: sh.ColStr(row, 1), Type: sh.ColStr(row, 2),
				Member: sh.ColStr(row, 3), Premium: sh.ColFloat(row, 4), Freq: sh.ColStr(row, 5),
				Paid: sh.ColFloat(row, 6), Value: sh.ColFloat(row, 7),
				Cover: sh.ColFloat(row, 8), Maturity: sh.ColInt(row, 9),
				DueDate: sh.ColStr(row, 10),
			})
		}
	} else {
		tabLog("Insurance", e)
	}

	if rows, e := h.client.ReadSheet("SIPs"); e == nil {
		for _, row := range rows[1:] {
			d.SIPs = append(d.SIPs, SIP{
				ID: sh.ColStr(row, 0), Fund: sh.ColStr(row, 1), Member: sh.ColStr(row, 2),
				Amount: sh.ColFloat(row, 3), Day: sh.ColInt(row, 4),
				Status: sh.ColStr(row, 5), StartDate: sh.ColStr(row, 6), Platform: sh.ColStr(row, 7),
				Kind: sh.ColStr(row, 8),
			})
		}
	} else {
		tabLog("SIPs", e)
	}

	if rows, e := h.client.ReadSheet("Lumpsums"); e == nil {
		for _, row := range rows[1:] {
			d.Lumpsums = append(d.Lumpsums, Lumpsum{
				ID: sh.ColStr(row, 0), Fund: sh.ColStr(row, 1), Member: sh.ColStr(row, 2),
				Amount: sh.ColFloat(row, 3), Date: sh.ColStr(row, 4),
			})
		}
	} else {
		tabLog("Lumpsums", e)
	}

	if rows, e := h.client.ReadSheet("History"); e == nil {
		for _, row := range rows[1:] {
			d.History = append(d.History, History{
				Month: sh.ColStr(row, 0), Value: sh.ColFloat(row, 1),
			})
		}
	} else {
		tabLog("History", e)
	}

	if rows, e := h.client.ReadSheet("Config"); e == nil {
		cfg := map[string]string{}
		for _, row := range rows[1:] {
			if len(row) >= 2 {
				cfg[strings.TrimSpace(sh.ColStr(row, 0))] = sh.ColStr(row, 1)
			}
		}
		d.Config = parseConfig(cfg)
	} else {
		tabLog("Config", e)
	}

	return d, nil
}

func parseConfig(m map[string]string) Config {
	pf := func(k string) float64 {
		v, _ := parseFloat64(m[k])
		return v
	}
	return Config{
		GrossIncome:          pf("gross_income"),
		Regime:               m["regime"],
		CapitalGainsThisYear: pf("capital_gains_this_year"),
		ShiftedToParents:     pf("shifted_to_parents"),
		SurchargeFrom:        pf("surcharge_from"),
		NextSurchargeAt:      pf("next_surcharge_at"),
		ParentsCapacityMom:   pf("parents_capacity_mom"),
		ParentsCapacityDad:   pf("parents_capacity_dad"),
		Deduction80C:         pf("deduction_80c"),
		Deduction80D:         pf("deduction_80d"),
		OtherDeductions:      pf("other_deductions"),
		SavedByFiling:        pf("saved_by_filing"),
	}
}

func parseFloat64(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ── Fixed / RD value computation ─────────────────────────────────────────────
// Values are computed at request time using quarterly compounding (standard for
// Indian banks). The current_value column in the sheet is ignored.

func computeFixedValues(d *Data) {
	for i := range d.Fixed {
		d.Fixed[i].CurrentValue = computeFixedValue(d.Fixed[i])
	}
}

func computeFixedValue(f Fixed) float64 {
	if f.Rate == 0 || f.Opened == "" {
		return f.Principal
	}
	if f.Kind == "RD" {
		return rdCurrentValue(f.Monthly, f.Rate, f.Opened, f.Matures)
	}
	return fdCurrentValue(f.Principal, f.Rate, f.Opened)
}

// fdCurrentValue: P × (1 + r)^t — annual compounding.
func fdCurrentValue(principal, ratePercent float64, opened string) float64 {
	if principal == 0 {
		return 0
	}
	t := yearsSince(opened)
	if t <= 0 {
		return principal
	}
	r := ratePercent / 100.0
	return principal * math.Pow(1+r, t)
}

// rdCurrentValue: each monthly installment compounds annually from its deposit date.
// Installments are capped at the tenure derived from opened→matures.
func rdCurrentValue(monthly, ratePercent float64, opened, matures string) float64 {
	if monthly == 0 {
		return 0
	}
	r := ratePercent / 100.0
	tenure := monthsBetween(opened, matures)
	elapsed := monthsBetween(opened, time.Now().Format("2006-01-02"))
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

func yearsSince(dateStr string) float64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	return time.Since(t).Hours() / (24 * 365.25)
}

func monthsBetween(fromStr, toStr string) int {
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
