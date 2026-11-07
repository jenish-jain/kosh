import { BrowserRouter, Routes, Route } from 'react-router-dom'
import AuthenticatedApp from './AuthenticatedApp.jsx'
import Home from './screens/marketing/Home.jsx'
import Features from './screens/marketing/Features.jsx'
import About from './screens/marketing/About.jsx'
import Docs from './screens/marketing/Docs.jsx'
import Pricing from './screens/marketing/Pricing.jsx'
import SharedReport from './screens/SharedReport.jsx'
import JoinRedirect from './screens/hosted/JoinRedirect.jsx'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/features" element={<Features />} />
        <Route path="/about" element={<About />} />
        <Route path="/docs" element={<Docs />} />
        <Route path="/pricing" element={<Pricing />} />
        <Route path="/app/*" element={<AuthenticatedApp />} />
        <Route path="/s" element={<SharedReport />} />
        <Route path="/s/:token" element={<SharedReport />} />
        <Route path="/join" element={<JoinRedirect />} />
        <Route path="*" element={<Home />} />
      </Routes>
    </BrowserRouter>
  )
}
