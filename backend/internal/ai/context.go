package ai

import (
	"fmt"
	"strings"
	"time"

	"kosh/internal/models"
)

// BuildSystemPrompt returns the system prompt with the user's full financial
// data embedded. The LLM receives this on every request so it can answer
// questions about the user's actual portfolio.
func BuildSystemPrompt(d *models.Data) string {
	var b strings.Builder
	today := time.Now().Format("2 January 2006")

	// ── Compute summary totals ───────────────────────────────────────────────
	var mfCur, mfInv float64
	for _, r := range d.MF {
		mfCur += r.Current
		mfInv += r.Invested
	}

	var stocksCur float64
	for _, r := range d.Stocks {
		stocksCur += r.Qty * r.LastPrice
	}

	var metalsCur float64
	for _, r := range d.Metals {
		metalsCur += r.Grams * r.TodayPrice
	}

	var fixedCur float64
	for _, r := range d.Fixed {
		fixedCur += r.CurrentValue
	}

	var npsCur, npsInv float64
	for _, r := range d.NPS {
		npsCur += r.Units * r.NAV
		npsInv += r.Invested
	}

	var insuranceCover, insurancePremiumAnnual float64
	for _, r := range d.Insurance {
		insuranceCover += r.Cover
		switch r.Freq {
		case "monthly":
			insurancePremiumAnnual += r.Premium * 12
		case "quarterly":
			insurancePremiumAnnual += r.Premium * 4
		default: // annual
			insurancePremiumAnnual += r.Premium
		}
	}

	var loanOutstanding, loanEMI float64
	for _, r := range d.Loans {
		loanOutstanding += r.Outstanding
		loanEMI += r.EMI
	}

	var sipMonthly float64
	activeSIPs := 0
	for _, s := range d.SIPs {
		if s.Status == "active" {
			sipMonthly += s.Amount
			activeSIPs++
		}
	}

	totalAssets := mfCur + stocksCur + metalsCur + fixedCur + npsCur
	netWorth := totalAssets - loanOutstanding

	// ── System prompt ────────────────────────────────────────────────────────
	fmt.Fprintf(&b, `You are a personal financial advisor for this user. Today is %s.

You have complete access to their financial data — all figures are in Indian Rupees (INR / ₹). Use the real numbers below to give specific, personalised answers. Show your calculations when projecting future values. State assumptions clearly (e.g. assumed annual return of 12%%, inflation of 6%%).

=== FINANCIAL SNAPSHOT ===

NET WORTH:        ₹%.0f
Total assets:     ₹%.0f
Total loans:      ₹%.0f

INVESTMENTS:
`, today, netWorth, totalAssets, loanOutstanding)

	if mfCur > 0 {
		fmt.Fprintf(&b, "  Mutual Funds:   ₹%.0f current  (invested ₹%.0f, gain ₹%.0f)\n", mfCur, mfInv, mfCur-mfInv)
	}
	if stocksCur > 0 {
		fmt.Fprintf(&b, "  Stocks:         ₹%.0f current\n", stocksCur)
	}
	if fixedCur > 0 {
		fmt.Fprintf(&b, "  FD / RD:        ₹%.0f\n", fixedCur)
	}
	if npsCur > 0 {
		fmt.Fprintf(&b, "  NPS:            ₹%.0f current  (invested ₹%.0f)\n", npsCur, npsInv)
	}
	if metalsCur > 0 {
		fmt.Fprintf(&b, "  Gold & Silver:  ₹%.0f\n", metalsCur)
	}

	fmt.Fprintf(&b, `
MONTHLY CASH FLOW:
  SIP investments: ₹%.0f/month  (%d active SIPs)
  Loan EMIs:       ₹%.0f/month
  Total outflow:   ₹%.0f/month

`, sipMonthly, activeSIPs, loanEMI, sipMonthly+loanEMI)

	if insuranceCover > 0 {
		fmt.Fprintf(&b, "INSURANCE: ₹%.0f total sum assured, ₹%.0f/year in premiums across %d plans\n\n",
			insuranceCover, insurancePremiumAnnual, len(d.Insurance))
	}

	// Family members
	if len(d.Members) > 0 {
		b.WriteString("FAMILY MEMBERS:\n")
		for _, m := range d.Members {
			fmt.Fprintf(&b, "  %s — %s\n", m.Name, m.Relation)
		}
		b.WriteString("\n")
	}

	// ── Detailed holdings ────────────────────────────────────────────────────
	b.WriteString("=== DETAILED HOLDINGS ===\n\n")

	if len(d.MF) > 0 {
		b.WriteString("MUTUAL FUNDS:\n")
		for _, r := range d.MF {
			line := fmt.Sprintf("  %-40s  invested ₹%.0f  current ₹%.0f", r.Name, r.Invested, r.Current)
			if r.SIP > 0 {
				line += fmt.Sprintf("  SIP ₹%.0f/mo", r.SIP)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if len(d.Stocks) > 0 {
		b.WriteString("STOCKS:\n")
		for _, r := range d.Stocks {
			cur := r.Qty * r.LastPrice
			fmt.Fprintf(&b, "  %-30s  %.2f units × ₹%.2f = ₹%.0f\n", r.Name, r.Qty, r.LastPrice, cur)
		}
		b.WriteString("\n")
	}

	if len(d.Fixed) > 0 {
		b.WriteString("FIXED DEPOSITS / RDs:\n")
		for _, r := range d.Fixed {
			fmt.Fprintf(&b, "  %-30s  %s  ₹%.0f @ %.1f%%  matures %s  value ₹%.0f\n",
				r.Name, r.Kind, r.Principal, r.Rate, r.Matures, r.CurrentValue)
		}
		b.WriteString("\n")
	}

	if len(d.NPS) > 0 {
		b.WriteString("NPS:\n")
		for _, r := range d.NPS {
			cur := r.Units * r.NAV
			fmt.Fprintf(&b, "  PRAN %-15s  %s %s  %.3f units × NAV ₹%.2f = ₹%.0f  (invested ₹%.0f)\n",
				r.PRAN, r.AssetClass, r.FundManager, r.Units, r.NAV, cur, r.Invested)
		}
		b.WriteString("\n")
	}

	if len(d.Metals) > 0 {
		b.WriteString("GOLD & SILVER:\n")
		for _, r := range d.Metals {
			cur := r.Grams * r.TodayPrice
			fmt.Fprintf(&b, "  %s  %.2fg  buy rate ₹%.0f/g  today ₹%.0f/g  value ₹%.0f\n",
				r.Type, r.Grams, r.BuyRate, r.TodayPrice, cur)
		}
		b.WriteString("\n")
	}

	if len(d.Insurance) > 0 {
		b.WriteString("INSURANCE PLANS:\n")
		for _, r := range d.Insurance {
			fmt.Fprintf(&b, "  %-35s  %s  cover ₹%.0f  premium ₹%.0f/%s\n",
				r.Name, r.Type, r.Cover, r.Premium, r.Freq)
		}
		b.WriteString("\n")
	}

	if len(d.Loans) > 0 {
		b.WriteString("LOANS:\n")
		for _, r := range d.Loans {
			fmt.Fprintf(&b, "  %-30s  %s  borrowed ₹%.0f  outstanding ₹%.0f  EMI ₹%.0f/mo  %.1f%%\n",
				r.Lender, r.Type, r.Principal, r.Outstanding, r.EMI, r.Rate)
		}
		b.WriteString("\n")
	}

	b.WriteString(`=== END OF DATA ===

Answer questions about financial planning, milestone budgeting (vacations, weddings, home purchase, education, retirement), investment strategy, loan management, tax planning, and insurance coverage. Be specific and practical — reference actual numbers from the data above. When projecting, state your assumption (return rate, inflation, etc.) and show the working.`)

	return b.String()
}
