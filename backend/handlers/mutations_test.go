package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kosh/handlers"
	sh "kosh/sheets"
)

// newDemoHandler returns a Handler in demo mode: client is non-nil (so
// loadDevData is not called), and isDemo always returns true — satisfying
// servingSampleData without touching the Sheets API.
func newDemoHandler() *handlers.Handler {
	return handlers.NewHandler(new(sh.Client), func(_ *http.Request) bool { return true })
}

func TestAddRow_DemoMode_ReturnsOKForKnownSheet(t *testing.T) {
	h := newDemoHandler()
	body := `{"name":"HDFC Top 100","plan":"Direct","platform":"Zerodha","member":"J","invested":"10000","current":"12000","sip":"1000","notes":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/MF", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.AddRow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAddRow_DemoMode_ReturnsOKForAnySheet(t *testing.T) {
	// In demo mode, AddRow short-circuits before sheet-name validation.
	h := newDemoHandler()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/UnknownSheet", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.AddRow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown sheet in demo mode, got %d", rr.Code)
	}
}

func TestAddRow_DemoMode_ResponseHasSampleNote(t *testing.T) {
	h := newDemoHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/Stocks", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.AddRow(rr, req)

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", resp["status"])
	}
	if resp["note"] == "" {
		t.Error("expected a non-empty 'note' field in demo mode response")
	}
	const want = "sample data"
	if !strings.Contains(resp["note"], want) {
		t.Errorf("note field %q does not contain %q", resp["note"], want)
	}
}

func TestUpdateRow_DemoMode_ReturnsOK(t *testing.T) {
	h := newDemoHandler()
	body := `{"name":"Reliance"}`
	req := httptest.NewRequest(http.MethodPut, "/api/MF/id1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateRow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestUpdateRow_BadPath_Returns400(t *testing.T) {
	// Path has only one segment after /api/ — no ID provided.
	// 400 is returned before the demo-mode check, so handler mode does not matter.
	h := newDemoHandler()
	req := httptest.NewRequest(http.MethodPut, "/api/MF", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateRow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for path without ID, got %d", rr.Code)
	}
}

func TestDeleteRow_DemoMode_ReturnsOK(t *testing.T) {
	h := newDemoHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/NPS/id1", nil)
	rr := httptest.NewRecorder()

	h.DeleteRow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestDeleteRow_BadPath_Returns400(t *testing.T) {
	// Path has only one segment after /api/ — no ID provided.
	h := newDemoHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/MF", nil)
	rr := httptest.NewRecorder()

	h.DeleteRow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for path without ID, got %d", rr.Code)
	}
}
