import { Link } from 'react-router-dom'
import MarketingLayout, { Container } from './MarketingLayout.jsx'
import { KICK, SERIF } from '../../data/tokens.js'
import { useDocumentTitle } from '../../data/useDocumentTitle.js'

export default function About() {
  useDocumentTitle('About — Kosh')
  return (
    <MarketingLayout>
      <Container style={{ paddingTop: 64, paddingBottom: 16, textAlign: 'center' }}>
        <div style={{ fontFamily: SERIF, fontSize: 44, color: 'var(--accent)', lineHeight: 1, marginBottom: 20 }}>क</div>
        <div style={{ ...KICK, marginBottom: 14 }}>About</div>
        <h1 style={{ fontFamily: SERIF, fontSize: 38, fontWeight: 600, margin: '0 0 16px', letterSpacing: '-.01em' }}>
          A ledger, not a product
        </h1>
      </Container>

      <Container style={{ maxWidth: 680, paddingTop: 24, paddingBottom: 64 }}>
        <div style={{ fontSize: 15.5, color: 'var(--ink-2)', lineHeight: 1.85 }}>
          <p>
            Kosh started as a spreadsheet — the same one most Indian households already keep, tracking FDs
            in one tab, mutual funds in another, and a running argument with yourself about whether this
            year's tax regime actually saves you money. It got built into an app because the spreadsheet
            kept needing formulas nobody remembered how to write six months later.
          </p>
          <p>
            It's a personal project by <a href="https://jenishjain.in/" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-ink)', fontWeight: 600 }}>Jenish Jain</a>,
            built first to manage his own family's finances — investments, insurance, loans, and the yearly
            tax-planning scramble — in one place instead of five. It's shared publicly because the problem
            it solves isn't unique to one household.
          </p>

          <div style={{ height: 1, background: 'var(--line)', margin: '40px 0' }} />

          <div style={{ ...KICK, marginBottom: 14 }}>Why a Google Sheet, not a database</div>
          <p>
            Every finance app asks you to trust a company with your bank balances, your insurance cover,
            your family's net worth — in exchange for a nicer chart. Kosh doesn't hold that data at all.
            It reads and writes a Google Sheet you own, the same way you'd edit it by hand. Self-hosted,
            the only parties with access to your numbers are you, Google (as your existing spreadsheet
            provider), and — only if you turn it on — Anthropic, for the AI features that parse documents
            and draft tax recommendations from your own data.
          </p>
          <p>
            That's also why it can stay a serious tool without becoming a subscription: there's no server
            bill scaling with your data, because there's no server holding your data.
          </p>

          <div style={{ height: 1, background: 'var(--line)', margin: '40px 0' }} />

          <div style={{ ...KICK, marginBottom: 14 }}>What "done" looks like</div>
          <p>
            Kosh isn't trying to become a bank-linked, auto-categorizing budgeting app — there are good
            ones of those already. It's trying to be the calmest possible view of a family's actual
            holdings: what you have, what it's worth, what you owe, and what to do about taxes before the
            financial year runs out. Investment tracking, tax planning, and CA-ready reporting are built;
            self-hosting documentation and further AI-assisted planning are the current focus.
          </p>
        </div>
      </Container>

      <div style={{ borderTop: '1px solid var(--line)', background: 'var(--surface)' }}>
        <Container style={{ paddingTop: 56, paddingBottom: 56, textAlign: 'center' }}>
          <div style={{ fontFamily: SERIF, fontSize: 22, fontWeight: 600, marginBottom: 20 }}>
            Curious how it's built, or want to run your own copy?
          </div>
          <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexWrap: 'wrap' }}>
            <Link to="/docs" className="btn primary" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}>
              Read the self-host guide
            </Link>
            <a
              href="https://github.com/jenish-jain/kosh" target="_blank" rel="noopener noreferrer"
              className="btn" style={{ textDecoration: 'none', padding: '11px 22px', fontSize: 14 }}
            >
              View source on GitHub
            </a>
          </div>
        </Container>
      </div>
    </MarketingLayout>
  )
}
