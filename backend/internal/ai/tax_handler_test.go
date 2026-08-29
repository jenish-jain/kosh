package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kosh/internal/ai"
	"kosh/internal/models"
	"kosh/internal/store"
)

type stubProvider struct {
	reply   string
	err     error
	calls   int
	lastMsg []ai.Message
}

func (s *stubProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	s.calls++
	s.lastMsg = messages
	if s.err != nil {
		return "", s.err
	}
	return s.reply, nil
}

func sampleData() *models.Data {
	return &models.Data{
		Config: models.Config{GrossIncome: 4800000, Regime: "old"},
		Members: []models.Member{
			{ID: "you", Name: "You", Relation: "Self"},
		},
	}
}

func newTestHandler(t *testing.T, provider ai.Provider, isDemo, isSample bool) (*ai.TaxHandler, *store.Repositories) {
	t.Helper()
	fake := store.NewFakeSheetsAPI()
	// Seed the header row FakeSheetsAPI needs to distinguish it from a data
	// row (Repository.All() skips row 0 as the header, same as a real sheet).
	fake.Data["TaxRecommendations"] = [][]interface{}{
		{"id", "generated_date", "fy", "regime", "category", "headline", "suggested_amount", "potential_tax_saving", "rationale", "status", "superseded_by"},
	}
	repos := store.NewRepositories(fake)
	h := ai.NewTaxHandler(
		provider,
		repos,
		func(r *http.Request) *models.Data { return sampleData() },
		func(r *http.Request) bool { return isDemo },
		func(r *http.Request) bool { return isSample },
	)
	return h, repos
}

func doGenerate(h *ai.TaxHandler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/tax/recommendations/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)
	return rec
}

func TestTaxHandler_DemoSessionSkipsProvider(t *testing.T) {
	provider := &stubProvider{reply: `[]`}
	h, _ := newTestHandler(t, provider, true /* isDemo */, true)

	rec := doGenerate(h, `{"regime":"old"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if provider.calls != 0 {
		t.Errorf("provider.calls = %d, want 0 (demo session must not hit the real provider)", provider.calls)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if _, ok := body["note"]; !ok {
		t.Errorf("expected a demo-mode note in the response, got %v", body)
	}
}

func TestTaxHandler_ParsesAndPersists(t *testing.T) {
	reply := `Here you go:
[
  {"category": "80C Gap", "headline": "Top up ELSS", "suggested_amount": 135000, "potential_tax_saving": 40500, "rationale": "You have room left.", "supersedes_id": ""}
]`
	provider := &stubProvider{reply: reply}
	h, repos := newTestHandler(t, provider, false /* isDemo */, false /* isSample: persist */)

	rec := doGenerate(h, `{"regime":"old"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if provider.calls != 1 {
		t.Fatalf("provider.calls = %d, want 1", provider.calls)
	}

	var out struct {
		Recommendations []models.TaxRecommendation `json:"recommendations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(out.Recommendations) != 1 {
		t.Fatalf("got %d recommendations, want 1", len(out.Recommendations))
	}
	got := out.Recommendations[0]
	if got.Headline != "Top up ELSS" || got.SuggestedAmount != 135000 || got.Status != "new" || got.ID == "" {
		t.Errorf("unexpected recommendation: %+v", got)
	}

	// Persisted (isSample=false) — should now be readable back from the repo.
	stored, err := repos.TaxRecommendations.All()
	if err != nil {
		t.Fatalf("reading back TaxRecommendations: %v", err)
	}
	if len(stored) != 1 || stored[0].ID != got.ID {
		t.Errorf("stored rows = %+v, want the generated recommendation persisted", stored)
	}
}

func TestTaxHandler_SampleModeSkipsPersistence(t *testing.T) {
	provider := &stubProvider{reply: `[{"category":"General","headline":"x","suggested_amount":0,"potential_tax_saving":0,"rationale":"y","supersedes_id":""}]`}
	h, repos := newTestHandler(t, provider, false, true /* isSample: dev/demo data */)

	rec := doGenerate(h, `{"regime":"new"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out struct {
		Recommendations []models.TaxRecommendation `json:"recommendations"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Recommendations) != 1 {
		t.Fatalf("expected the generated recommendation still returned in sample mode, got %d", len(out.Recommendations))
	}

	stored, _ := repos.TaxRecommendations.All()
	if len(stored) != 0 {
		t.Errorf("sample mode must not persist — got %d stored rows", len(stored))
	}
}

func TestTaxHandler_ProviderErrorReturnsBadGateway(t *testing.T) {
	provider := &stubProvider{err: context.DeadlineExceeded}
	h, _ := newTestHandler(t, provider, false, true)

	rec := doGenerate(h, `{"regime":"old"}`)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
}

func TestTaxHandler_MalformedJSONReturnsBadGateway(t *testing.T) {
	provider := &stubProvider{reply: "not json at all"}
	h, _ := newTestHandler(t, provider, false, true)

	rec := doGenerate(h, `{"regime":"old"}`)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502, body = %s", rec.Code, rec.Body.String())
	}
}
