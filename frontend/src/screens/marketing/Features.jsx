import { Link } from 'react-router-dom'
import MarketingLayout, { Container } from './MarketingLayout.jsx'
import { KICK, SERIF } from '../../data/tokens.js'
import { useDocumentTitle } from '../../data/useDocumentTitle.js'

const MAJOR = [
  {
    kicker: 'Net worth',
    title: 'One number that actually means something',
    body: 'Total assets minus loans, tracked over time, broken down by asset class and by family member. No more adding up a dozen apps in your head before you know where you stand.',
    img: '/screenshots/dashboard.webp',
    alt: 'Kosh dashboard with total net worth, a 12-month trend chart, asset allocation, and per-member breakdown',
  },
  {
    kicker: 'Investments',
    title: 'Mutual funds, stocks, gold, FDs, and NPS in one ledger',
    body: 'Every holding in one place with invested vs. current value and gain at a glance — instead of five different apps and a spreadsheet nobody keeps up to date.',
    img: '/screenshots/investments.webp',
    alt: 'Kosh investments tab showing a mutual fund holdings table with invested, current value, and gain columns',
  },
  {
    kicker: 'Tax planning',
    title: 'A real tax advisor for the year ahead, not just last year\'s numbers',
    body: 'Compare old vs. new regime instantly, see what\'s maturing this financial year, and get AI-generated, fact-grounded tax-saving recommendations — reviewed against your actual holdings, not generic advice. Even the tax slabs themselves are data you can update and approve when a Budget changes them, not hardcoded numbers waiting on an app update.',
    img: '/screenshots/tax.webp',
    alt: 'Kosh tax tab showing regime comparison, tax payable, advance tax schedule, and AI tax-saving recommendations',
  },
  {
    kicker: 'For your CA',
    title: 'A proper report, not a screen-share',
    body: 'Pick the sections you want — investments, insurance, loans, income, a net worth summary — and download a clean PDF or Excel file shaped for handing to an accountant, not a raw data dump.',
    img: '/screenshots/export.webp',
    alt: 'Kosh export dialog with checkboxes for each report section and a PDF/Excel format toggle',
  },
]

const MINOR = [
  {
    title: 'SIPs & bills, on schedule',
    body: 'Every active SIP, EMI, and insurance premium with its next due date — one screen instead of five reminders.',
  },
  {
    title: 'Income tracking',
    body: 'Payslip-by-payslip gross, net, and deductions — feeds straight into the tax calculations, no manual re-entry.',
  },
  {
    title: 'Built for a family',
    body: 'Track holdings per member or as a household, and see how much tax shifting income to a lower-slab parent could save.',
  },
  {
    title: 'AI-assisted document upload',
    body: 'Drop in an FD certificate, insurance policy, or NPS statement — Claude extracts the details, you review before saving.',
  },
  {
    title: 'Your Google Sheet is the database',
    body: 'No third-party server holds your financial data. Self-host it, and it\'s genuinely just your spreadsheet with a nice front end.',
  },
  {
    title: 'Works properly on your phone',
    body: 'The sidebar becomes a bottom tab bar, every panel is full-width — not a desktop app squeezed onto a small screen.',
  },
]

function FeatureRow({ f, reverse }) {
  const text = (
    <div key="text" className="marketing-row-text" style={{ order: reverse ? 2 : 1 }}>
      <div style={{ ...KICK, marginBottom: 10 }}>{f.kicker}</div>
      <div style={{ fontFamily: SERIF, fontSize: 26, fontWeight: 600, lineHeight: 1.25, marginBottom: 14 }}>
        {f.title}
      </div>
      <div style={{ fontSize: 14, color: 'var(--ink-2)', lineHeight: 1.75 }}>{f.body}</div>
    </div>
  )
  const image = (
    <div key="image" className="marketing-row-image" style={{ order: reverse ? 1 : 2 }}>
      <div style={{
        border: '1px solid var(--line)', borderRadius: 10, overflow: 'hidden',
        boxShadow: 'var(--shadow-lg)',
      }}>
        <img src={f.img} alt={f.alt} style={{ display: 'block', width: '100%', height: 'auto' }} loading="lazy" />
      </div>
    </div>
  )
  return (
    <div className="marketing-grid-2" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 56, alignItems: 'center' }}>
      {text}
      {image}
    </div>
  )
}

export default function Features() {
  useDocumentTitle('Features — Kosh')
  return (
    <MarketingLayout>
      <Container style={{ paddingTop: 64, paddingBottom: 8, textAlign: 'center' }}>
        <div style={{ ...KICK, marginBottom: 14 }}>Features</div>
        <h1 style={{ fontFamily: SERIF, fontSize: 38, fontWeight: 600, margin: '0 0 16px', letterSpacing: '-.01em' }}>
          Everything a family's finances actually need
        </h1>
        <p style={{ fontSize: 15, color: 'var(--ink-3)', lineHeight: 1.7, maxWidth: 560, margin: '0 auto' }}>
          Not a budgeting app bolted onto a bank feed — a ledger you fill in and fully own,
          built around how Indian households actually track investments and tax.
        </p>
      </Container>

      <Container style={{ paddingTop: 64, paddingBottom: 64, display: 'flex', flexDirection: 'column', gap: 88 }}>
        {MAJOR.map((f, i) => <FeatureRow key={f.kicker} f={f} reverse={i % 2 === 1} />)}
      </Container>

      <div style={{ borderTop: '1px solid var(--line)', background: 'var(--surface)' }}>
        <Container style={{ paddingTop: 64, paddingBottom: 64 }}>
          <div style={{ ...KICK, marginBottom: 32, textAlign: 'center' }}>And the rest</div>
          <div className="marketing-grid-3" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '36px 40px' }}>
            {MINOR.map(m => (
              <div key={m.title}>
                <div style={{ fontSize: 14.5, fontWeight: 700, marginBottom: 8 }}>{m.title}</div>
                <div style={{ fontSize: 13.5, color: 'var(--ink-3)', lineHeight: 1.7 }}>{m.body}</div>
              </div>
            ))}
          </div>
        </Container>
      </div>

      <Container style={{ paddingTop: 72, paddingBottom: 88, textAlign: 'center' }}>
        <div style={{ fontFamily: SERIF, fontSize: 26, fontWeight: 600, marginBottom: 16 }}>
          See it with your own data, or the demo's.
        </div>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link to="/app" className="btn primary" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            Try the demo →
          </Link>
          <Link to="/docs" className="btn" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
            Self-host it
          </Link>
        </div>
      </Container>
    </MarketingLayout>
  )
}
