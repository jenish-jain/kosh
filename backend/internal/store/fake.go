package store

import "fmt"

// FakeSheetsAPI is an in-memory implementation of SheetsAPI for use in tests.
type FakeSheetsAPI struct {
	Data map[string][][]interface{} // sheet name -> rows (row 0 = header)
}

// NewFakeSheetsAPI returns an initialised FakeSheetsAPI with no sheets.
func NewFakeSheetsAPI() *FakeSheetsAPI {
	return &FakeSheetsAPI{Data: make(map[string][][]interface{})}
}

// ReadSheet returns all rows for the named sheet (including header row 0).
// Returns an empty slice (not an error) if the sheet hasn't been seeded.
func (f *FakeSheetsAPI) ReadSheet(name string) ([][]interface{}, error) {
	return f.Data[name], nil
}

// AppendRow appends row to the named sheet and returns its 1-indexed row number.
func (f *FakeSheetsAPI) AppendRow(sheetName string, row []interface{}) (int, error) {
	f.Data[sheetName] = append(f.Data[sheetName], row)
	return len(f.Data[sheetName]), nil
}

// UpdateRow replaces the row in sheetName where column 0 equals id.
func (f *FakeSheetsAPI) UpdateRow(sheetName, id string, row []interface{}) error {
	rows := f.Data[sheetName]
	for i, r := range rows {
		if len(r) > 0 && fmt.Sprint(r[0]) == id {
			f.Data[sheetName][i] = row
			return nil
		}
	}
	return fmt.Errorf("row with id %s not found in %s", id, sheetName)
}

// DeleteRow removes the row in sheetName where column 0 equals id.
func (f *FakeSheetsAPI) DeleteRow(sheetName, id string) error {
	rows := f.Data[sheetName]
	for i, r := range rows {
		if len(r) > 0 && fmt.Sprint(r[0]) == id {
			f.Data[sheetName] = append(rows[:i], rows[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("row with id %s not found in %s", id, sheetName)
}
