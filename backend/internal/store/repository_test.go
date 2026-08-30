package store_test

import (
	"testing"

	"kosh/internal/models"
	"kosh/internal/store"
)

const mfSheetName = "MF"

func seedMFHeader(f *store.FakeSheetsAPI) {
	f.Data[mfSheetName] = [][]interface{}{
		{"id", "name", "plan", "platform", "member", "invested", "current", "sip", "notes"},
	}
}

// --- All() tests ---

func TestAll_EmptySheet_ReturnsNil(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	// No data at all for the sheet — not even a header.
	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	rows, err := repo.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected empty, got %d rows", len(rows))
	}
}

func TestAll_OnlyHeader_ReturnsEmpty(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)
	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	rows, err := repo.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 data rows, got %d", len(rows))
	}
}

func TestAll_WithRows_MapsFields(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)
	f.Data[mfSheetName] = append(f.Data[mfSheetName], []interface{}{
		"mf1", "HDFC Top 100", "Direct", "Zerodha", "J", "10000", "12000", "1000", "good fund",
	})

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	rows, err := repo.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := rows[0]
	if got.ID != "mf1" {
		t.Errorf("ID: want %q, got %q", "mf1", got.ID)
	}
	if got.Name != "HDFC Top 100" {
		t.Errorf("Name: want %q, got %q", "HDFC Top 100", got.Name)
	}
	if got.Invested != 10000 {
		t.Errorf("Invested: want 10000, got %v", got.Invested)
	}
	if got.Current != 12000 {
		t.Errorf("Current: want 12000, got %v", got.Current)
	}
	if got.Notes != "good fund" {
		t.Errorf("Notes: want %q, got %q", "good fund", got.Notes)
	}
}

// --- Add() tests ---

func TestAdd_AppendsCorrectRow(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	item := models.MFRow{
		ID:       "mf42",
		Name:     "Parag Parikh",
		Plan:     "Direct",
		Platform: "Zerodha",
		Member:   "J",
		Invested: 5000,
		Current:  6000,
		SIP:      500,
		Notes:    "",
	}
	rowNum, err := repo.Add(item)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	// Fake: header=index0, appended row=index1 -> AppendRow returns len=2.
	if rowNum != 2 {
		t.Errorf("expected rowNum 2, got %d", rowNum)
	}

	stored := f.Data[mfSheetName]
	if len(stored) != 2 { // header + data
		t.Fatalf("expected 2 rows in fake, got %d", len(stored))
	}
	dataRow := stored[1]
	if len(dataRow) < 2 {
		t.Fatalf("data row too short: %v", dataRow)
	}
	if dataRow[0] != "mf42" {
		t.Errorf("col 0 (id): want %q, got %v", "mf42", dataRow[0])
	}
	if dataRow[1] != "Parag Parikh" {
		t.Errorf("col 1 (name): want %q, got %v", "Parag Parikh", dataRow[1])
	}
}

func TestAdd_LargeFloatDoesNotUseScientificNotation(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	// fmt.Sprintf("%g", 1350000.0) renders "1.35e+06" — with the Sheets API's
	// USER_ENTERED input option that's still a valid number, but it's a
	// confusing thing for a human to see when opening the sheet directly,
	// and the cell keeps that scientific formatting going forward.
	if _, err := repo.Add(models.MFRow{ID: "mf1", Invested: 1350000, Current: 5000000}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	dataRow := f.Data[mfSheetName][1]
	if dataRow[5] != "1350000" {
		t.Errorf("Invested cell = %v, want %q (no scientific notation)", dataRow[5], "1350000")
	}
	if dataRow[6] != "5000000" {
		t.Errorf("Current cell = %v, want %q (no scientific notation)", dataRow[6], "5000000")
	}
}

// --- Update() tests ---

func TestUpdate_ModifiesSpecifiedFields(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)
	f.Data[mfSheetName] = append(f.Data[mfSheetName], []interface{}{
		"mf1", "Old Name", "Direct", "Zerodha", "J", "10000", "12000", "1000", "notes",
	})

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	err := repo.Update("mf1", map[string]any{"name": "New Name", "current": "15000"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	row := f.Data[mfSheetName][1]
	if row[1] != "New Name" {
		t.Errorf("name: want %q, got %v", "New Name", row[1])
	}
	if row[6] != "15000" {
		t.Errorf("current: want %q, got %v", "15000", row[6])
	}
	// Other fields must be unchanged.
	if row[0] != "mf1" {
		t.Errorf("id should be unchanged, got %v", row[0])
	}
	if row[5] != "10000" {
		t.Errorf("invested should be unchanged, got %v", row[5])
	}
}

func TestUpdate_UnknownID_ReturnsError(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	err := repo.Update("nonexistent", map[string]any{"name": "X"})
	if err == nil {
		t.Error("expected error for unknown id, got nil")
	}
}

// --- Delete() tests ---

func TestDelete_RemovesRow(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)
	f.Data[mfSheetName] = append(f.Data[mfSheetName], []interface{}{
		"mf1", "Fund A", "Direct", "Zerodha", "J", "1000", "1100", "100", "",
	})
	f.Data[mfSheetName] = append(f.Data[mfSheetName], []interface{}{
		"mf2", "Fund B", "Direct", "Zerodha", "J", "2000", "2200", "200", "",
	})

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	if err := repo.Delete("mf1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	rows := f.Data[mfSheetName]
	if len(rows) != 2 { // header + mf2
		t.Fatalf("expected 2 rows after delete, got %d", len(rows))
	}
	if rows[1][0] != "mf2" {
		t.Errorf("remaining row id: want %q, got %v", "mf2", rows[1][0])
	}
}

// --- AddJSON / UpdateJSON tests ---

func TestAddJSON_ParsesAndAppends(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	body := []byte(`{"id":"mf99","name":"Axis Bluechip","plan":"Direct","platform":"Zerodha","member":"J","invested":8000,"current":9000,"sip":0,"notes":""}`)

	rowNum, err := repo.AddJSON(body)
	if err != nil {
		t.Fatalf("AddJSON failed: %v", err)
	}
	if rowNum == 0 {
		t.Error("expected non-zero row number")
	}

	allRows, _ := repo.All()
	if len(allRows) != 1 {
		t.Fatalf("expected 1 row via All(), got %d", len(allRows))
	}
	if allRows[0].ID != "mf99" {
		t.Errorf("ID: want %q, got %q", "mf99", allRows[0].ID)
	}
	if allRows[0].Invested != 8000 {
		t.Errorf("Invested: want 8000, got %v", allRows[0].Invested)
	}
}

func TestUpdateJSON_ParsesAndPatches(t *testing.T) {
	f := store.NewFakeSheetsAPI()
	seedMFHeader(f)
	f.Data[mfSheetName] = append(f.Data[mfSheetName], []interface{}{
		"mf1", "Old Name", "Direct", "Zerodha", "J", "5000", "6000", "500", "",
	})

	repo := store.NewRepository[models.MFRow](f, mfSheetName)
	body := []byte(`{"name":"New Name","current":"7000"}`)
	if err := repo.UpdateJSON("mf1", body); err != nil {
		t.Fatalf("UpdateJSON failed: %v", err)
	}

	row := f.Data[mfSheetName][1]
	if row[1] != "New Name" {
		t.Errorf("name: want %q, got %v", "New Name", row[1])
	}
	if row[6] != "7000" {
		t.Errorf("current: want %q, got %v", "7000", row[6])
	}
}
