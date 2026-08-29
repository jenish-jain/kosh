package ai

import (
	"fmt"
	"strings"

	"kosh/internal/models"
)

// BuildTaxRecommendationPrompt returns the system prompt for the tax-saving
// recommendation generator. It carries the deterministic facts (computed in
// Go from real data — never trusted from the model) plus prior-cycle
// recommendations, so the model reviews what it suggested before instead of
// repeating stale advice. The model only prioritizes, explains trade-offs,
// and drafts recommendation text/amounts from these facts.
func BuildTaxRecommendationPrompt(facts models.TaxFacts, prior []models.TaxRecommendation) string {
	var b strings.Builder

	fmt.Fprintf(&b, `You are a top-tier Indian personal financial and tax advisor. Today is %s, financial year %s. The user has selected the %s tax regime.

Use ONLY the facts below — never invent numbers. Prioritize, explain trade-offs, and draft specific, actionable recommendations grounded in these facts. Recommend both tax-saving levers AND corpus-growth levers (e.g. how much to shift from low-yield FDs into mutual funds), not tax-saving alone.

=== FACTS ===

Gross income:              ₹%.0f
Tax payable (%s regime):   ₹%.0f  (effective rate %.1f%%)

`, facts.Today, facts.FY, facts.Regime, facts.GrossIncome, facts.Regime, facts.TaxPayable, facts.EffRate)

	if facts.Regime == "new" {
		b.WriteString("NOTE: under the new regime, 80C/80D/NPS deductions do NOT reduce tax payable — only recommend them if there's a genuine case for switching to the old regime, or purely for the corpus-growth angle unrelated to tax.\n\n")
	} else {
		fmt.Fprintf(&b, `80C usage:       ₹%.0f / ₹%.0f cap  (₹%.0f room remaining)
  - of which PF (from latest payslip, annualized): ₹%.0f
  - of which life insurance premiums (annualized): ₹%.0f
  - of which ELSS this FY (estimated — see caveats): ₹%.0f
80D usage (health insurance, annualized): ₹%.0f / ₹%.0f cap
NPS 80CCD(1B): lifetime invested ₹%.0f (cumulative, not FY-scoped — see caveats) / ₹%.0f additional cap

`, facts.Deduction80CUsed, facts.Deduction80CCap, facts.Deduction80CCap-facts.Deduction80CUsed,
			facts.PFAnnualized, facts.LifeInsPremium, facts.ELSSThisFYApprox,
			facts.Deduction80DUsed, facts.Deduction80DCap,
			facts.NPSInvestedTotal, facts.NPS80CCD1BCap)
	}

	if len(facts.MFByCategory) > 0 {
		b.WriteString("Mutual fund holdings by category:\n")
		for cat, tot := range facts.MFByCategory {
			fmt.Fprintf(&b, "  %-14s invested ₹%.0f, current ₹%.0f\n", cat, tot.Invested, tot.Current)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "Potential inflows this FY: ₹%.0f taxable (FD/RD interest — income from other sources, NOT a capital gain), ₹%.0f typically exempt (insurance/other maturities)\n", facts.NextFYTotalTaxable, facts.NextFYTotalExempt)
	for _, i := range facts.NextFYTaxableInterest {
		fmt.Fprintf(&b, "  Taxable: %s matures %s, ₹%.0f interest\n", i.Name, i.When, i.Amount)
	}
	for _, i := range facts.NextFYExemptMaturity {
		fmt.Fprintf(&b, "  Exempt:  %s, %s, ₹%.0f\n", i.Name, i.When, i.Amount)
	}
	b.WriteString("\n")

	if len(facts.Caveats) > 0 {
		b.WriteString("=== KNOWN DATA CAVEATS (hedge accordingly in your rationale) ===\n")
		for _, c := range facts.Caveats {
			b.WriteString("  - " + c + "\n")
		}
		b.WriteString("\n")
	}

	if len(prior) > 0 {
		b.WriteString("=== PRIOR RECOMMENDATIONS (last cycle — review before repeating) ===\n")
		for _, p := range prior {
			fmt.Fprintf(&b, "  [%s] id=%s category=%s status=%s: %s (suggested ₹%.0f, potential saving ₹%.0f)\n",
				p.GeneratedDate, p.ID, p.Category, p.Status, p.Headline, p.SuggestedAmount, p.PotentialTaxSaving)
		}
		b.WriteString(`
If the user's current facts show a prior suggestion was acted on (e.g. 80C usage grew, ELSS holdings increased), acknowledge that progress and move to the next priority instead of repeating it. If a prior recommendation is still unaddressed and still the top priority, you may resurface it — set supersedes_id to that row's id so it's tracked as an evolution, not a duplicate. Recommendations already marked "dismissed" should generally not be repeated unless the underlying facts changed materially.

`)
	}

	b.WriteString(`=== OUTPUT FORMAT ===

Return ONLY a JSON array (no markdown fences, no prose outside the array), 3-6 objects, each shaped exactly as:
{"category": "80C Gap" | "NPS 80CCD(1B)" | "FD-to-MF Shift" | "Regime Optimisation" | "General", "headline": "short actionable headline", "suggested_amount": <number, rupees, 0 if not applicable>, "potential_tax_saving": <number, rupees, 0 if not applicable>, "rationale": "2-3 sentences grounded in the facts above, hedging on any caveated figures", "supersedes_id": "<prior recommendation id this evolves, or empty string>"}

Order by priority, most impactful first. Use specific rupee amounts from the facts given — never vague language like "consider investing more."`)

	return b.String()
}
