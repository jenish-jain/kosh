package sheets

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	svc           *sheets.Service
	spreadsheetID string
}

func NewClient(credentialsPath, spreadsheetID string) (*Client, error) {
	ctx := context.Background()
	svc, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("creating sheets service: %w", err)
	}
	return &Client{svc: svc, spreadsheetID: spreadsheetID}, nil
}

// ReadSheet returns all rows (including header) from a named sheet.
func (c *Client) ReadSheet(name string) ([][]interface{}, error) {
	resp, err := c.svc.Spreadsheets.Values.Get(c.spreadsheetID, name).Do()
	if err != nil {
		return nil, fmt.Errorf("reading sheet %s: %w", name, err)
	}
	return resp.Values, nil
}

// AppendRow adds a new row to the given sheet. Returns the new row number (1-indexed).
func (c *Client) AppendRow(sheetName string, row []interface{}) (int, error) {
	vr := &sheets.ValueRange{Values: [][]interface{}{row}}
	resp, err := c.svc.Spreadsheets.Values.
		Append(c.spreadsheetID, sheetName, vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return 0, fmt.Errorf("appending to %s: %w", sheetName, err)
	}
	// Parse row number from updatedRange like "MF!A5:J5"
	parts := strings.Split(resp.Updates.UpdatedRange, "!")
	if len(parts) < 2 {
		return 0, nil
	}
	rowStr := strings.TrimFunc(parts[1], func(r rune) bool { return r < '0' || r > '9' })
	n, _ := strconv.Atoi(rowStr)
	return n, nil
}

// UpdateRow updates the row in sheetName where column A equals id.
func (c *Client) UpdateRow(sheetName, id string, row []interface{}) error {
	rows, err := c.ReadSheet(sheetName)
	if err != nil {
		return err
	}
	for i, r := range rows {
		if len(r) > 0 && fmt.Sprint(r[0]) == id {
			rowNum := i + 1
			rangeStr := fmt.Sprintf("%s!A%d", sheetName, rowNum)
			vr := &sheets.ValueRange{Values: [][]interface{}{row}}
			_, err := c.svc.Spreadsheets.Values.
				Update(c.spreadsheetID, rangeStr, vr).
				ValueInputOption("USER_ENTERED").
				Do()
			return err
		}
	}
	return fmt.Errorf("row with id %s not found in %s", id, sheetName)
}

// DeleteRow deletes the row in sheetName where column A equals id.
func (c *Client) DeleteRow(sheetName, id string) error {
	rows, err := c.ReadSheet(sheetName)
	if err != nil {
		return err
	}
	// Find the sheet ID
	meta, err := c.svc.Spreadsheets.Get(c.spreadsheetID).Do()
	if err != nil {
		return err
	}
	var sheetID int64
	for _, sh := range meta.Sheets {
		if sh.Properties.Title == sheetName {
			sheetID = sh.Properties.SheetId
			break
		}
	}
	for i, r := range rows {
		if len(r) > 0 && fmt.Sprint(r[0]) == id {
			rowIndex := int64(i)
			req := &sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{{
					DeleteDimension: &sheets.DeleteDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheetID,
							Dimension:  "ROWS",
							StartIndex: rowIndex,
							EndIndex:   rowIndex + 1,
						},
					},
				}},
			}
			_, err := c.svc.Spreadsheets.BatchUpdate(c.spreadsheetID, req).Do()
			return err
		}
	}
	return fmt.Errorf("row with id %s not found in %s", id, sheetName)
}

// ColStr returns cell[col] as string, empty if missing.
func ColStr(row []interface{}, col int) string {
	if col >= len(row) {
		return ""
	}
	return fmt.Sprint(row[col])
}

// ColFloat returns cell[col] as float64, 0 if missing or unparseable.
func ColFloat(row []interface{}, col int) float64 {
	s := ColStr(row, col)
	v, _ := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	return v
}

// ColInt returns cell[col] as int, 0 if missing.
func ColInt(row []interface{}, col int) int {
	return int(ColFloat(row, col))
}

// EnvOrDefault returns the env var or the default.
func EnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
