package store

import (
	"fmt"
	"sync"
)

// FakeSheetsAPI is an in-memory implementation of SheetsAPI for use in tests
// (and in dev mode, where family shares live in it). Safe for concurrent use.
type FakeSheetsAPI struct {
	Data map[string][][]interface{} // sheet name -> rows (row 0 = header)
	// Calls counts read round trips (ReadSheet and ReadSheets each count
	// one), so tests can assert on API usage.
	Calls int
	mu    sync.Mutex
}

// NewFakeSheetsAPI returns an initialised FakeSheetsAPI with no sheets.
func NewFakeSheetsAPI() *FakeSheetsAPI {
	return &FakeSheetsAPI{Data: make(map[string][][]interface{})}
}

// ReadSheet returns all rows for the named sheet (including header row 0).
// Returns an empty slice (not an error) if the sheet hasn't been seeded.
func (f *FakeSheetsAPI) ReadSheet(name string) ([][]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	return f.Data[name], nil
}

// ReadSheets implements BatchReader. Like the real API, it fails as a whole
// if any requested tab doesn't exist.
func (f *FakeSheetsAPI) ReadSheets(names []string) (map[string][][]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	out := make(map[string][][]interface{}, len(names))
	for _, n := range names {
		rows, ok := f.Data[n]
		if !ok {
			return nil, fmt.Errorf("unable to parse range: %s", n)
		}
		out[n] = rows
	}
	return out, nil
}

// AppendRow appends row to the named sheet and returns its 1-indexed row number.
func (f *FakeSheetsAPI) AppendRow(sheetName string, row []interface{}) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Data[sheetName] = append(f.Data[sheetName], row)
	return len(f.Data[sheetName]), nil
}

// UpdateRow replaces the row in sheetName where column 0 equals id.
func (f *FakeSheetsAPI) UpdateRow(sheetName, id string, row []interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := f.Data[sheetName]
	for i, r := range rows {
		if len(r) > 0 && fmt.Sprint(r[0]) == id {
			f.Data[sheetName] = append(rows[:i], rows[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("row with id %s not found in %s", id, sheetName)
}
