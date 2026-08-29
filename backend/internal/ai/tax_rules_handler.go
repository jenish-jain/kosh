package ai

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

// Route dispatches "/api/tax/rules/{id}/approve" and ".../reject" — the
// exact-path "/api/tax/rules/propose" pattern registered separately in
// server.go takes priority over this subtree handler for that one path.
func (h *TaxRulesHandler) Route(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/approve"):
		h.Approve(w, r)
	case strings.HasSuffix(r.URL.Path, "/reject"):
		h.Reject(w, r)
	default:
		http.NotFound(w, r)
	}
}

// idFromPath extracts the {id} segment from "/api/tax/rules/{id}/approve"
// or "/api/tax/rules/{id}/reject".
func idFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/tax/rules/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func (h *TaxRulesHandler) findByID(id string) (models.TaxRuleSet, bool) {
	rows, err := h.repos.TaxRules.All()
	if err != nil {
		return models.TaxRuleSet{}, false
	}
	for _, r := range rows {
		if r.ID == id {
			return r, true
		}
	}
	return models.TaxRuleSet{}, false
}

// Approve flips a "pending" row to "active" and supersedes whatever other
// row currently holds "active" for the same regime — activate FIRST, then
// supersede: Repository[T].Update is a read-modify-write per call with no
// batching (Sheets has no transactions), so a crash between the two steps
// must never leave a regime with zero active rows. Two-active-rows-at-once
// is the tolerable failure mode: activeRuleSet() resolves it by picking
// whichever was activated most recently.
func (h *TaxRulesHandler) Approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.isDemo != nil && h.isDemo(r) {
		writeJSON(w, map[string]any{"note": "Approving tax rule changes is not available in demo mode."})
		return
	}
	if h.isSample != nil && h.isSample(r) {
		writeJSON(w, map[string]any{"status": "ok", "note": "sample data — not persisted"})
		return
	}

	id := idFromPath(r.URL.Path)
	target, ok := h.findByID(id)
	if !ok {
		http.Error(w, "tax rule proposal not found", http.StatusNotFound)
		return
	}
	if target.Status != "pending" {
		http.Error(w, "only a pending proposal can be approved", http.StatusConflict)
		return
	}

	today := time.Now().Format("2006-01-02")
	if err := h.repos.TaxRules.Update(id, map[string]any{"status": "active", "activated_date": today}); err != nil {
		http.Error(w, "activating proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := h.repos.TaxRules.All()
	if err != nil {
		// The new row is already active — superseding the old one failing to
		// even look up candidates isn't fatal, just log-worthy in a bigger app.
		writeJSON(w, map[string]any{"status": "ok", "warning": "activated, but could not check for rows to supersede: " + err.Error()})
		return
	}
	for _, other := range rows {
		if other.ID == id || other.Status != "active" || !strings.EqualFold(other.Regime, target.Regime) {
			continue
		}
		if err := h.repos.TaxRules.Update(other.ID, map[string]any{"status": "superseded"}); err != nil {
			writeJSON(w, map[string]any{"status": "ok", "warning": "activated, but failed to supersede " + other.ID + ": " + err.Error()})
			return
		}
	}
	writeJSON(w, map[string]any{"status": "ok"})
}

// Reject flips a "pending" row to "rejected". No cascade — other pending
// proposals for the same FY+regime are left untouched for an admin to
// handle individually.
func (h *TaxRulesHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.isDemo != nil && h.isDemo(r) {
		writeJSON(w, map[string]any{"note": "Rejecting tax rule changes is not available in demo mode."})
		return
	}
	if h.isSample != nil && h.isSample(r) {
		writeJSON(w, map[string]any{"status": "ok", "note": "sample data — not persisted"})
		return
	}

	id := idFromPath(r.URL.Path)
	target, ok := h.findByID(id)
	if !ok {
		http.Error(w, "tax rule proposal not found", http.StatusNotFound)
		return
	}
	if target.Status != "pending" {
		http.Error(w, "only a pending proposal can be rejected", http.StatusConflict)
		return
	}

	if err := h.repos.TaxRules.Update(id, map[string]any{"status": "rejected"}); err != nil {
		http.Error(w, "rejecting proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
