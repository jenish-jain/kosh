package metalrates

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"kosh/internal/models"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/ibja.html")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParse_PicksLatestSessionAndConvertsToPerGram(t *testing.T) {
	r, err := Parse(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	// 13/11 PM beats 13/11 AM; gold is quoted per 10g, silver per kg.
	want := models.MetalRates{Source: "IBJA", Date: "2026-11-13", Session: "PM", Gold: 13580.9, Silver: 178}
	if *r != want {
		t.Errorf("got %+v, want %+v", *r, want)
	}
}

func TestParse_AMOnlyDay(t *testing.T) {
	html := `<tr><td data-label="AM"><strong>14/11/2026</strong></td><td data-label="Gold 999">136000</td><td data-label="Silver 999">179,000 </td></tr>`
	r, err := Parse(append(fixture(t), html...))
	if err != nil {
		t.Fatal(err)
	}
	if r.Date != "2026-11-14" || r.Session != "AM" || r.Gold != 13600 || r.Silver != 179 {
		t.Errorf("got %+v", *r)
	}
}

func TestParse_RejectsImplausibleAndUnknownLayouts(t *testing.T) {
	for name, html := range map[string]string{
		"empty":         ``,
		"no rows":       `<html><span id="GoldRatesCompare999">13581</span></html>`,
		"per-gram typo": `<tr><td data-label="PM"><strong>13/11/2026</strong></td><td data-label="Gold 999">1358</td><td data-label="Silver 999">178</td></tr>`,
		"missing metal": `<tr><td data-label="PM"><strong>13/11/2026</strong></td><td data-label="Gold 999">135809</td></tr>`,
		"bad date":      `<tr><td data-label="PM"><strong>yesterday</strong></td><td data-label="Gold 999">135809</td><td data-label="Silver 999">178000</td></tr>`,
	} {
		if r, err := Parse([]byte(html)); err == nil {
			t.Errorf("%s: expected error, got %+v", name, *r)
		}
	}
}

func TestApply(t *testing.T) {
	metals := []models.Metal{
		{Type: "Gold", TodayPrice: 1}, {Type: " silver ", TodayPrice: 2}, {Type: "Platinum", TodayPrice: 3},
	}
	Apply(metals, &models.MetalRates{Gold: 13580.9, Silver: 178})
	if metals[0].TodayPrice != 13580.9 || metals[1].TodayPrice != 178 || metals[2].TodayPrice != 3 {
		t.Errorf("got %+v", metals)
	}
	Apply(metals, nil) // no rates: untouched
	if metals[0].TodayPrice != 13580.9 {
		t.Errorf("nil rates changed prices: %+v", metals)
	}
}

func TestFetcher_CachesAndKeepsLastGoodRate(t *testing.T) {
	var hits atomic.Int32
	var fail atomic.Bool
	page := fixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if fail.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		w.Write(page)
	}))
	defer srv.Close()

	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	f := NewFetcher(srv.URL, srv.Client())
	f.now = func() time.Time { return now }

	if r := f.Get(); r == nil || r.Gold != 13580.9 {
		t.Fatalf("first Get = %+v", r)
	}
	f.Get()
	if hits.Load() != 1 {
		t.Fatalf("expected 1 fetch within the refresh window, got %d", hits.Load())
	}

	// Past the refresh window with IBJA down: still serves the cached rate.
	fail.Store(true)
	now = now.Add(2 * time.Hour)
	if r := f.Get(); r == nil || r.Gold != 13580.9 {
		t.Fatalf("stale Get = %+v", r)
	}
	waitFor(t, func() bool { return hits.Load() == 2 })
	waitFor(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return !f.refreshing })
	if r := f.Get(); r == nil || r.Gold != 13580.9 {
		t.Fatalf("after failed refresh Get = %+v", r)
	}
	if hits.Load() != 2 {
		t.Errorf("failed refresh should back off, got %d fetches", hits.Load())
	}
}

func TestFetcher_NeverFetched_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	if r := NewFetcher(srv.URL, srv.Client()).Get(); r != nil {
		t.Errorf("got %+v, want nil", *r)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
