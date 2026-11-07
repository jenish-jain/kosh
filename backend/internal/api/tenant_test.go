package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kosh/internal/api"
	"kosh/internal/models"
	"kosh/internal/store"
	"kosh/internal/tenant"
)

// liveTenant returns a tenant whose fake spreadsheet has every data tab's
// header row (as EnsureTabs would create) plus one Members row.
func liveTenant(t *testing.T, id, memberName string) (*tenant.Tenant, *store.FakeSheetsAPI) {
	t.Helper()
	fake := store.NewFakeSheetsAPI()
	tn := tenant.New(id, fake)
	for _, r := range tn.Repos.Registry {
		header := make([]interface{}, len(r.Columns()))
		for i, c := range r.Columns() {
			header[i] = c
		}
		fake.Data[r.Sheet()] = [][]interface{}{header}
	}
	fake.Data["Config"] = [][]interface{}{{"key", "value"}, {"regime", "new"}}
	fake.Data["Members"] = append(fake.Data["Members"], []interface{}{"m1", memberName})
	return tn, fake
}

func getData(t *testing.T, h *api.Handler, req *http.Request) (int, models.Data) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.GetData(rr, req)
	var d models.Data
	if rr.Code == http.StatusOK {
		if err := json.NewDecoder(rr.Body).Decode(&d); err != nil {
			t.Fatalf("decoding: %v", err)
		}
	}
	return rr.Code, d
}

func notDemo(*http.Request) bool { return false }

func TestGetData_Live_ReadsAllTabsInOneBatch(t *testing.T) {
	tn, fake := liveTenant(t, "default", "Asha")
	h := api.NewHandler(tenant.Single(tn), false, notDemo)

	code, d := getData(t, h, httptest.NewRequest(http.MethodGet, "/api/data", nil))
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(d.Members) != 1 || d.Members[0].Name != "Asha" || d.Config.Regime != "new" {
		t.Fatalf("unexpected data: members=%+v config=%+v", d.Members, d.Config)
	}
	if fake.Calls != 1 {
		t.Errorf("want 1 batched read, got %d read calls", fake.Calls)
	}
}

func TestGetData_Live_MissingTabFallsBackToPerTabReads(t *testing.T) {
	tn, fake := liveTenant(t, "default", "Asha")
	delete(fake.Data, "NPS") // e.g. an older sheet created before NPS existed
	h := api.NewHandler(tenant.Single(tn), false, notDemo)

	code, d := getData(t, h, httptest.NewRequest(http.MethodGet, "/api/data", nil))
	if code != http.StatusOK || len(d.Members) != 1 || d.Config.Regime != "new" {
		t.Fatalf("missing tab should only empty that tab: status=%d members=%+v", code, d.Members)
	}
	if fake.Calls <= 1 {
		t.Errorf("expected per-tab fallback reads after the batch failed, got %d calls", fake.Calls)
	}
}

// householdResolver picks a tenant by request header — a stand-in for the
// hosted resolver, which will pick by session.
type householdResolver map[string]*tenant.Tenant

func (m householdResolver) Resolve(r *http.Request) (*tenant.Tenant, error) {
	if t, ok := m[r.Header.Get("X-Household")]; ok {
		return t, nil
	}
	return nil, errors.New("no household")
}

func TestHandler_RoutesEachRequestToItsTenant(t *testing.T) {
	a, fakeA := liveTenant(t, "a", "Asha")
	b, fakeB := liveTenant(t, "b", "Bharat")
	h := api.NewHandler(householdResolver{"a": a, "b": b}, false, notDemo)

	for id, want := range map[string]string{"a": "Asha", "b": "Bharat"} {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("X-Household", id)
		if _, d := getData(t, h, req); len(d.Members) != 1 || d.Members[0].Name != want {
			t.Errorf("household %s: want member %s, got %+v", id, want, d.Members)
		}
	}

	// A write lands only in the requesting household's sheet.
	req := httptest.NewRequest(http.MethodPost, "/api/Members", strings.NewReader(`{"id":"m2","name":"Chitra"}`))
	req.Header.Set("X-Household", "b")
	rr := httptest.NewRecorder()
	h.Mutations(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("add: %d %s", rr.Code, rr.Body)
	}
	if len(fakeB.Data["Members"]) != 3 || len(fakeA.Data["Members"]) != 2 {
		t.Errorf("write leaked across tenants: a=%d rows, b=%d rows", len(fakeA.Data["Members"]), len(fakeB.Data["Members"]))
	}
}

func TestHandler_UnresolvableTenantIsRejected(t *testing.T) {
	h := api.NewHandler(householdResolver{}, false, notDemo)
	if code, _ := getData(t, h, httptest.NewRequest(http.MethodGet, "/api/data", nil)); code != http.StatusUnauthorized {
		t.Errorf("GetData: want 401, got %d", code)
	}
	rr := httptest.NewRecorder()
	h.Mutations(rr, httptest.NewRequest(http.MethodPost, "/api/Members", strings.NewReader(`{}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Mutations: want 401, got %d", rr.Code)
	}
}
