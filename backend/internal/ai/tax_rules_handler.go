package ai

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"kosh/internal/models"
	"kosh/internal/store"
)

// TaxRulesHandler serves the tax-rules propose/approve/reject endpoints.
// Nothing it does ever changes what ComputeTax reads until a human
// approves — Propose only ever creates status="pending" rows, regardless
// of what a client sends.
type TaxRulesHandler struct {
	repos    *store.Repositories
	isDemo   func(*http.Request) bool
	isSample func(*http.Request) bool
}

func NewTaxRulesHandler(repos *store.Repositories, isDemo, isSample func(*http.Request) bool) *TaxRulesHandler {
	return &TaxRulesHandler{repos: repos, isDemo: isDemo, isSample: isSample}
}

// proposeRequest is the admin-reviewed payload from the review UI (after a
// document upload + human edit pass — see backend/prompts/tax_rules.md).
// The server ignores/overrides ID, Status, and GeneratedDate no matter what
// a client sends: a proposal can never be created as anything but "pending".
type proposeRequest struct {
	FY     string          `json:"fy"`
	Regime string          `json:"regime"`
	Source string          `json:"source"`
	Notes  string          `json:"notes"`
	Rules  models.TaxRules `json:"rules"`
}

func (h *TaxRulesHandler) Propose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Same soft "not available" pattern as ai.Handler/ai.TaxHandler for demo
	// sessions — a 200 with a note, not a hard error, so the frontend needs
	// no special-case handling.
	if h.isDemo != nil && h.isDemo(r) {
		writeJSON(w, map[string]any{
			"note": "Proposing tax rule changes is not available in demo mode. To enable it, deploy your own instance and sign in with your own account.",
		})
		return
	}

	var req proposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Regime != "old" && req.Regime != "new" {
		http.Error(w, `regime must be "old" or "new"`, http.StatusBadRequest)
		return
	}
	if req.FY == "" {
		http.Error(w, "fy is required", http.StatusBadRequest)
		return
	}

	rulesJSON, err := json.Marshal(req.Rules)
	if err != nil {
		http.Error(w, "encoding rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	row := models.TaxRuleSet{
		ID:            "trs" + strconv.FormatInt(time.Now().UnixNano(), 36),
		FY:            req.FY,
		Regime:        req.Regime,
		Status:        "pending",
		Source:        req.Source,
		GeneratedDate: time.Now().Format("2006-01-02"),
		RulesJSON:     string(rulesJSON),
		Notes:         req.Notes,
	}

	// Dev mode / demo session: mirrors every other mutation's no-op there
	// (see api.Handler.servingSampleData) but still returns the row so the
	// review UI can render what would have been created.
	if h.isSample != nil && h.isSample(r) {
		writeJSON(w, map[string]any{"status": "ok", "note": "sample data — not persisted", "rule_set": row})
		return
	}

	if _, err := h.repos.TaxRules.Add(row); err != nil {
		http.Error(w, "saving proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "rule_set": row})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
