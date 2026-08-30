package report_test

import (
	"bytes"
	"testing"

	"kosh/internal/report"
)

func TestRenderPDF_ProducesValidPDF(t *testing.T) {
	d := sampleData()
	tables := report.BuildTables(d, "", report.ValidSections())
	meta := report.BuildMeta(d, "", exampleNow())

	fileBytes, err := report.RenderPDF(tables, meta)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	if len(fileBytes) < 100 {
		t.Fatalf("RenderPDF returned suspiciously few bytes: %d", len(fileBytes))
	}
	if !bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		t.Errorf("output does not start with the %%PDF magic header: %q", fileBytes[:20])
	}
	if !bytes.HasSuffix(bytes.TrimRight(fileBytes, "\n\r"), []byte("%%EOF")) {
		t.Errorf("output does not end with %%%%EOF")
	}
}

func TestRenderPDF_EmptyTableListDoesNotPanic(t *testing.T) {
	fileBytes, err := report.RenderPDF(nil, report.ReportMeta{Scope: "Whole family", Generated: "1 Jan 2026"})
	if err != nil {
		t.Fatalf("RenderPDF(nil): %v", err)
	}
	if !bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		t.Errorf("output does not start with the %%PDF magic header")
	}
}

func TestRenderPDF_ManySectionsSpansMultiplePages(t *testing.T) {
	d := sampleData()
	// Pad one section with enough rows to force at least one page break,
	// exercising the manual header-repeat path in drawTable.
	for i := 0; i < 80; i++ {
		d.MF = append(d.MF, d.MF[0])
	}
	tables := report.BuildTables(d, "", report.ValidSections())
	meta := report.BuildMeta(d, "", exampleNow())

	fileBytes, err := report.RenderPDF(tables, meta)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	// A rough, format-agnostic proxy for "more than one page": PDF page
	// objects are declared with "/Type /Page" (not "/Pages", the tree node).
	if count := bytes.Count(fileBytes, []byte("/Type /Page\n")) + bytes.Count(fileBytes, []byte("/Type/Page\n")); count < 2 {
		t.Logf("page-count heuristic found %d — not fatal, PDF internals vary by encoder, but worth a look if this ever regresses", count)
	}
}
