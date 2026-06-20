package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// AnyRepository is the type-erased interface for the mutation dispatch registry.
type AnyRepository interface {
	Columns() []string
	Sheet() string
	AddJSON(body []byte) (int, error)
	UpdateJSON(id string, body []byte) error
	DeleteByID(id string) error
}

// Repository is a generic, reflection-driven mapper between Go structs and
// Google Sheets rows. T must be a struct with `sheet:"..."` field tags.
type Repository[T any] struct {
	api     SheetsAPI
	sheet   string
	columns []string
	indices map[string]int // column name -> position index
}

// NewRepository constructs a Repository for sheet tab `sheet`, reading/writing
// via `api`. Column order is derived from the `sheet` struct tags of T in field
// declaration order.
func NewRepository[T any](api SheetsAPI, sheet string) *Repository[T] {
	cols := columnsOf[T]()
	idx := make(map[string]int, len(cols))
	for i, c := range cols {
		idx[c] = i
	}
	return &Repository[T]{api: api, sheet: sheet, columns: cols, indices: idx}
}

// Columns returns the ordered column names derived from T's sheet tags.
func (r *Repository[T]) Columns() []string { return r.columns }

// Sheet returns the name of the Google Sheets tab this repository reads/writes.
func (r *Repository[T]) Sheet() string { return r.sheet }

// All reads all data rows (skipping the header row at index 0) and maps each
// row to a T using the sheet-tag column mapping.
func (r *Repository[T]) All() ([]T, error) {
	rows, err := r.api.ReadSheet(r.sheet)
	if err != nil {
		return nil, fmt.Errorf("reading sheet %s: %w", r.sheet, err)
	}
	if len(rows) <= 1 {
		return nil, nil
	}
	result := make([]T, 0, len(rows)-1)
	for _, row := range rows[1:] {
		var item T
		scanRow(row, r.columns, r.indices, &item)
		result = append(result, item)
	}
	return result, nil
}

// Add serializes item to a row and appends it to the sheet. Returns the new
// 1-indexed row number.
func (r *Repository[T]) Add(item T) (int, error) {
	row := buildRow(item, r.columns, r.indices)
	n, err := r.api.AppendRow(r.sheet, row)
	if err != nil {
		return 0, fmt.Errorf("appending to %s: %w", r.sheet, err)
	}
	return n, nil
}

// Update does a read-modify-write: finds the existing row by id (column 0),
// applies patch (column name -> new value as string), then writes back the
// full row via UpdateRow.
func (r *Repository[T]) Update(id string, patch map[string]any) error {
	rows, err := r.api.ReadSheet(r.sheet)
	if err != nil {
		return fmt.Errorf("reading %s for update: %w", r.sheet, err)
	}
	for _, row := range rows {
		if len(row) == 0 || fmt.Sprint(row[0]) != id {
			continue
		}
		// Extend row to full column width if shorter.
		for len(row) < len(r.columns) {
			row = append(row, "")
		}
		// Apply patch.
		for colName, val := range patch {
			if i, ok := r.indices[colName]; ok {
				row[i] = fmt.Sprint(val)
			}
		}
		return r.api.UpdateRow(r.sheet, id, row)
	}
	return fmt.Errorf("row with id %s not found in %s", id, r.sheet)
}

// Delete removes the row with the given id.
func (r *Repository[T]) Delete(id string) error {
	return r.api.DeleteRow(r.sheet, id)
}

// AddJSON implements AnyRepository: unmarshals body into T then calls Add.
func (r *Repository[T]) AddJSON(body []byte) (int, error) {
	var item T
	if err := json.Unmarshal(body, &item); err != nil {
		return 0, fmt.Errorf("decoding JSON for %s: %w", r.sheet, err)
	}
	return r.Add(item)
}

// UpdateJSON implements AnyRepository: unmarshals body as a patch map then
// calls Update.
func (r *Repository[T]) UpdateJSON(id string, body []byte) error {
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		return fmt.Errorf("decoding JSON patch for %s: %w", r.sheet, err)
	}
	return r.Update(id, patch)
}

// DeleteByID implements AnyRepository.
func (r *Repository[T]) DeleteByID(id string) error { return r.Delete(id) }

// --- reflection helpers ---

// columnsOf derives the ordered column name list from the `sheet` struct tags
// of T, in field declaration order. Fields with empty or "-" tags are skipped.
func columnsOf[T any]() []string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	var cols []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("sheet")
		if tag == "" || tag == "-" {
			continue
		}
		// Handle "name,omitempty" style — take only the name part.
		if idx := strings.Index(tag, ","); idx >= 0 {
			tag = tag[:idx]
		}
		cols = append(cols, tag)
	}
	return cols
}

// scanRow populates dst (pointer to a struct) from a Sheets row, mapping each
// column by position to the struct field whose `sheet` tag matches that column
// name. Handles string, float64, and int field kinds.
func scanRow(row []interface{}, columns []string, indices map[string]int, dst any) {
	rv := reflect.ValueOf(dst).Elem()
	rt := rv.Type()

	// Build a reverse map: sheet-tag-name -> field index (once per call is fine
	// for the sizes involved here).
	tagToField := make(map[string]int, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("sheet")
		if tag == "" || tag == "-" {
			continue
		}
		if idx := strings.Index(tag, ","); idx >= 0 {
			tag = tag[:idx]
		}
		tagToField[tag] = i
	}

	for colIdx, colName := range columns {
		if colIdx >= len(row) {
			break
		}
		fieldIdx, ok := tagToField[colName]
		if !ok {
			continue
		}
		cell := row[colIdx]
		fv := rv.Field(fieldIdx)
		s := fmt.Sprint(cell)

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(s)
		case reflect.Float64:
			s2 := strings.ReplaceAll(s, ",", "")
			if v, err := strconv.ParseFloat(s2, 64); err == nil {
				fv.SetFloat(v)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			s2 := strings.ReplaceAll(s, ",", "")
			// Allow "3.0" to parse as int.
			if v, err := strconv.ParseFloat(s2, 64); err == nil {
				fv.SetInt(int64(v))
			}
		}
	}
}

// buildRow builds a Sheets row []interface{} from item, using the ordered
// column list and the struct's sheet tags.
func buildRow(item any, columns []string, indices map[string]int) []interface{} {
	rv := reflect.ValueOf(item)
	rt := rv.Type()

	// Build tag -> field index map.
	tagToField := make(map[string]int, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("sheet")
		if tag == "" || tag == "-" {
			continue
		}
		if idx := strings.Index(tag, ","); idx >= 0 {
			tag = tag[:idx]
		}
		tagToField[tag] = i
	}

	row := make([]interface{}, len(columns))
	for colIdx, colName := range columns {
		fieldIdx, ok := tagToField[colName]
		if !ok {
			row[colIdx] = ""
			continue
		}
		fv := rv.Field(fieldIdx)
		switch fv.Kind() {
		case reflect.String:
			row[colIdx] = fv.String()
		case reflect.Float64:
			row[colIdx] = fmt.Sprintf("%g", fv.Float())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			row[colIdx] = fmt.Sprintf("%d", fv.Int())
		default:
			row[colIdx] = fmt.Sprint(fv.Interface())
		}
	}
	return row
}
