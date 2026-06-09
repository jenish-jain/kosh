package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// sheetColumns maps sheet name → ordered column names (matches ReadSheet header).
var sheetColumns = map[string][]string{
	"MF":        {"id", "name", "plan", "platform", "member", "invested", "current", "sip", "notes"},
	"Stocks":    {"id", "name", "ticker", "qty", "avg_price", "last_price", "member"},
	"Metals":    {"id", "type", "date_purchased", "grams", "buy_rate", "today_price", "place", "member"},
	"Fixed":     {"id", "kind", "name", "member", "principal", "rate", "current_value", "opened", "matures", "monthly", "doc_link"},
	"Insurance": {"id", "name", "type", "member", "premium", "freq", "paid", "value", "cover", "maturity", "due_date", "doc_link"},
	"Loans":     {"id", "lender", "type", "member", "principal", "outstanding", "rate", "emi", "emi_day", "started", "tenure_months"},
	"NPS":       {"id", "pran", "member", "tier", "asset_class", "scheme", "fund_manager", "units", "nav", "invested"},
	"SIPs":      {"id", "fund", "member", "amount", "day", "status", "start_date", "platform", "kind"},
	"Lumpsums":  {"id", "fund", "member", "amount", "date"},
	"History":   {"month", "value"},
	"Members":   {"id", "name", "full_name", "relation", "slab", "color"},
}

func (h *Handler) AddRow(w http.ResponseWriter, r *http.Request) {
	sheetName := sheetFromPath(r.URL.Path, "/api/")
	if h.servingSampleData(r) {
		writeJSON(w, map[string]string{"status": "ok", "note": "sample data — not persisted"})
		return
	}
	cols, ok := sheetColumns[sheetName]
	if !ok {
		http.Error(w, "unknown sheet: "+sheetName, http.StatusBadRequest)
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	row := make([]interface{}, len(cols))
	for i, col := range cols {
		row[i] = fmt.Sprint(body[col])
	}
	rowNum, err := h.client.AppendRow(sheetName, row)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "row": rowNum})
}

func (h *Handler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	// path: /api/{sheet}/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	sheetName, id := parts[0], parts[1]
	if h.servingSampleData(r) {
		writeJSON(w, map[string]string{"status": "ok", "note": "sample data — not persisted"})
		return
	}
	cols, ok := sheetColumns[sheetName]
	if !ok {
		http.Error(w, "unknown sheet: "+sheetName, http.StatusBadRequest)
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	row := make([]interface{}, len(cols))
	for i, col := range cols {
		if v, exists := body[col]; exists {
			row[i] = fmt.Sprint(v)
		}
	}
	if err := h.client.UpdateRow(sheetName, id, row); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	sheetName, id := parts[0], parts[1]
	if h.servingSampleData(r) {
		writeJSON(w, map[string]string{"status": "ok", "note": "sample data — not persisted"})
		return
	}
	if err := h.client.DeleteRow(sheetName, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func sheetFromPath(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	return strings.Split(s, "/")[0]
}

// Capitalize first letter of sheet name for API path matching (e.g. "mf" → "MF").
func normalizeSheet(s string) string {
	// Map lowercase API paths to actual sheet names
	m := map[string]string{
		"mf": "MF", "stocks": "Stocks", "metals": "Metals",
		"fixed": "Fixed", "insurance": "Insurance", "loans": "Loans", "sips": "SIPs",
		"lumpsums": "Lumpsums", "history": "History", "members": "Members",
	}
	if v, ok := m[strings.ToLower(s)]; ok {
		return v
	}
	return s
}

func init() {
	_ = normalizeSheet // used via sheetFromPath in router
}
