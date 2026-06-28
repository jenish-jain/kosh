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

const ICON_BTN = {
  background: 'none', border: 'none', cursor: 'pointer',
  padding: 6, borderRadius: 8, color: 'var(--ink-3)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  transition: 'color .15s, background .15s',
}

export default function Sidebar({ screen, setScreen, data, user, onLogout, onOpenPinSettings, onLockNow, aiEnabled, onOpenAsk }) {
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
        {aiEnabled && (
          <button className="nav-item ask-nav-item" onClick={onOpenAsk}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            Ask
          </button>
        )}
      </nav>
      {aiEnabled && (
        <button className="ask-kosh-btn" onClick={onOpenAsk}>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          Ask Kosh
        </button>
      )}
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
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 8 }}>
          {onLockNow && (
            <button
              onClick={onLockNow}
              style={ICON_BTN}
              title="Lock now"
              onMouseEnter={e => { e.currentTarget.style.color = 'var(--ink)'; e.currentTarget.style.background = 'var(--border)' }}
              onMouseLeave={e => { e.currentTarget.style.color = 'var(--ink-3)'; e.currentTarget.style.background = 'none' }}
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            </button>
          )}
          <button
            onClick={onOpenPinSettings}
            style={ICON_BTN}
            title="Security settings"
            onMouseEnter={e => { e.currentTarget.style.color = 'var(--ink)'; e.currentTarget.style.background = 'var(--border)' }}
            onMouseLeave={e => { e.currentTarget.style.color = 'var(--ink-3)'; e.currentTarget.style.background = 'none' }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
          </button>
          {user && onLogout && (
            <button onClick={onLogout} style={{ ...ICON_BTN, flex: 1, justifyContent: 'flex-start', gap: 6, fontSize: 12, fontWeight: 600 }}
              onMouseEnter={e => { e.currentTarget.style.color = 'var(--ink)'; e.currentTarget.style.background = 'var(--border)' }}
              onMouseLeave={e => { e.currentTarget.style.color = 'var(--ink-3)'; e.currentTarget.style.background = 'none' }}
            >
              <Icon name="x" size={14} />
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {user.demo ? 'Exit demo' : user.email}
              </span>
            </button>
          )}
        </div>
      </div>
    </aside>
  )
}
