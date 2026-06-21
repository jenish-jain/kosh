import { Icon } from '../Icons.jsx'
import { Avatar } from '../Primitives.jsx'
import { netWorth } from '../../data/aggregate.js'
import { fmtINR } from '../../data/format.js'

const NAV = [
  { id: 'dashboard',   label: 'Dashboard',      icon: 'dash' },
  { id: 'investments', label: 'Investments',     icon: 'invest' },
  { id: 'sips',        label: 'SIPs & schedule', icon: 'sip' },
  { id: 'expenses',    label: 'Expenses',        icon: 'expense' },
  { id: 'family',      label: 'Family',          icon: 'members' },
  { id: 'tax',         label: 'Tax',             icon: 'tax' },
]

export default function Sidebar({ screen, setScreen, data, user, onLogout }) {
  const household = data ? netWorth(data, null) : 0
  return (
    <aside className="sidebar">
      <div className="brand">
        <div className="brand-mark">क</div>
        <div>
          <div className="brand-name">Kosh</div>
          <div className="brand-sub">Family Wealth</div>
        </div>
      </div>
      <nav className="nav">
        <div className="nav-label">Track</div>
        {NAV.map(n => (
          <button
            key={n.id}
            className={'nav-item' + (screen === n.id ? ' active' : '')}
            onClick={() => setScreen(n.id)}
          >
            <Icon name={n.icon} size={18} /> {n.label}
          </button>
        ))}
      </nav>
      <div className="sidebar-foot">
        <div className="sidebar-net">
          <div className="stat-label">Household net worth</div>
          {data ? (
            <>
              <div className="net-val num">{fmtINR(household)}</div>
              <div className="net-sub">{data.members?.length || 0} members · live</div>
            </>
          ) : (
            <div className="net-val" style={{ opacity: .4 }}>—</div>
          )}
        </div>
        {user && onLogout && (
          <button onClick={onLogout} style={{
            marginTop: 10, width: '100%', background: 'none', border: 'none',
            cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8,
            padding: '8px 0', color: 'var(--ink-3)', fontSize: 12, fontWeight: 600,
          }}>
            <Icon name="x" size={14} />
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {user.demo ? 'Exit demo' : user.email}
            </span>
          </button>
        )}
      </div>
    </aside>
  )
}
