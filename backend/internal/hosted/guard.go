package hosted

import (
	"errors"
	"sync"

	"golang.org/x/oauth2"

	"kosh/internal/store"
	sh "kosh/sheets"
)

// guardedSheets wraps a household's Sheets client so a revoked Google grant
// surfaces as ErrReconnect (HTTP 409 reconnect_required) instead of a
// generic read failure — callers must not mistake "we lost access" for
// "the spreadsheet is empty".
type guardedSheets struct {
	inner     store.SheetsAPI
	once      sync.Once
	onRevoked func()
}

func (g *guardedSheets) check(err error) error {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) && re.ErrorCode == "invalid_grant" {
		g.once.Do(g.onRevoked)
		return ErrReconnect
	}
	return err
}

func (g *guardedSheets) ReadSheet(name string) ([][]interface{}, error) {
	rows, err := g.inner.ReadSheet(name)
	return rows, g.check(err)
}

func (g *guardedSheets) AppendRow(sheet string, row []interface{}) (int, error) {
	n, err := g.inner.AppendRow(sheet, row)
	return n, g.check(err)
}

func (g *guardedSheets) UpdateRow(sheet, id string, row []interface{}) error {
	return g.check(g.inner.UpdateRow(sheet, id, row))
}

func (g *guardedSheets) DeleteRow(sheet, id string) error {
	return g.check(g.inner.DeleteRow(sheet, id))
}

// ReadSheets implements store.BatchReader when the inner client does.
func (g *guardedSheets) ReadSheets(names []string) (map[string][][]interface{}, error) {
	br, ok := g.inner.(store.BatchReader)
	if !ok {
		return nil, errors.New("batch reads not supported")
	}
	rows, err := br.ReadSheets(names)
	return rows, g.check(err)
}

// EnsureTabs passes provisioning through when the inner client supports it.
func (g *guardedSheets) EnsureTabs(tabs []sh.TabDef) error {
	e, ok := g.inner.(interface{ EnsureTabs([]sh.TabDef) error })
	if !ok {
		return nil
	}
	return g.check(e.EnsureTabs(tabs))
}
