package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kosh/internal/models"
	"kosh/internal/store"
)

// TaxHandler serves POST /api/tax/recommendations/generate.
type TaxHandler struct {
	provider Provider
	repos    *store.Repositories
	loadData DataLoader
	isDemo   func(*http.Request) bool
	isSample func(*http.Request) bool
}

func NewTaxHandler(provider Provider, repos *store.Repositories, loadData DataLoader, isDemo, isSample func(*http.Request) bool) *TaxHandler {
	return &TaxHandler{provider: provider, repos: repos, loadData: loadData, isDemo: isDemo, isSample: isSample}
}

func (h *TaxHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return a friendly message for demo sessions instead of hitting the real provider.
	if h.isDemo != nil && h.isDemo(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"recommendations": []any{},
			"note":            "Tax-saving recommendations are not available in demo mode. To enable them, deploy your own instance with AI_ENABLED=true and your preferred AI provider configured.",
		})
		return
	}

	var req struct {
		Regime string `json:"regime"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Regime == "" {
		req.Regime = "old"
	}

	data := h.loadData(r)
	now := time.Now()
	facts := models.ComputeTaxFacts(data, req.Regime, now)

	var prior []models.TaxRecommendation
	if h.repos != nil && h.repos.TaxRecommendations != nil {
		if rows, err := h.repos.TaxRecommendations.All(); err == nil {
			prior = latestCycle(rows)
		}
	}

	messages := []Message{
		{Role: "system", Content: BuildTaxRecommendationPrompt(facts, prior)},
		{Role: "user", Content: "Generate my tax-saving recommendations now, per the output format specified."},
	}

	reply, err := h.provider.Chat(r.Context(), messages)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	items, err := parseRecommendations(reply)
	if err != nil {
		http.Error(w, `{"error":"parsing AI response: `+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	today := now.Format("2006-01-02")
	stamp := strconv.FormatInt(now.UnixNano(), 36)
	out := make([]models.TaxRecommendation, 0, len(items))
	for i, item := range items {
		out = append(out, models.TaxRecommendation{
			ID:                 fmt.Sprintf("tr%s-%d", stamp, i),
			GeneratedDate:      today,
			FY:                 facts.FY,
			Regime:             facts.Regime,
			Category:           item.Category,
			Headline:           item.Headline,
			SuggestedAmount:    item.SuggestedAmount,
			PotentialTaxSaving: item.PotentialTaxSaving,
			Rationale:          item.Rationale,
			Status:             "new",
		})
	}

	// In dev mode / demo sessions, skip persistence (mirrors every other
	// mutation's no-op there) but still return the generated rows so local
	// testing exercises a real AI round trip.
	if h.isSample == nil || !h.isSample(r) {
		for i, item := range items {
			if item.SupersedesID != "" {
				_ = h.repos.TaxRecommendations.Update(item.SupersedesID, map[string]any{"superseded_by": out[i].ID})
			}
		}
		for _, rec := range out {
			if _, err := h.repos.TaxRecommendations.Add(rec); err != nil {
				http.Error(w, `{"error":"saving recommendation: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"recommendations": out})
}

// latestCycle returns only the most recent generation's rows (by
// GeneratedDate) so the prompt's prior-recommendations block doesn't grow
// unbounded across many cycles.
func latestCycle(rows []models.TaxRecommendation) []models.TaxRecommendation {
	if len(rows) == 0 {
		return nil
	}
	latest := rows[0].GeneratedDate
	for _, r := range rows {
		if r.GeneratedDate > latest {
			latest = r.GeneratedDate
		}
	}
	var out []models.TaxRecommendation
	for _, r := range rows {
		if r.GeneratedDate == latest {
			out = append(out, r)
		}
	}
	return out
}

type recommendationItem struct {
	Category           string  `json:"category"`
	Headline           string  `json:"headline"`
	SuggestedAmount    float64 `json:"suggested_amount"`
	PotentialTaxSaving float64 `json:"potential_tax_saving"`
	Rationale          string  `json:"rationale"`
	SupersedesID       string  `json:"supersedes_id"`
}

// parseRecommendations extracts a JSON array from the model's raw text
// response — same first-bracket/last-bracket approach as
// documents.ClaudeParser uses for a top-level object, applied to an array.
func parseRecommendations(text string) ([]recommendationItem, error) {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "["); i > 0 {
		text = text[i:]
	}
	if i := strings.LastIndex(text, "]"); i >= 0 {
		text = text[:i+1]
	}
	var items []recommendationItem
	if err := json.Unmarshal([]byte(text), &items); err != nil {
		return nil, fmt.Errorf("%w — raw: %s", err, text)
	}
	return items, nil
}
