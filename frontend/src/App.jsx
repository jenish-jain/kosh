import { BrowserRouter, Routes, Route } from 'react-router-dom'
import AuthenticatedApp from './AuthenticatedApp.jsx'
import Home from './screens/marketing/Home.jsx'
import Features from './screens/marketing/Features.jsx'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/features" element={<Features />} />
        <Route path="/app/*" element={<AuthenticatedApp />} />
        <Route path="*" element={<Home />} />
      </Routes>
    </BrowserRouter>
  )
}
