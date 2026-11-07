import { useState } from 'react'
import { useData } from '../../data/context.jsx'
import { Toast } from '../Primitives.jsx'
import { Icon } from '../Icons.jsx'
import ExportModal from '../ExportModal.jsx'
import SharingModal from '../SharingModal.jsx'
import HouseholdModal from '../HouseholdModal.jsx'
import PlanModal from '../PlanModal.jsx'
import PlanBanner from '../PlanBanner.jsx'
import FeatureTiersModal from '../FeatureTiersModal.jsx'
import { canUse } from '../../data/billing.js'
import Sidebar from './Sidebar.jsx'
import MemberSwitcher from './MemberSwitcher.jsx'
import LoadingScreen from './LoadingScreen.jsx'
import ErrorScreen from './ErrorScreen.jsx'
import Dashboard from '../../screens/Dashboard.jsx'
import Investments from '../../screens/investments/index.jsx'
import SIPs from '../../screens/SIPs.jsx'
import Income from '../../screens/Income.jsx'
import Expenses from '../../screens/Expenses.jsx'
import Family from '../../screens/Family.jsx'
import Tax from '../../screens/tax/index.jsx'
import { usePinLock } from '../../auth/usePinLock.js'
import PinLock from '../PinLock.jsx'
import PinSettings from '../PinSettings.jsx'
import AskModal from '../AskModal.jsx'

export default function AppShell({ user, onLogout, aiEnabled, shareEnabled, household, reloadHousehold, devLogin, operator }) {
  const { data, loading, error } = useData()
  const [screen, setScreenRaw] = useState(() => localStorage.getItem('kosh.screen') || 'dashboard')
  const [member, setMemberRaw] = useState(() => { const v = localStorage.getItem('kosh.member'); return v === 'null' || !v ? null : v })
  const [toast, setToast] = useState(null)
  const [showPinSettings, setShowPinSettings] = useState(false)
  const [showAsk, setShowAsk] = useState(false)
  const [showExport, setShowExport] = useState(false)
  const [showSharing, setShowSharing] = useState(false)
  const [showHousehold, setShowHousehold] = useState(false)
  const [showPlan, setShowPlan] = useState(false)
  const [showTiers, setShowTiers] = useState(false)
  const pinLock = usePinLock()

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
      case 'income':      return <Income      {...props} />
      case 'expenses':    return <Expenses    {...props} />
      case 'family':      return <Family      {...props} setScreen={setScreen} onSelect={setMember} />
      case 'tax':         return <Tax         {...props} />
      default:            return null
    }
  })()

  return (
    <div className="app">
      <Sidebar
        screen={screen} setScreen={setScreen} data={data} user={user} onLogout={onLogout}
        onOpenPinSettings={() => setShowPinSettings(true)}
        onLockNow={pinLock.pinConfigured ? pinLock.lockNow : null}
        aiEnabled={aiEnabled && canUse(household, 'ask')} onOpenAsk={() => setShowAsk(true)}
      />
      <div className="main">
        <header className="topbar">
          <div className="topbar-spacer" style={{ flex: 1 }} />
          {household && (
            <button className="btn ghost sm" onClick={() => setShowHousehold(true)} title={`Household: ${household.name}`} aria-label="Household members" style={{ marginRight: 4 }}>
              <Icon name="members" size={15} />
            </button>
          )}
          {shareEnabled && (
            <button className="btn ghost sm" onClick={() => setShowSharing(true)} title="Share with family" aria-label="Share with family" style={{ marginRight: 4 }}>
              <Icon name="share" size={15} />
            </button>
          )}
          <button className="btn ghost sm" onClick={() => setShowExport(true)} title="Export report" style={{ marginRight: 10 }}>
            <Icon name="download" size={15} />
          </button>
          <MemberSwitcher member={member} setMember={setMember} members={data?.members} />
        </header>
        <PlanBanner household={household} onOpen={() => setShowPlan(true)} />
        <main className="content fade-in" key={screen}>
          {screenEl}
        </main>
      </div>
      {toast && (
        <Toast message={toast.message} type={toast.type} onDone={() => setToast(null)} />
      )}
      {pinLock.locked && <PinLock onVerify={pinLock.verify} />}
      {showPinSettings && <PinSettings pinLock={pinLock} onClose={() => setShowPinSettings(false)} />}
      {showAsk && <AskModal onClose={() => setShowAsk(false)} />}
      {showHousehold && <HouseholdModal onClose={() => setShowHousehold(false)} showToast={showToast} onLeft={() => window.location.reload()}
        onOpenPlan={() => { setShowHousehold(false); setShowPlan(true) }}
        onOpenOperator={operator ? () => { setShowHousehold(false); setShowTiers(true) } : null} />}
      {showTiers && <FeatureTiersModal showToast={showToast}
        onClose={() => setShowTiers(false)} onSaved={reloadHousehold} />}
      {showPlan && <PlanModal household={household} devLogin={devLogin} showToast={showToast}
        onClose={() => setShowPlan(false)} onChanged={() => { setShowPlan(false); reloadHousehold() }} />}
      {showSharing && <SharingModal onClose={() => setShowSharing(false)} showToast={showToast} canCreate={canUse(household, 'share')} />}
      {showExport && <ExportModal data={data} memberId={member} onClose={() => setShowExport(false)} />}
    </div>
  )
}
