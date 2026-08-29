package ai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kosh/internal/ai"
	"kosh/internal/models"
	"kosh/internal/store"
)

func newRulesTestHandler(t *testing.T, isDemo, isSample bool) (*ai.TaxRulesHandler, *store.Repositories) {
	t.Helper()
	fake := store.NewFakeSheetsAPI()
	fake.Data["TaxRules"] = [][]interface{}{
		{"id", "fy", "regime", "status", "source", "generated_date", "activated_date", "rules_json", "notes"},
	}
	repos := store.NewRepositories(fake)
	h := ai.NewTaxRulesHandler(repos,
		func(r *http.Request) bool { return isDemo },
		func(r *http.Request) bool { return isSample },
	)
	return h, repos
}

func doPropose(h *ai.TaxRulesHandler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/tax/rules/propose", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Propose(rec, req)
	return rec
}

const validProposeBody = `{
  "fy": "FY 2026-27",
  "regime": "new",
  "source": "Union Budget 2026 Memorandum",
  "notes": "test",
  "rules": {
    "schemaVersion": 1,
    "slabs": [{"upto": 400000, "rate": 0}, {"rate": 0.30}],
    "stdDeduction": 87500,
    "rebateThreshold": 1200000,
    "rebateAmount": 999999999,
    "surcharge": [{"above": 5000000, "rate": 0.10}],
    "cessRate": 0.04,
    "deductionCaps": {"section80C": 150000, "section80DSelf": 25000, "section80DSenior": 50000, "nps80CCD1B": 50000}
  }
}`

func TestTaxRulesHandler_Propose_PersistsAsPending(t *testing.T) {
	h, repos := newRulesTestHandler(t, false /* isDemo */, false /* isSample */)

	rec := doPropose(h, validProposeBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	stored, err := repos.TaxRules.All()
	if err != nil {
		t.Fatalf("reading back TaxRules: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("got %d TaxRules rows, want 1", len(stored))
	}
	row := stored[0]
	if row.Status != "pending" {
		t.Errorf("Status = %q, want %q", row.Status, "pending")
	}
	if row.ID == "" || row.GeneratedDate == "" {
		t.Errorf("expected server-generated ID/GeneratedDate, got %+v", row)
	}
	if row.FY != "FY 2026-27" || row.Regime != "new" {
		t.Errorf("unexpected FY/Regime: %+v", row)
	}

	var rules models.TaxRules
	if err := json.Unmarshal([]byte(row.RulesJSON), &rules); err != nil {
		t.Fatalf("rules_json did not round-trip: %v", err)
	}
	if rules.StdDeduction != 87500 {
		t.Errorf("StdDeduction = %v, want 87500", rules.StdDeduction)
	}
}

func TestTaxRulesHandler_Propose_DemoSessionDoesNotPersist(t *testing.T) {
	h, repos := newRulesTestHandler(t, true /* isDemo */, true)

	rec := doPropose(h, validProposeBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	json.Unmarshal(rec.Body.Bytes(), &body)
	if _, ok := body["note"]; !ok {
		t.Errorf("expected a demo-mode note, got %v", body)
	}

	stored, _ := repos.TaxRules.All()
	if len(stored) != 0 {
		t.Errorf("demo session must not persist — got %d stored rows", len(stored))
	}
}

func TestTaxRulesHandler_Propose_SampleModeSkipsPersistence(t *testing.T) {
	h, repos := newRulesTestHandler(t, false, true /* isSample */)

	rec := doPropose(h, validProposeBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	json.Unmarshal(rec.Body.Bytes(), &body)
	if _, ok := body["rule_set"]; !ok {
		t.Errorf("expected the generated rule_set to still be returned in sample mode, got %v", body)
	}

	stored, _ := repos.TaxRules.All()
	if len(stored) != 0 {
		t.Errorf("sample mode must not persist — got %d stored rows", len(stored))
	}
}

func TestTaxRulesHandler_Propose_ClientCannotSetStatus(t *testing.T) {
	// The wire format has no "status" field at all on proposeRequest — sending
	// one must be silently ignored, not honored, no matter what a client sends.
	h, repos := newRulesTestHandler(t, false, false)

	bodyWithStatus := strings.Replace(validProposeBody, `"fy": "FY 2026-27",`, `"fy": "FY 2026-27", "status": "active",`, 1)
	rec := doPropose(h, bodyWithStatus)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	stored, _ := repos.TaxRules.All()
	if len(stored) != 1 || stored[0].Status != "pending" {
		t.Errorf("expected a single pending row regardless of client-supplied status, got %+v", stored)
	}
}

func TestTaxRulesHandler_Propose_RejectsInvalidRegime(t *testing.T) {
	h, _ := newRulesTestHandler(t, false, true)
	body := strings.Replace(validProposeBody, `"regime": "new"`, `"regime": "bogus"`, 1)
	rec := doPropose(h, body)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestTaxRulesHandler_Propose_RejectsMissingFY(t *testing.T) {
	h, _ := newRulesTestHandler(t, false, true)
	body := strings.Replace(validProposeBody, `"fy": "FY 2026-27",`, ``, 1)
	rec := doPropose(h, body)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
