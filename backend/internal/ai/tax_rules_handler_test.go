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

func doApproveReject(h *ai.TaxRulesHandler, id, action string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/tax/rules/"+id+"/"+action, nil)
	rec := httptest.NewRecorder()
	h.Route(rec, req)
	return rec
}

func TestTaxRulesHandler_Approve_ActivatesAndSupersedesPriorActiveRow(t *testing.T) {
	h, repos := newRulesTestHandler(t, false, false)

	oldActive := models.TaxRuleSet{ID: "old-active", FY: "FY 2025-26", Regime: "new", Status: "active", RulesJSON: "{}"}
	pending := models.TaxRuleSet{ID: "new-pending", FY: "FY 2026-27", Regime: "new", Status: "pending", RulesJSON: "{}"}
	if _, err := repos.TaxRules.Add(oldActive); err != nil {
		t.Fatalf("seeding old-active: %v", err)
	}
	if _, err := repos.TaxRules.Add(pending); err != nil {
		t.Fatalf("seeding pending: %v", err)
	}

	rec := doApproveReject(h, "new-pending", "approve")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rows, err := repos.TaxRules.All()
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	byID := map[string]models.TaxRuleSet{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	if got := byID["new-pending"].Status; got != "active" {
		t.Errorf("new-pending.Status = %q, want active", got)
	}
	if byID["new-pending"].ActivatedDate == "" {
		t.Errorf("expected ActivatedDate to be set on approval")
	}
	if got := byID["old-active"].Status; got != "superseded" {
		t.Errorf("old-active.Status = %q, want superseded", got)
	}
}

func TestTaxRulesHandler_Approve_OnlyPendingCanBeApproved(t *testing.T) {
	h, repos := newRulesTestHandler(t, false, false)
	repos.TaxRules.Add(models.TaxRuleSet{ID: "already-active", FY: "FY 2026-27", Regime: "old", Status: "active", RulesJSON: "{}"})

	rec := doApproveReject(h, "already-active", "approve")
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestTaxRulesHandler_Reject_FlipsToRejected(t *testing.T) {
	h, repos := newRulesTestHandler(t, false, false)
	repos.TaxRules.Add(models.TaxRuleSet{ID: "p1", FY: "FY 2026-27", Regime: "old", Status: "pending", RulesJSON: "{}"})

	rec := doApproveReject(h, "p1", "reject")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rows, _ := repos.TaxRules.All()
	if len(rows) != 1 || rows[0].Status != "rejected" {
		t.Errorf("expected p1 to be rejected, got %+v", rows)
	}
}

func TestTaxRulesHandler_ApproveReject_UnknownIDNotFound(t *testing.T) {
	h, _ := newRulesTestHandler(t, false, false)
	rec := doApproveReject(h, "does-not-exist", "approve")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
