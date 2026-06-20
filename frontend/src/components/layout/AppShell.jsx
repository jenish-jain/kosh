import { useState } from 'react'
import { useData } from '../../data/context.jsx'
import { Toast } from '../Primitives.jsx'
import Sidebar from './Sidebar.jsx'
import MemberSwitcher from './MemberSwitcher.jsx'
import LoadingScreen from './LoadingScreen.jsx'
import ErrorScreen from './ErrorScreen.jsx'
import Dashboard from '../../screens/Dashboard.jsx'
import Investments from '../../screens/investments/index.jsx'
import SIPs from '../../screens/SIPs.jsx'
import Expenses from '../../screens/Expenses.jsx'
import Family from '../../screens/Family.jsx'
import Tax from '../../screens/Tax.jsx'

export default function AppShell({ user, onLogout }) {
  const { data, loading, error } = useData()
  const [screen, setScreenRaw] = useState(() => localStorage.getItem('kosh.screen') || 'dashboard')
  const [member, setMemberRaw] = useState(() => { const v = localStorage.getItem('kosh.member'); return v === 'null' || !v ? null : v })
  const [toast, setToast] = useState(null)

  const setScreen = s => { setScreenRaw(s); localStorage.setItem('kosh.screen', s) }
  const setMember = m => { setMemberRaw(m); localStorage.setItem('kosh.member', m ?? 'null') }

  const showToast = (message, type = 'success') => setToast({ message, type })

  if (loading && !data) return <LoadingScreen />
  if (error) return <ErrorScreen message={error} />

  const screenEl = (() => {
    const props = { data, memberId: member, showToast }
    switch (screen) {
      case 'dashboard':   return <Dashboard   {...props} setScreen={setScreen} onSelectMember={m => { setMember(m); setScreen('family') }} />
      case 'investments': return <Investments {...props} />
      case 'sips':        return <SIPs        {...props} />
      case 'expenses':    return <Expenses    {...props} />
      case 'family':      return <Family      {...props} setScreen={setScreen} onSelect={setMember} />
      case 'tax':         return <Tax         {...props} />
      default:            return null
    }
  })()

  return (
    <div className="app">
      <Sidebar screen={screen} setScreen={setScreen} data={data} user={user} onLogout={onLogout} />
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
