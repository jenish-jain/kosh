// Package metalrates fetches India's benchmark gold and silver rates from IBJA
// (India Bullion and Jewellers Association, ibjarates.com) so metal holdings
// are valued at the day's published rate instead of a hand-typed price.
//
// IBJA publishes twice a day (AM/PM) on working days only; weekends and
// central-government holidays carry the last published rate forward.
package metalrates

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"kosh/internal/models"
)

const (
	DefaultURL = "https://ibjarates.com/"
	// IBJA updates at most twice a day, so an hourly refresh is plenty.
	refreshEvery = time.Hour
	// After a failed fetch, wait before hitting the site again.
	retryEvery   = 10 * time.Minute
	fetchTimeout = 10 * time.Second
	maxBody      = 2 << 20
)

var (
	rowRe  = regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	cellRe = regexp.MustCompile(`(?s)data-label="([^"]+)"[^>]*>\s*(?:<strong>)?\s*([^<]*)`)
)

// Parse extracts the most recent published rates from the ibjarates.com
// homepage. Rates come from the history tables, whose rows look like
// [AM|PM: dd/mm/yyyy] [Gold 999: ₹/10g] … [Silver 999: ₹/kg]; the newest
// date wins, and PM beats AM on the same date.
func Parse(html []byte) (*models.MetalRates, error) {
	var best *models.MetalRates
	var bestKey string
	for _, row := range rowRe.FindAllSubmatch(html, -1) {
		cells := cellRe.FindAllSubmatch(row[1], -1)
		if len(cells) == 0 {
			continue
		}
		session := string(cells[0][1])
		if session != "AM" && session != "PM" {
			continue
		}
		date, err := time.Parse("02/01/2006", strings.TrimSpace(string(cells[0][2])))
		if err != nil {
			continue
		}
		var gold10g, silverKg float64
		for _, c := range cells[1:] {
			v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(string(c[2])), ",", ""), 64)
			if err != nil {
				continue
			}
			switch string(c[1]) {
			case "Gold 999":
				gold10g = v
			case "Silver 999":
				silverKg = v
			}
		}
		r := &models.MetalRates{
			Source:  "IBJA",
			Date:    date.Format("2006-01-02"),
			Session: session,
			Gold:    gold10g / 10,
			Silver:  silverKg / 1000,
		}
		if !plausible(r) {
			continue
		}
		// "2026-11-13 PM" > "2026-11-13 AM" > "2026-11-12 PM" lexically.
		if key := r.Date + " " + session; key > bestKey {
			best, bestKey = r, key
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no IBJA rate rows found (page layout may have changed)")
	}
	return best, nil
}

// plausible guards against a layout change silently feeding nonsense (e.g. a
// per-gram figure read as per-10g) into everyone's net worth.
func plausible(r *models.MetalRates) bool {
	return r.Gold >= 1000 && r.Gold <= 1_000_000 && r.Silver >= 10 && r.Silver <= 100_000
}

// Fetcher serves the latest rates, refreshing from IBJA at most hourly. Once a
// rate has been fetched, refreshes happen in the background so requests never
// wait on IBJA; if it is unreachable, the last good rate keeps being served.
type Fetcher struct {
	url    string
	client *http.Client
	now    func() time.Time

	mu         sync.Mutex
	cur        *models.MetalRates
	nextTry    time.Time
	refreshing bool
}

// NewFetcher returns a Fetcher for url (DefaultURL when empty).
func NewFetcher(url string, client *http.Client) *Fetcher {
	if url == "" {
		url = DefaultURL
	}
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}
	return &Fetcher{url: url, client: client, now: time.Now}
}

// Get returns the current rates, or nil if none have ever been fetched
// successfully — callers then fall back to the prices stored in the sheet.
func (f *Fetcher) Get() *models.MetalRates {
	f.mu.Lock()
	due := !f.now().Before(f.nextTry) && !f.refreshing
	if due {
		f.refreshing = true
	}
	cur := f.cur
	f.mu.Unlock()

	if !due {
		return cur
	}
	if cur != nil {
		go f.refresh()
		return cur
	}
	f.refresh() // first fetch: nothing to serve yet, so wait for it
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cur
}

func (f *Fetcher) refresh() {
	r, err := f.fetch()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshing = false
	if err != nil {
		log.Printf("⚠ IBJA metal rates: %v", err)
		f.nextTry = f.now().Add(retryEvery)
		return
	}
	f.cur = r
	f.nextTry = f.now().Add(refreshEvery)
}

func (f *Fetcher) fetch() (*models.MetalRates, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Kosh)")
	res, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, err
	}
	return Parse(body)
}

// Apply sets each holding's TodayPrice to the live per-gram rate for its metal.
// Rates are for 999 purity; holdings of other metal types are left untouched.
func Apply(metals []models.Metal, r *models.MetalRates) {
	if r == nil {
		return
	}
	for i := range metals {
		switch strings.ToLower(strings.TrimSpace(metals[i].Type)) {
		case "gold":
			metals[i].TodayPrice = r.Gold
		case "silver":
			metals[i].TodayPrice = r.Silver
		}
	}
}
