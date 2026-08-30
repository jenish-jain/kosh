package report

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// RenderXLSX builds a workbook with one sheet per table — a title/scope row
// at the top, a bold header row, the data rows, and a bold totals row where
// present.
func RenderXLSX(tables []Table, meta ReportMeta) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	headerStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	totalsStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	titleStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 13}})
	if err != nil {
		return nil, err
	}

	usedNames := map[string]int{}
	var firstSheet string
	for _, t := range tables {
		name := sheetName(t.Title, usedNames)
		if _, err := f.NewSheet(name); err != nil {
			return nil, fmt.Errorf("creating sheet %q: %w", name, err)
		}
		if firstSheet == "" {
			firstSheet = name
		}

		row := 1
		titleCell := cellRef(0, row)
		f.SetCellStr(name, titleCell, "Kosh — "+t.Title)
		f.SetCellStyle(name, titleCell, titleCell, titleStyle)
		row++
		f.SetCellStr(name, cellRef(0, row), meta.Scope+" · Generated "+meta.Generated)
		row += 2

		headerRow := row
		for col, h := range t.Headers {
			cell := cellRef(col, headerRow)
			f.SetCellStr(name, cell, h)
			f.SetCellStyle(name, cell, cell, headerStyle)
		}
		row++

		for _, r := range t.Rows {
			for col, v := range r {
				f.SetCellStr(name, cellRef(col, row), v)
			}
			row++
		}

		if len(t.Totals) > 0 {
			for col, v := range t.Totals {
				cell := cellRef(col, row)
				f.SetCellStr(name, cell, v)
				f.SetCellStyle(name, cell, cell, totalsStyle)
			}
		}

		for col, h := range t.Headers {
			colName, err := excelize.ColumnNumberToName(col + 1)
			if err != nil {
				continue
			}
			width := float64(len(h)) + 6
			if width < 14 {
				width = 14
			}
			f.SetColWidth(name, colName, colName, width)
		}
	}

	if firstSheet != "" {
		f.SetActiveSheet(0)
		f.DeleteSheet("Sheet1")
	}

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cellRef(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col+1, row)
	return name
}

// sheetName sanitizes and de-duplicates an Excel sheet name — max 31 chars,
// none of : \ / ? * [ ].
func sheetName(title string, used map[string]int) string {
	replacer := strings.NewReplacer(":", "-", "\\", "-", "/", "-", "?", "", "*", "", "[", "(", "]", ")")
	name := replacer.Replace(title)
	if len(name) > 31 {
		name = name[:31]
	}
	used[name]++
	if n := used[name]; n > 1 {
		suffix := fmt.Sprintf(" %d", n)
		if len(name)+len(suffix) > 31 {
			name = name[:31-len(suffix)]
		}
		name += suffix
	}
	return name
}
