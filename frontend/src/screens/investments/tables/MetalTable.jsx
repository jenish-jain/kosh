import EditableTable from '../../../components/shared/EditableTable.jsx'
import { fmtINR, fmtDate } from '../../../data/format.js'

export default function MetalTable({ data, rows, all, dirty, setDirty }) {
  const mark = (id, field, val) =>
    setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))

  const columns = [
    {
      key: 'type',
      label: 'Metal',
      render: r => (
        <>
          <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
            <span className={'pill ' + (r.type === 'Gold' ? 'gold' : 'silver')}>{r.type}</span>
          </div>
          <div className="cell-sub">{fmtDate(r.date_purchased)} · {r.place}</div>
        </>
      ),
    },
    all && { key: 'member', label: 'Owner', isMember: true },
    { key: 'grams',       label: 'Grams',       align: 'right', editable: true, format: v => v + ' g' },
    { key: 'buy_rate',    label: 'Buy rate /g',  align: 'right', editable: true, format: fmtINR },
    { key: 'today_price', label: 'Today /g',     align: 'right', editable: true, format: fmtINR },
    {
      key: 'invested_computed',
      label: 'Invested',
      align: 'right',
      compute: (row, dirtyRow) => {
        const grams = dirtyRow.grams    ?? row.grams
        const buy   = dirtyRow.buy_rate ?? row.buy_rate
        return (grams || 0) * (buy || 0)
      },
      format: fmtINR,
    },
    {
      key: 'current_computed',
      label: 'Current value',
      align: 'right',
      compute: (row, dirtyRow) => {
        const grams = dirtyRow.grams       ?? row.grams
        const today = dirtyRow.today_price ?? row.today_price
        return (grams || 0) * (today || 0)
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
        label: `${rows.length} holding${rows.length !== 1 ? 's' : ''}`,
        invFn: r => (r.grams || 0) * (r.buy_rate    || 0),
        curFn: r => (r.grams || 0) * (r.today_price || 0),
      }}
    />
  )
}
