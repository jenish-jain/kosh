import { useAuth } from './auth/useAuth.js'
import { DataProvider } from './data/context.jsx'
import AppShell from './components/layout/AppShell.jsx'
import LoadingScreen from './components/layout/LoadingScreen.jsx'
import Login from './screens/Login.jsx'
import HostedGate from './screens/hosted/HostedGate.jsx'

// The actual product — everything that used to be App.jsx before the public
// marketing site (Home/Features/About/Docs) needed real routes to live
// alongside it. Mounted at /app/* by App.jsx; internal screen switching
// inside AppShell stays state-driven, not route-driven.
export default function AuthenticatedApp() {
  const { authState, user, clientId, demoAvailable, aiEnabled, shareEnabled, mode, devLogin, handleLogin, handleDemo, handleLogout } = useAuth()

  if (authState === 'checking') return <LoadingScreen />
  if (authState === 'login') return (
    <Login clientId={clientId} demoAvailable={demoAvailable} onDemo={handleDemo} onLogin={handleLogin}
      hosted={mode === 'hosted'} devLogin={devLogin} />
  )
  const shell = (household, reloadHousehold, operator) => (
    <DataProvider clientId={clientId}>
      <AppShell user={user} onLogout={clientId ? handleLogout : null} aiEnabled={aiEnabled}
        shareEnabled={shareEnabled && !user?.demo} household={household}
        reloadHousehold={reloadHousehold} devLogin={devLogin} operator={operator} />
    </DataProvider>
  )
  // Hosted: the account must belong to a (connected) household first. Demo
  // sessions skip straight to the sample data, as when self-hosted.
  if (mode === 'hosted' && !user?.demo) return (
    <HostedGate user={user} clientId={clientId} devLogin={devLogin} onSignOut={handleLogout}>
      {(household, reload, operator) => shell(household, reload, operator)}
    </HostedGate>
  )
  return shell(null)
}
