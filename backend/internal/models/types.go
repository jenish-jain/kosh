package models

// Data is the full payload returned by GET /api/data.
type Data struct {
	Members   []Member    `json:"members"`
	MF        []MFRow     `json:"mf"`
	Stocks    []Stock     `json:"stocks"`
	Metals    []Metal     `json:"metals"`
	Fixed     []Fixed     `json:"fixed"`
	Insurance []Insurance `json:"insurance"`
	Loans     []Loan      `json:"loans"`
	NPS       []NPS       `json:"nps"`
	SIPs      []SIP       `json:"sips"`
	Lumpsums  []Lumpsum   `json:"lumpsums"`
	History   []History   `json:"history"`
	Config    Config      `json:"config"`
}

type Member struct {
	ID       string `json:"id"        sheet:"id"`
	Name     string `json:"name"      sheet:"name"`
	FullName string `json:"full_name" sheet:"full_name"`
	Relation string `json:"relation"  sheet:"relation"`
	Slab     int    `json:"slab"      sheet:"slab"`
	Color    string `json:"color"     sheet:"color"`
}

type MFRow struct {
	ID       string  `json:"id"       sheet:"id"`
	Name     string  `json:"name"     sheet:"name"`
	Plan     string  `json:"plan"     sheet:"plan"`
	Platform string  `json:"platform" sheet:"platform"`
	Member   string  `json:"member"   sheet:"member"`
	Invested float64 `json:"invested" sheet:"invested"`
	Current  float64 `json:"current"  sheet:"current"`
	SIP      float64 `json:"sip"      sheet:"sip"`
	Notes    string  `json:"notes"    sheet:"notes"`
}

type Stock struct {
	ID        string  `json:"id"         sheet:"id"`
	Name      string  `json:"name"       sheet:"name"`
	Ticker    string  `json:"ticker"     sheet:"ticker"`
	Qty       float64 `json:"qty"        sheet:"qty"`
	AvgPrice  float64 `json:"avg_price"  sheet:"avg_price"`
	LastPrice float64 `json:"last_price" sheet:"last_price"`
	Member    string  `json:"member"     sheet:"member"`
}

type Metal struct {
	ID            string  `json:"id"             sheet:"id"`
	Type          string  `json:"type"           sheet:"type"`
	DatePurchased string  `json:"date_purchased" sheet:"date_purchased"`
	Grams         float64 `json:"grams"          sheet:"grams"`
	BuyRate       float64 `json:"buy_rate"       sheet:"buy_rate"`
	TodayPrice    float64 `json:"today_price"    sheet:"today_price"`
	Place         string  `json:"place"          sheet:"place"`
	Member        string  `json:"member"         sheet:"member"`
}

type Fixed struct {
	ID           string  `json:"id"            sheet:"id"`
	Kind         string  `json:"kind"          sheet:"kind"`
	Name         string  `json:"name"          sheet:"name"`
	Member       string  `json:"member"        sheet:"member"`
	Principal    float64 `json:"principal"     sheet:"principal"`
	Rate         float64 `json:"rate"          sheet:"rate"`
	CurrentValue float64 `json:"current_value" sheet:"current_value"`
	Opened       string  `json:"opened"        sheet:"opened"`
	Matures      string  `json:"matures"       sheet:"matures"`
	Monthly      float64 `json:"monthly"       sheet:"monthly"`
	DocLink      string  `json:"doc_link"      sheet:"doc_link"`
}

type Insurance struct {
	ID       string  `json:"id"       sheet:"id"`
	Name     string  `json:"name"     sheet:"name"`
	Type     string  `json:"type"     sheet:"type"`
	Member   string  `json:"member"   sheet:"member"`
	Premium  float64 `json:"premium"  sheet:"premium"`
	Freq     string  `json:"freq"     sheet:"freq"`
	Paid     float64 `json:"paid"     sheet:"paid"`
	Value    float64 `json:"value"    sheet:"value"`
	Cover    float64 `json:"cover"    sheet:"cover"`
	Maturity int     `json:"maturity" sheet:"maturity"`
	DueDate  string  `json:"due_date" sheet:"due_date"` // anchor date (YYYY-MM-DD); month/day reused each cycle per Freq
	DocLink  string  `json:"doc_link" sheet:"doc_link"`
}

type Loan struct {
	ID           string  `json:"id"            sheet:"id"`
	Lender       string  `json:"lender"        sheet:"lender"`
	Type         string  `json:"type"          sheet:"type"`
	Member       string  `json:"member"        sheet:"member"`
	Principal    float64 `json:"principal"     sheet:"principal"`
	Outstanding  float64 `json:"outstanding"   sheet:"outstanding"`
	Rate         float64 `json:"rate"          sheet:"rate"`
	EMI          float64 `json:"emi"           sheet:"emi"`
	EMIDay       int     `json:"emi_day"       sheet:"emi_day"`
	Started      string  `json:"started"       sheet:"started"`
	TenureMonths int     `json:"tenure_months" sheet:"tenure_months"`
}

type SIP struct {
	ID        string  `json:"id"         sheet:"id"`
	Fund      string  `json:"fund"       sheet:"fund"`
	Member    string  `json:"member"     sheet:"member"`
	Amount    float64 `json:"amount"     sheet:"amount"`
	Day       int     `json:"day"        sheet:"day"`
	Status    string  `json:"status"     sheet:"status"`
	StartDate string  `json:"start_date" sheet:"start_date"`
	Platform  string  `json:"platform"   sheet:"platform"`
	Kind      string  `json:"kind"       sheet:"kind"`
}

type NPS struct {
	ID          string  `json:"id"           sheet:"id"`
	PRAN        string  `json:"pran"         sheet:"pran"`
	Member      string  `json:"member"       sheet:"member"`
	Tier        string  `json:"tier"         sheet:"tier"`
	AssetClass  string  `json:"asset_class"  sheet:"asset_class"`
	Scheme      string  `json:"scheme"       sheet:"scheme"`
	FundManager string  `json:"fund_manager" sheet:"fund_manager"`
	Units       float64 `json:"units"        sheet:"units"`
	NAV         float64 `json:"nav"          sheet:"nav"`
	Invested    float64 `json:"invested"     sheet:"invested"`
}

type Lumpsum struct {
	ID     string  `json:"id"     sheet:"id"`
	Fund   string  `json:"fund"   sheet:"fund"`
	Member string  `json:"member" sheet:"member"`
	Amount float64 `json:"amount" sheet:"amount"`
	Date   string  `json:"date"   sheet:"date"`
}

type History struct {
	Month string  `json:"month" sheet:"month"`
	Value float64 `json:"value" sheet:"value"`
}

// Config is read via parseConfig from a key-value sheet — no sheet tags needed.
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
