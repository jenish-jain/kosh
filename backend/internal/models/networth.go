package models

// NetWorthTotal sums current asset values minus outstanding loan balances —
// mirrors netWorth() in frontend/src/data/helpers.js (assets − liabilities).
func NetWorthTotal(d *Data) float64 {
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
	for _, x := range d.NPS {
		total += x.Units * x.NAV
	}
	for _, x := range d.Loans {
		total -= x.Outstanding
	}
	return total
}
