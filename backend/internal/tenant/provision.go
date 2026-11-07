package tenant

import (
	"fmt"
	"log"

	"kosh/internal/models"
	"kosh/internal/store"
	sh "kosh/sheets"
)

// Tabs lists every tab a Kosh spreadsheet needs, with its header columns —
// derived from the repositories, the single source of truth.
func Tabs(r *store.Repositories) []sh.TabDef {
	tabs := make([]sh.TabDef, 0, 17)
	for _, repo := range []store.AnyRepository{
		r.Members, r.MF, r.Stocks, r.Metals, r.Fixed, r.Insurance, r.Loans, r.NPS,
		r.SIPs, r.Lumpsums, r.History, r.Income, r.TaxRecommendations, r.TaxRules,
		r.Shares, r.ShareAccess,
	} {
		tabs = append(tabs, sh.TabDef{Name: repo.Sheet(), Columns: repo.Columns()})
	}
	return append(tabs, sh.TabDef{Name: "Config", Columns: []string{"key", "value"}})
}

// tabEnsurer is implemented by *sheets.Client.
type tabEnsurer interface {
	EnsureTabs(tabs []sh.TabDef) error
}

// Provision makes t's spreadsheet ready to use: creates any missing tabs
// with their headers, then seeds the default tax rule sets if none exist.
// Safe to run repeatedly — on every self-host startup, and once when a
// hosted household is created.
func Provision(t *Tenant) error {
	if e, ok := t.API.(tabEnsurer); ok {
		if err := e.EnsureTabs(Tabs(t.Repos)); err != nil {
			return fmt.Errorf("creating tabs: %w", err)
		}
	}
	if seedTaxRules(t.Repos.TaxRules) {
		log.Println("✓ Seeded default TaxRules (old + new regime)")
	}
	return nil
}

// seedTaxRules inserts the two bundled default TaxRuleSet rows (old + new
// regime, status "active") if the TaxRules sheet is empty — EnsureTabs only
// writes a header row into an empty tab, it never seeds data. Returns true
// if it seeded anything.
func seedTaxRules(repo *store.Repository[models.TaxRuleSet]) bool {
	existing, err := repo.All()
	if err != nil {
		log.Printf("⚠ checking TaxRules for seeding: %v", err)
		return false
	}
	if len(existing) > 0 {
		return false
	}
	for _, regime := range []string{"old", "new"} {
		if _, err := repo.Add(models.DefaultTaxRuleSet(regime)); err != nil {
			log.Printf("⚠ seeding TaxRules(%s): %v", regime, err)
		}
	}
	return true
}
