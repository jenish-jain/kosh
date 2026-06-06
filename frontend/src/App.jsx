import { useState } from 'react'
import { DataProvider, useData } from './data/context.jsx'
import { memberTotal, fmtINR, fmtCompact } from './data/helpers.js'
import { Icon } from './components/Icons.jsx'
import { Avatar, Toast } from './components/Primitives.jsx'
import Dashboard from './screens/Dashboard.jsx'
import Investments from './screens/Investments.jsx'
import SIPs from './screens/SIPs.jsx'
import Family from './screens/Family.jsx'
import Tax from './screens/Tax.jsx'

const NAV = [
  { id: 'dashboard',   label: 'Dashboard',      icon: 'dash' },
  { id: 'investments', label: 'Investments',     icon: 'invest' },
  { id: 'sips',        label: 'SIPs & schedule', icon: 'sip' },
  { id: 'family',      label: 'Family',          icon: 'members' },
  { id: 'tax',         label: 'Tax',             icon: 'tax' },
]

function Sidebar({ screen, setScreen, data }) {
  const household = data ? memberTotal(data, null) : 0
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
        <div className="nav-label" style={{ marginTop: 18 }}>Soon</div>
        <button className="nav-item" style={{ opacity: .45, cursor: 'default' }} disabled>
          <Icon name="expense" size={18} /> Expenses
          <span style={{ flex: 1 }} />
          <span className="pill neutral" style={{ fontSize: 10, padding: '2px 7px' }}>soon</span>
        </button>
      </nav>
      <div className="sidebar-foot">
        <div className="sidebar-net">
          <div className="stat-label">Household net worth</div>
          {data ? (
            <>
              <div className="net-val num">{fmtINR(household)}</div>
              <div className="net-sub">{data.members?.length || 0} members · local</div>
            </>
          ) : (
            <div className="net-val" style={{ opacity: .4 }}>—</div>
          )}
        </div>
      </div>
    </aside>
  )
}

function MemberSwitcher({ member, setMember, members }) {
  return (
    <div className="memberbar">
      <button
        className={'member-chip' + (!member ? ' active' : '')}
        onClick={() => setMember(null)}
      >
        <span className="avatar" style={{ background: 'var(--ink)', width: 26, height: 26, flexBasis: 26, fontSize: 12 }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M16 19c0-2.2-1.8-4-4-4s-4 1.8-4 4M12 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6" />
          </svg>
        </span>
        Whole family
      </button>
      {(members || []).map(m => (
        <button
          key={m.id}
          className={'member-chip' + (member === m.id ? ' active' : '')}
          onClick={() => setMember(m.id)}
        >
          <Avatar member={m} />
          {m.name}
        </button>
      ))}
    </div>
  )
}

function LoadingScreen() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 16, color: 'var(--ink-3)' }}>
      <div style={{ fontFamily: 'var(--serif)', fontSize: 64, color: 'var(--accent)', lineHeight: 1 }}>क</div>
      <div style={{ fontSize: 14, fontWeight: 600, letterSpacing: '.08em', textTransform: 'uppercase' }}>Loading…</div>
    </div>
  )
}

function ErrorScreen({ message }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 12, padding: 32 }}>
      <div style={{ fontFamily: 'var(--serif)', fontSize: 40, color: 'var(--neg)' }}>!</div>
      <div style={{ fontSize: 16, fontWeight: 700 }}>Could not connect to backend</div>
      <div style={{ fontSize: 13, color: 'var(--ink-3)', maxWidth: 380, textAlign: 'center', lineHeight: 1.6 }}>{message}</div>
      <button className="btn" onClick={() => window.location.reload()}>
        <Icon name="refresh" size={15} /> Retry
      </button>
    </div>
  )
}

function AppShell() {
  const { data, loading, error } = useData()
  const [screen, setScreenRaw] = useState(() => localStorage.getItem('kosh.screen') || 'dashboard')
  const [member, setMemberRaw] = useState(() => { const v = localStorage.getItem('kosh.member'); return v === 'null' || !v ? null : v })
  const [toast, setToast] = useState(null)

  const setScreen = s => { setScreenRaw(s); localStorage.setItem('kosh.screen', s) }
  const setMember = m => { setMemberRaw(m); localStorage.setItem('kosh.member', m ?? 'null') }

  const showToast = (message, type = 'success') => setToast({ message, type })

  if (loading) return <LoadingScreen />
  if (error) return <ErrorScreen message={error} />

  const screenEl = (() => {
    const props = { data, memberId: member, showToast }
    switch (screen) {
      case 'dashboard':   return <Dashboard   {...props} setScreen={setScreen} onSelectMember={m => { setMember(m); setScreen('family') }} />
      case 'investments': return <Investments {...props} />
      case 'sips':        return <SIPs        {...props} />
      case 'family':      return <Family      {...props} setScreen={setScreen} onSelect={setMember} />
      case 'tax':         return <Tax         {...props} />
      default:            return null
    }
  })()

  return (
    <div className="app">
      <Sidebar screen={screen} setScreen={setScreen} data={data} />
      <div className="main">
        <header className="topbar">
          <div style={{ flex: 1 }} />
          <MemberSwitcher member={member} setMember={setMember} members={data?.members} />
        </header>
        <main className="content fade-in" key={screen}>
          {screenEl}
        </main>
      </div>
      {toast && (
        <Toast message={toast.message} type={toast.type} onDone={() => setToast(null)} />
      )}
    </div>
  )
}

export default function App() {
  return (
    <DataProvider>
      <AppShell />
    </DataProvider>
  )
}
