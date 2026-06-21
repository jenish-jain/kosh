package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kosh/internal/api"
	"kosh/internal/store"
)

// newDemoHandlerForData returns a Handler in demo mode suitable for GetData
// tests. Reuses the same construction as newDemoHandler in mutations_test.go.
// Declared separately to keep each test file self-contained when read in
// isolation; the compiler will complain about the duplicate if they are ever
// merged into one file.
func newDemoHandlerForData() *api.Handler {
	fakeAPI := store.NewFakeSheetsAPI()
	repos := store.NewRepositories(fakeAPI)
	// Pass nil configAPI so the handler enters dev mode (loads devData from file
	// or returns empty Data). isDemo returning true ensures demo-mode short-circuit.
	return api.NewHandler(repos, nil, func(_ *http.Request) bool { return true })
}

func TestGetData_DemoMode_Returns200(t *testing.T) {
	h := newDemoHandlerForData()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()

	h.GetData(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetData_DemoMode_ResponseIsValidJSON(t *testing.T) {
	h := newDemoHandlerForData()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()

	h.GetData(rr, req)

	var v interface{}
	if err := json.NewDecoder(rr.Body).Decode(&v); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestGetData_DemoMode_ResponseHasExpectedTopLevelFields(t *testing.T) {
	h := newDemoHandlerForData()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()

	h.GetData(rr, req)

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response as JSON object: %v", err)
	}

	required := []string{"members", "mf", "stocks", "metals", "fixed", "insurance", "loans", "nps", "sips", "lumpsums", "history", "config"}
	for _, field := range required {
		if _, ok := body[field]; !ok {
			t.Errorf("expected top-level field %q to be present in response", field)
		}
	}
}

func TestGetData_DemoMode_EmptyDataHasNullSlicesOrEmptyArrays(t *testing.T) {
	// With no dev_data.json in the test's working directory (backend/internal/api/),
	// loadDevData returns &Data{} with nil slices. JSON-encoding a nil Go slice
	// produces "null", so each array field is either null or an empty array [].
	h := newDemoHandlerForData()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()

	h.GetData(rr, req)

	var body map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	arrayFields := []string{"members", "mf", "stocks", "metals", "fixed", "insurance", "loans", "nps", "sips", "lumpsums", "history"}
	for _, field := range arrayFields {
		raw, ok := body[field]
		if !ok {
			t.Errorf("field %q missing from response", field)
			continue
		}
		s := string(raw)
		if s != "null" && s != "[]" {
			t.Errorf("field %q: expected null or [], got %s", field, s)
		}
	}
}
