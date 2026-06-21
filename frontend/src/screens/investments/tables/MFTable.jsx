import EditableTable from '../../../components/shared/EditableTable.jsx'
import { fmtINR } from '../../../data/format.js'

export default function MFTable({ data, rows, all, dirty, setDirty }) {
  const mark = (id, field, val) =>
    setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))

  const columns = [
    {
      key: 'name',
      label: 'Fund',
      render: r => (
        <>
          <div className="cell-strong">{r.name}</div>
          <div className="cell-sub">{r.plan} · {r.platform}</div>
        </>
      ),
    },
    all && { key: 'member', label: 'Owner', isMember: true },
    { key: 'sip',      label: 'Monthly SIP',   align: 'right', editable: true, format: v => v ? fmtINR(v) : '—' },
    { key: 'invested', label: 'Invested',       align: 'right', editable: true, format: fmtINR },
    { key: 'current',  label: 'Current value',  align: 'right', editable: true, format: v => v ? fmtINR(v) : '—' },
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
        label: `${rows.length} fund${rows.length !== 1 ? 's' : ''}`,
        invFn: r => r.invested || 0,
        curFn: r => r.current  || 0,
      }}
    />
  )
}
