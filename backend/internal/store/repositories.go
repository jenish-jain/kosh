package store

import "kosh/internal/models"

// Repositories is a typed container holding one Repository per sheet tab.
type Repositories struct {
	Members   *Repository[models.Member]
	MF        *Repository[models.MFRow]
	Stocks    *Repository[models.Stock]
	Metals    *Repository[models.Metal]
	Fixed     *Repository[models.Fixed]
	Insurance *Repository[models.Insurance]
	Loans     *Repository[models.Loan]
	NPS       *Repository[models.NPS]
	SIPs      *Repository[models.SIP]
	Lumpsums  *Repository[models.Lumpsum]
	History   *Repository[models.History]
	Income    *Repository[models.Income]

	TaxRecommendations *Repository[models.TaxRecommendation]

	// Registry enables generic mutation dispatch by sheet name.
	Registry map[string]AnyRepository
}

// NewRepositories constructs all repositories backed by api.
func NewRepositories(api SheetsAPI) *Repositories {
	r := &Repositories{
		Members:   NewRepository[models.Member](api, "Members"),
		MF:        NewRepository[models.MFRow](api, "MF"),
		Stocks:    NewRepository[models.Stock](api, "Stocks"),
		Metals:    NewRepository[models.Metal](api, "Metals"),
		Fixed:     NewRepository[models.Fixed](api, "Fixed"),
		Insurance: NewRepository[models.Insurance](api, "Insurance"),
		Loans:     NewRepository[models.Loan](api, "Loans"),
		NPS:       NewRepository[models.NPS](api, "NPS"),
		SIPs:      NewRepository[models.SIP](api, "SIPs"),
		Lumpsums:  NewRepository[models.Lumpsum](api, "Lumpsums"),
		History:   NewRepository[models.History](api, "History"),
		Income:    NewRepository[models.Income](api, "Income"),

		TaxRecommendations: NewRepository[models.TaxRecommendation](api, "TaxRecommendations"),
	}
	r.Registry = map[string]AnyRepository{
		"Members":   r.Members,
		"MF":        r.MF,
		"Stocks":    r.Stocks,
		"Metals":    r.Metals,
		"Fixed":     r.Fixed,
		"Insurance": r.Insurance,
		"Loans":     r.Loans,
		"NPS":       r.NPS,
		"SIPs":      r.SIPs,
		"Lumpsums":  r.Lumpsums,
		"History":   r.History,
		"Income":    r.Income,

		"TaxRecommendations": r.TaxRecommendations,
	}
	return r
}
