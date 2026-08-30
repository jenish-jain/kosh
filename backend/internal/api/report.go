package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"kosh/internal/report"
)

// ExportReport serves GET /api/report/export?format=pdf|xlsx&member=<id-or-empty>&sections=mf,fixed,...
// Read-only, so unlike the AI/rules endpoints there's no demo-mode gating —
// h.LoadData already resolves to the right dataset (sample/demo/live) for
// this session the same way every other read path does.
func (h *Handler) ExportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format != "pdf" && format != "xlsx" {
		http.Error(w, `format must be "pdf" or "xlsx"`, http.StatusBadRequest)
		return
	}

	member := r.URL.Query().Get("member")

	valid := map[string]bool{}
	for _, s := range report.ValidSections() {
		valid[s] = true
	}
	var sections []string
	for _, s := range strings.Split(r.URL.Query().Get("sections"), ",") {
		s = strings.TrimSpace(s)
		if valid[s] {
			sections = append(sections, s)
		}
	}
	if len(sections) == 0 {
		http.Error(w, "no valid sections requested — pass e.g. sections=mf,fixed,insurance", http.StatusBadRequest)
		return
	}

	data := h.LoadData(r)
	tables := report.BuildTables(data, member, sections)
	meta := report.BuildMeta(data, member, time.Now())
	filename := reportFilename(meta)

	switch format {
	case "xlsx":
		fileBytes, err := report.RenderXLSX(tables, meta)
		if err != nil {
			http.Error(w, "generating report: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))
		w.Write(fileBytes)
	case "pdf":
		http.Error(w, "PDF export is not available yet", http.StatusNotImplemented)
	}
}

func reportFilename(meta report.ReportMeta) string {
	scopeSlug := strings.ToLower(strings.ReplaceAll(meta.Scope, " ", "-"))
	dateSlug := time.Now().Format("2006-01-02")
	return fmt.Sprintf("Kosh-Report-%s-%s", scopeSlug, dateSlug)
}
