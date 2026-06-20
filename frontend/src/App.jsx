import { useAuth } from './auth/useAuth.js'
import { DataProvider } from './data/context.jsx'
import AppShell from './components/layout/AppShell.jsx'
import LoadingScreen from './components/layout/LoadingScreen.jsx'
import Login from './screens/Login.jsx'

export default function App() {
  const { authState, user, clientId, demoAvailable, handleLogin, handleDemo, handleLogout } = useAuth()

  if (authState === 'checking') return <LoadingScreen />
  if (authState === 'login') return (
    <Login clientId={clientId} demoAvailable={demoAvailable} onDemo={handleDemo} onLogin={handleLogin} />
  )
  return (
    <DataProvider clientId={clientId}>
      <AppShell user={user} onLogout={clientId ? handleLogout : null} />
    </DataProvider>
  )
}
