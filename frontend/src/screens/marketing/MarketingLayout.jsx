import { Link, useLocation } from 'react-router-dom'
import { SERIF } from '../../data/tokens.js'

const NAV = [
  { to: '/features', label: 'Features' },
  { to: '/about', label: 'About' },
  { to: '/docs', label: 'Docs' },
]

// Centered max-width wrapper reused by every marketing page's content
// sections — the header/footer above/below intentionally span full width.
export function Container({ children, style }) {
  return (
    <div style={{ maxWidth: 1080, margin: '0 auto', padding: '0 32px', ...style }}>
      {children}
    </div>
  )
}

export default function MarketingLayout({ children }) {
  const { pathname } = useLocation()

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg)', color: 'var(--ink)', display: 'flex', flexDirection: 'column' }}>
      <header className="marketing-header" style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '18px 32px', borderBottom: '1px solid var(--line)', gap: 16,
      }}>
        <Link to="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: 'var(--ink)', flexShrink: 0 }}>
          <div style={{ fontFamily: SERIF, fontSize: 24, color: 'var(--accent)', lineHeight: 1 }}>क</div>
          <div style={{ fontSize: 15.5, fontWeight: 800, letterSpacing: '-.01em' }}>Kosh</div>
        </Link>
        <nav className="marketing-nav-links" style={{ display: 'flex', alignItems: 'center', gap: 26 }}>
          {NAV.map(n => (
            <Link
              key={n.to}
              to={n.to}
              style={{
                fontSize: 13.5, fontWeight: 600, textDecoration: 'none',
                color: pathname.startsWith(n.to) ? 'var(--ink)' : 'var(--ink-3)',
              }}
            >
              {n.label}
            </Link>
          ))}
        </nav>
        <Link to="/app" className="btn primary sm" style={{ textDecoration: 'none', flexShrink: 0 }}>
          Sign in
        </Link>
      </header>

      <main style={{ flex: 1 }}>{children}</main>

      <footer style={{ borderTop: '1px solid var(--line)', padding: '32px', textAlign: 'center' }}>
        <a
          href="https://jenishjain.in/" target="_blank" rel="noopener noreferrer"
          style={{ display: 'inline-block', fontSize: 10.5, letterSpacing: '.04em', color: 'var(--ink-3)', opacity: .6, textDecoration: 'none' }}
        >
          Developed by Jenish Jain
        </a>
      </footer>
    </div>
  )
}
