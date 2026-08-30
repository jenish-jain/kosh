package report_test

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
	"kosh/internal/report"
)

func TestRenderXLSX_RoundTrips(t *testing.T) {
	d := sampleData()
	tables := report.BuildTables(d, "", report.ValidSections())
	meta := report.BuildMeta(d, "", exampleNow())

	fileBytes, err := report.RenderXLSX(tables, meta)
	if err != nil {
		t.Fatalf("RenderXLSX: %v", err)
	}
	if len(fileBytes) == 0 {
		t.Fatal("RenderXLSX returned no bytes")
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatalf("reopening generated workbook: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	// +1 for the trailing "About" branding sheet.
	if len(sheets) != len(tables)+1 {
		t.Errorf("got %d sheets, want %d (tables: %v, sheets: %v)", len(sheets), len(tables)+1, tableTitles(tables), sheets)
	}
	for _, s := range sheets {
		if s == "Sheet1" {
			t.Errorf("default Sheet1 was not removed; sheets = %v", sheets)
		}
	}

	about, err := f.GetCellValue("About", "A1")
	if err != nil {
		t.Fatalf("GetCellValue(About, A1): %v", err)
	}
	if about != report.BrandingFooter {
		t.Errorf("About!A1 = %q, want %q", about, report.BrandingFooter)
	}

	got, err := f.GetCellValue("Mutual Funds", "A4")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if got != "Fund" {
		t.Errorf("A4 (first header cell) = %q, want %q", got, "Fund")
	}
	got, err = f.GetCellValue("Mutual Funds", "A5")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if got != "Fund A" {
		t.Errorf("A5 (first data row) = %q, want %q", got, "Fund A")
	}
}

func TestRenderXLSX_EmptyTableListDoesNotPanic(t *testing.T) {
	if _, err := report.RenderXLSX(nil, report.ReportMeta{Scope: "Whole family", Generated: "1 Jan 2026"}); err != nil {
		t.Fatalf("RenderXLSX(nil): %v", err)
	}
}

func tableTitles(tables []report.Table) []string {
	out := make([]string, len(tables))
	for i, tb := range tables {
		out[i] = tb.Title
	}
	return out
}
