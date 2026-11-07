package store

// SheetsAPI is the minimal interface needed to read/write sheet rows.
// *sheets.Client implements this automatically (structural typing).
type SheetsAPI interface {
	ReadSheet(name string) ([][]interface{}, error)
	AppendRow(sheetName string, row []interface{}) (int, error)
	UpdateRow(sheetName, id string, row []interface{}) error
	DeleteRow(sheetName, id string) error
}

// BatchReader is optionally implemented by a SheetsAPI that can read many
// tabs in one round trip. Reading the whole dataset tab-by-tab costs one
// Sheets API call per tab, and every household on a hosted deploy shares one
// project-wide read quota — so callers should prefer this when available.
type BatchReader interface {
	// ReadSheets returns each named tab's rows (header included), keyed by
	// name. It fails as a whole if any tab is missing; callers fall back to
	// per-tab reads to find out which.
	ReadSheets(names []string) (map[string][][]interface{}, error)
}
