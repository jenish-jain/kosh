import EditableTable from '../../../components/shared/EditableTable.jsx'
import { fmtINR } from '../../../data/format.js'

export default function StockTable({ data, rows, all, dirty, setDirty }) {
  const mark = (id, field, val) =>
    setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))

  const columns = [
    {
      key: 'name',
      label: 'Stock',
      render: r => (
        <>
          <div className="cell-strong">{r.name}</div>
          <div className="cell-sub">{r.ticker}</div>
        </>
      ),
    },
    all && { key: 'member', label: 'Owner', isMember: true },
    { key: 'qty',        label: 'Qty',           align: 'right', editable: true },
    { key: 'avg_price',  label: 'Avg price',      align: 'right', editable: true, format: fmtINR },
    { key: 'last_price', label: 'Last price',     align: 'right', editable: true, format: fmtINR },
    {
      key: 'invested_computed',
      label: 'Invested',
      align: 'right',
      compute: (row, dirtyRow) => {
        const qty = dirtyRow.qty       ?? row.qty
        const avg = dirtyRow.avg_price ?? row.avg_price
        return (qty || 0) * (avg || 0)
      },
      format: fmtINR,
    },
    {
      key: 'current_computed',
      label: 'Current value',
      align: 'right',
      compute: (row, dirtyRow) => {
        const qty = dirtyRow.qty        ?? row.qty
        const ltp = dirtyRow.last_price ?? row.last_price
        return (qty || 0) * (ltp || 0)
      },
      format: v => <span className="cell-strong">{fmtINR(v)}</span>,
    },
  ].filter(Boolean)

  return (
    <EditableTable
      rows={rows}
      columns={columns}
      dirty={dirty}
      onMark={mark}
      showOwner={all}
      memberOf={id => data.members?.find(m => m.id === id)}
      totals={{
        label: `${rows.length} stock${rows.length !== 1 ? 's' : ''}`,
        invFn: r => (r.qty || 0) * (r.avg_price  || 0),
        curFn: r => (r.qty || 0) * (r.last_price || 0),
      }}
    />
  )
}
