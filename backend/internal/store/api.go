package store

// SheetsAPI is the minimal interface needed to read/write sheet rows.
// *sheets.Client implements this automatically (structural typing).
type SheetsAPI interface {
	ReadSheet(name string) ([][]interface{}, error)
	AppendRow(sheetName string, row []interface{}) (int, error)
	UpdateRow(sheetName, id string, row []interface{}) error
	DeleteRow(sheetName, id string) error
}
