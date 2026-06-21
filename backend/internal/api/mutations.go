package api

import (
	"io"
	"net/http"
	"strings"
)

func (h *Handler) AddRow(w http.ResponseWriter, r *http.Request) {
	sheetName := sheetFromPath(r.URL.Path, "/api/")
	if h.servingSampleData(r) {
		writeJSON(w, map[string]string{"status": "ok", "note": "sample data — not persisted"})
		return
	}
	repo, ok := h.repos.Registry[sheetName]
	if !ok {
		http.Error(w, "unknown sheet: "+sheetName, http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	rowNum, err := repo.AddJSON(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	repo, ok := h.repos.Registry[sheetName]
	if !ok {
		http.Error(w, "unknown sheet: "+sheetName, http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	if err := repo.UpdateJSON(id, body); err != nil {
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
	repo, ok := h.repos.Registry[sheetName]
	if !ok {
		http.Error(w, "unknown sheet: "+sheetName, http.StatusBadRequest)
		return
	}
	if err := repo.DeleteByID(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// Mutations dispatches POST/PUT/DELETE /api/{sheet}[/{id}] to the right handler.
func (h *Handler) Mutations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	switch {
	case r.Method == http.MethodPost && len(parts) == 1:
		h.AddRow(w, r)
	case r.Method == http.MethodPut && len(parts) == 2:
		h.UpdateRow(w, r)
	case r.Method == http.MethodDelete && len(parts) == 2:
		h.DeleteRow(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func sheetFromPath(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	return strings.Split(s, "/")[0]
}
