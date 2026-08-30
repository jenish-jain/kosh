package report

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

const (
	pageMargin   = 12.0
	rowHeight    = 7.0
	headerHeight = 8.0
)

// RenderPDF lays out a cover block (scope, generation date, a short
// disclaimer) followed by one bordered table per section. Page breaks are
// managed manually (SetAutoPageBreak is off) so a table's header row can be
// repeated at the top of a new page instead of orphaning data rows under no
// header.
func RenderPDF(tables []Table, meta ReportMeta) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	// Core fonts (Helvetica) render cp1252/WinAnsi bytes, not raw UTF-8 — a
	// Go string passed straight to CellFormat/MultiCell is written byte-for-
	// byte, so any multi-byte UTF-8 rune (accented letters, the ellipsis
	// truncation marker, anything pdfSafe doesn't already substitute)
	// corrupts into garbage. tr() is the documented fpdf/gofpdf pattern for
	// converting UTF-8 input to what the core font can actually render.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetMargins(pageMargin, pageMargin, pageMargin)
	pdf.SetAutoPageBreak(false, pageMargin)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, tr("Kosh - Financial Report"), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 7, tr(pdfSafe(meta.Scope+" - Generated "+meta.Generated)), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(0, 5, tr("For informational and accounting purposes only. Figures reflect the data recorded in Kosh as of the generation date above -- please verify against source documents before relying on them."), "", "L", false)
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	for _, t := range tables {
		drawTable(pdf, tr, t)
		pdf.Ln(6)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("rendering pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func drawTable(pdf *fpdf.Fpdf, tr func(string) string, t Table) {
	pageW, pageH := pdf.GetPageSize()
	_, _, _, marginBottom := pdf.GetMargins()
	printableW := pageW - pageMargin*2
	printableBottom := pageH - marginBottom

	// Don't orphan a title + header row alone at the bottom of a page.
	if pdf.GetY()+10+headerHeight+rowHeight > printableBottom {
		pdf.AddPage()
	}

	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, tr(pdfSafe(t.Title)), "", 1, "L", false, 0, "")
	pdf.Ln(1)

	n := len(t.Headers)
	if n == 0 {
		return
	}
	widths := columnWidths(printableW, n)

	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 8.5)
		pdf.SetFillColor(230, 227, 219)
		for i, h := range t.Headers {
			pdf.CellFormat(widths[i], headerHeight, pdfCell(pdf, tr, h, widths[i]), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
	drawHeader()

	pdf.SetFont("Helvetica", "", 8.5)
	for _, row := range t.Rows {
		if pdf.GetY()+rowHeight > printableBottom {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont("Helvetica", "", 8.5)
		}
		for i := 0; i < n; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			pdf.CellFormat(widths[i], rowHeight, pdfCell(pdf, tr, cell, widths[i]), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	if len(t.Totals) > 0 {
		if pdf.GetY()+rowHeight > printableBottom {
			pdf.AddPage()
			drawHeader()
		}
		pdf.SetFont("Helvetica", "B", 8.5)
		pdf.SetFillColor(245, 243, 238)
		for i := 0; i < n; i++ {
			cell := ""
			if i < len(t.Totals) {
				cell = t.Totals[i]
			}
			pdf.CellFormat(widths[i], rowHeight, pdfCell(pdf, tr, cell, widths[i]), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
}

// columnWidths gives the first column (typically a name/label) more room
// than the rest, evenly distributing the remainder.
func columnWidths(total float64, n int) []float64 {
	if n == 0 {
		return nil
	}
	weights := make([]float64, n)
	sum := 0.0
	for i := range weights {
		if i == 0 {
			weights[i] = 2.0
		} else {
			weights[i] = 1.0
		}
		sum += weights[i]
	}
	out := make([]float64, n)
	for i, w := range weights {
		out[i] = total * w / sum
	}
	return out
}

// pdfSafe substitutes characters worth a deliberate, readable ASCII
// replacement rather than leaving them to generic transliteration — ₹
// (U+20B9) becomes "Rs " and the minus sign U+2212 FormatINR uses for
// negatives becomes a plain hyphen.
func pdfSafe(s string) string {
	s = strings.ReplaceAll(s, "₹", "Rs ")
	s = strings.ReplaceAll(s, "−", "-")
	return s
}

// pdfCell applies pdfSafe, truncates by actual rendered string width (not
// byte/rune count) so long fund/scheme names don't spill into the next
// column, then runs the result through tr() — the Unicode-to-core-font
// translation — as the final step, since GetStringWidth here is measuring
// against an already ASCII-heavy string where UTF-8 byte length and glyph
// count agree.
func pdfCell(pdf *fpdf.Fpdf, tr func(string) string, s string, maxWidth float64) string {
	s = pdfSafe(s)
	const padding = 2.0
	const ellipsis = ".."
	avail := maxWidth - padding
	if pdf.GetStringWidth(s) > avail {
		runes := []rune(s)
		for len(runes) > 1 && pdf.GetStringWidth(string(runes)+ellipsis) > avail {
			runes = runes[:len(runes)-1]
		}
		s = string(runes) + ellipsis
	}
	return tr(s)
}
