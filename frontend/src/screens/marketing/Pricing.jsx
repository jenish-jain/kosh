import { Link } from 'react-router-dom'
import MarketingLayout, { Container } from './MarketingLayout.jsx'
import { KICK, SERIF } from '../../data/tokens.js'
import { useDocumentTitle } from '../../data/useDocumentTitle.js'
import { PRICING, HOSTED_APP_URL } from '../../data/pricing.js'

const inr = n => '₹' + n.toLocaleString('en-IN')

const HOSTED = [
  `Everything in Kosh, for the whole household: up to ${PRICING.maxPeople} people, each with their own Google sign-in`,
  'Your data in a Google Sheet in your own Drive, not our database',
  'Family sharing: read-only links for parents, unlocked with a phone code',
  `AI each month: ${PRICING.quotas.ask} questions to Ask Kosh, ${PRICING.quotas.parse} document scans, ${PRICING.quotas.tax} tax-recommendation runs`,
  'Updates and security fixes handled for you',
]

const SELF = [
  'The same app, on your own Google Cloud project',
  'Free and open source (AGPL-3.0)',
  'Bring your own Anthropic key for the AI features, with no usage limits but your own bill',
  'You run it, update it, and secure it',
]

const FAQ = [
  ['Where is my financial data stored?',
    "In a Google Sheet that Kosh creates in your own Google Drive. Kosh's servers keep only what's needed to run your account: who's in your household, your subscription, this month's AI usage, and an encrypted key that lets Kosh open that one spreadsheet. Your holdings, income and tax figures stay in the Sheet."],
  ['What about the AI features?',
    "When you ask Kosh a question or scan a document, the figures or file needed for that request are sent to Anthropic's Claude API to process it. Nothing is sent unless you use those features."],
  ['What can Kosh see in my Google Drive?',
    'Only the files Kosh itself creates (Google calls this the "drive.file" permission). Nothing else in your Drive is visible to Kosh.'],
  ['What happens if I stop paying?',
    'Your household becomes read-only: you can still open and export everything, and the spreadsheet is yours regardless. Subscribe again any time to pick up where you left off.'],
  ['Can I switch to self-hosting later?',
    'Yes. Point a self-hosted Kosh at the same spreadsheet. Nothing is locked in.'],
  ['Does the trial need a card?',
    `No. You get ${PRICING.trialDays} days of the full plan. If you subscribe during the trial, the first charge waits until it ends.`],
  ['How do I cancel?',
    "From Plan in the app, any time. You keep full access until the end of the period you've paid for."],
]

function Check({ children }) {
  return (
    <li style={{ display: 'flex', gap: 10, fontSize: 14, color: 'var(--ink-2)', lineHeight: 1.6, marginBottom: 10 }}>
      <span style={{ color: 'var(--accent)', fontWeight: 800 }}>✓</span><span>{children}</span>
    </li>
  )
}

const card = { border: '1px solid var(--line)', borderRadius: 12, padding: 28, background: 'var(--surface)' }
const external = HOSTED_APP_URL.startsWith('http')

export default function Pricing() {
  useDocumentTitle('Pricing — Kosh')
  const ctaStyle = { textDecoration: 'none', width: '100%', justifyContent: 'center', padding: '12px 22px', fontSize: 14, marginTop: 8 }
  return (
    <MarketingLayout>
      <Container style={{ paddingTop: 64, paddingBottom: 24, textAlign: 'center' }}>
        <div style={{ ...KICK, marginBottom: 14 }}>Pricing</div>
        <h1 style={{ fontFamily: SERIF, fontSize: 38, fontWeight: 600, margin: '0 0 16px', letterSpacing: '-.01em' }}>
          One plan for the whole family. Or run it yourself, free.
        </h1>
        <p style={{ fontSize: 15, color: 'var(--ink-3)', lineHeight: 1.7, maxWidth: 560, margin: '0 auto' }}>
          Either way, your data lives in a Google Sheet you own.
        </p>
      </Container>

      <Container style={{ paddingTop: 32, paddingBottom: 64 }}>
        <div className="pricing-grid" style={{ display: 'grid', gridTemplateColumns: '1.15fr 1fr', gap: 24, alignItems: 'start' }}>
          <div style={{ ...card, borderColor: 'var(--accent)' }}>
            <div style={KICK}>Hosted Kosh</div>
            <div style={{ fontFamily: SERIF, fontSize: 44, margin: '12px 0 2px' }}>
              {inr(PRICING.monthly)}<span style={{ fontSize: 15, color: 'var(--ink-3)' }}> /month</span>
            </div>
            <div style={{ fontSize: 13.5, color: 'var(--ink-3)', marginBottom: 20 }}>
              or {inr(PRICING.annual)}/year (about 2 months free) · {PRICING.trialDays}-day free trial, no card needed
            </div>
            <ul style={{ listStyle: 'none', padding: 0, margin: '0 0 20px' }}>{HOSTED.map(t => <Check key={t}>{t}</Check>)}</ul>
            {external
              ? <a href={HOSTED_APP_URL} className="btn primary" style={ctaStyle}>Start free trial →</a>
              : <Link to={HOSTED_APP_URL} className="btn primary" style={ctaStyle}>Start free trial →</Link>}
          </div>
          <div style={card}>
            <div style={KICK}>Self-hosted</div>
            <div style={{ fontFamily: SERIF, fontSize: 44, margin: '12px 0 20px' }}>Free</div>
            <ul style={{ listStyle: 'none', padding: 0, margin: '0 0 20px' }}>{SELF.map(t => <Check key={t}>{t}</Check>)}</ul>
            <Link to="/docs" className="btn" style={ctaStyle}>Self-hosting guide →</Link>
          </div>
        </div>
        <p style={{ textAlign: 'center', fontSize: 12.5, color: 'var(--ink-3)', marginTop: 18 }}>Prices in Indian rupees, payable by UPI, card or netbanking.</p>
      </Container>

      <div style={{ borderTop: '1px solid var(--line)', background: 'var(--surface)' }}>
        <Container style={{ paddingTop: 56, paddingBottom: 72, maxWidth: 760 }}>
          <div style={{ ...KICK, marginBottom: 24, textAlign: 'center' }}>Questions</div>
          {FAQ.map(([q, a]) => (
            <div key={q} style={{ padding: '18px 0', borderBottom: '1px solid var(--line)' }}>
              <div style={{ fontSize: 15, fontWeight: 700, marginBottom: 6 }}>{q}</div>
              <div style={{ fontSize: 14, color: 'var(--ink-2)', lineHeight: 1.7 }}>{a}</div>
            </div>
          ))}
        </Container>
      </div>
    </MarketingLayout>
  )
}
