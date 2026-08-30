import { Link } from 'react-router-dom'
import MarketingLayout, { Container } from './MarketingLayout.jsx'
import { KICK, SERIF } from '../../data/tokens.js'
import { useDocumentTitle } from '../../data/useDocumentTitle.js'

const VALUE_PROPS = [
  {
    title: 'Your data stays yours',
    body: 'Kosh reads and writes a Google Sheet you own — no third-party database holding your financial life. Self-host it, or run it privately for your own family.',
  },
  {
    title: 'Everything in one ledger',
    body: 'Mutual funds, stocks, gold, FDs, NPS, insurance, loans, income, and tax planning — one calm view instead of a dozen apps and spreadsheets.',
  },
  {
    title: 'Built for a family, not just you',
    body: 'Track holdings per member or as a household, split income across lower tax slabs, and see who owns what at a glance.',
  },
]

export default function Home() {
  useDocumentTitle('Kosh — Family Wealth')
  return (
    <MarketingLayout>
      {/* Hero */}
      <Container style={{ paddingTop: 72, paddingBottom: 56, textAlign: 'center' }}>
        <div style={{ fontFamily: SERIF, fontSize: 56, color: 'var(--accent)', lineHeight: 1, marginBottom: 20 }}>क</div>
        <h1 style={{
          fontFamily: SERIF, fontSize: 44, fontWeight: 600, letterSpacing: '-.01em',
          lineHeight: 1.15, margin: '0 0 18px', maxWidth: 640, marginLeft: 'auto', marginRight: 'auto',
        }}>
          Your family's wealth, in one calm place.
        </h1>
        <p style={{ fontSize: 16, color: 'var(--ink-3)', lineHeight: 1.7, maxWidth: 520, margin: '0 auto 32px' }}>
          A quiet ledger for investments, tax planning, and net worth — backed by a Google Sheet you own,
          not a company's database.
        </p>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link to="/app" className="btn primary" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            Try the demo →
          </Link>
          <Link to="/features" className="btn" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            See what it does
          </Link>
        </div>
      </Container>

      {/* Hero screenshot */}
      <Container style={{ paddingBottom: 72 }}>
        <div style={{
          border: '1px solid var(--line)', borderRadius: 12, overflow: 'hidden',
          boxShadow: 'var(--shadow-lg)', background: 'var(--surface)',
        }}>
          <img
            src="/screenshots/dashboard.webp"
            alt="Kosh dashboard showing household net worth, asset allocation, and per-member holdings"
            style={{ display: 'block', width: '100%', height: 'auto' }}
            loading="eager"
          />
        </div>
      </Container>

      {/* Value props */}
      <div style={{ borderTop: '1px solid var(--line)', background: 'var(--surface)' }}>
        <Container style={{ paddingTop: 56, paddingBottom: 56 }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 40 }} className="marketing-grid-3">
            {VALUE_PROPS.map(v => (
              <div key={v.title}>
                <div style={{ ...KICK, marginBottom: 10 }}>{v.title}</div>
                <div style={{ fontSize: 14, color: 'var(--ink-2)', lineHeight: 1.7 }}>{v.body}</div>
              </div>
            ))}
          </div>
        </Container>
      </div>

      {/* Footer CTA */}
      <Container style={{ paddingTop: 72, paddingBottom: 88, textAlign: 'center' }}>
        <div style={{ fontFamily: SERIF, fontSize: 26, fontWeight: 600, marginBottom: 16 }}>
          Ready to see your whole financial picture?
        </div>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link to="/app" className="btn primary" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            Try the demo →
          </Link>
          <Link to="/docs" className="btn" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            Read the self-host guide
          </Link>
        </div>
      </Container>
    </MarketingLayout>
  )
}
