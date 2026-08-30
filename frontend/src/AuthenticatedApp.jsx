import { useAuth } from './auth/useAuth.js'
import { DataProvider } from './data/context.jsx'
import AppShell from './components/layout/AppShell.jsx'
import LoadingScreen from './components/layout/LoadingScreen.jsx'
import Login from './screens/Login.jsx'

// The actual product — everything that used to be App.jsx before the public
// marketing site (Home/Features/About/Docs) needed real routes to live
// alongside it. Mounted at /app/* by App.jsx; internal screen switching
// inside AppShell stays state-driven, not route-driven.
export default function AuthenticatedApp() {
  const { authState, user, clientId, demoAvailable, aiEnabled, handleLogin, handleDemo, handleLogout } = useAuth()

  if (authState === 'checking') return <LoadingScreen />
  if (authState === 'login') return (
    <Login clientId={clientId} demoAvailable={demoAvailable} onDemo={handleDemo} onLogin={handleLogin} />
  )
  return (
    <DataProvider clientId={clientId}>
      <AppShell user={user} onLogout={clientId ? handleLogout : null} aiEnabled={aiEnabled} />
    </DataProvider>
  )
}
