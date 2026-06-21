import { EditCell, MemberTag } from '../Primitives.jsx'
import TotalsBar from './TotalsBar.jsx'

// ── EditableTable ────────────────────────────────────────────
// A reusable table component that replaces the near-identical
// table components in Investments.jsx and Expenses.jsx.
//
// Props:
//   rows          — array of row objects with an `id` field
//   columns       — array of column config objects (see below)
//   dirty         — { [rowId]: { [fieldKey]: newValue } }
//   onMark        — (id, field, value) => void
//   totals        — optional { label, invFn, curFn } where invFn/curFn are (row) => number
//   showOwner     — boolean — whether to render member columns
//   memberOf      — (id) => member object — needed when showOwner=true and a column has isMember:true
//   emptyMessage  — optional string when rows is empty
//
// Column config shape:
//   key       — field name on the row object
//   label     — header label
//   align     — 'left' | 'right' (default 'left')
//   editable  — if true, renders EditCell
//   format    — (v) => string — format function for EditCell display
//   render    — (row) => JSX — custom render for non-editable cells
//   compute   — (row, dirtyRow) => number — for computed columns
//   isMember  — marks the owner column; hidden when showOwner=false
//   isPill    — if true and render is provided, just render the content
export default function EditableTable({
  rows,
  columns,
  dirty = {},
  onMark,
  totals,
  showOwner = true,
  memberOf,
  emptyMessage,
}) {
  const visibleCols = columns.filter(col => showOwner || !col.isMember)

  const inv = totals ? rows.reduce((a, row) => a + (totals.invFn ? totals.invFn(row) : 0), 0) : 0
  const cur = totals ? rows.reduce((a, row) => a + (totals.curFn ? totals.curFn(row) : 0), 0) : 0

  if (rows.length === 0) {
    return (
      <div style={{ fontStyle: 'italic', color: 'var(--ink-3)', padding: '24px 0', fontSize: 13.5 }}>
        {emptyMessage || 'No entries yet.'}
      </div>
    )
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead>
          <tr>
            {visibleCols.map(col => (
              <th key={col.key} className={col.align === 'right' ? 'r' : undefined}>
                {col.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map(row => {
            const dirtyRow = dirty[row.id] || {}
            return (
              <tr key={row.id}>
                {visibleCols.map(col => {
                  if (col.isMember) {
                    // Member column — render MemberTag
                    const member = memberOf ? memberOf(row[col.key]) : null
                    return (
                      <td key={col.key}>
                        <MemberTag member={member} />
                      </td>
                    )
                  }

                  if (col.editable) {
                    // Editable column — render EditCell
                    const rawVal = col.compute
                      ? col.compute(row, dirtyRow)
                      : (dirtyRow[col.key] ?? row[col.key])
                    const currentVal = dirtyRow[col.key] ?? rawVal
                    return (
                      <td key={col.key} className={col.align === 'right' ? 'r' : undefined}>
                        <EditCell
                          value={currentVal}
                          type="number"
                          format={col.format}
                          align={col.align === 'right' ? 'right' : undefined}
                          onChange={v => onMark && onMark(row.id, col.key, v)}
                        />
                      </td>
                    )
                  }

                  if (col.render) {
                    // Custom render column
                    return (
                      <td key={col.key} className={col.align === 'right' ? 'r' : undefined}>
                        {col.render(row)}
                      </td>
                    )
                  }

                  if (col.compute) {
                    // Computed non-editable column
                    const computed = col.compute(row, dirtyRow)
                    const display = col.format ? col.format(computed) : computed
                    return (
                      <td key={col.key} className={col.align === 'right' ? 'r num' : 'num'}>
                        {display}
                      </td>
                    )
                  }

                  // Plain value column
                  return (
                    <td key={col.key} className={col.align === 'right' ? 'r' : undefined}>
                      {col.format ? col.format(row[col.key]) : row[col.key]}
                    </td>
                  )
                })}
              </tr>
            )
          })}
        </tbody>
      </table>
      {totals && (
        <TotalsBar
          inv={inv}
          cur={cur}
          label={totals.label}
          curLabel={totals.curLabel}
        />
      )}
    </div>
  )
}
