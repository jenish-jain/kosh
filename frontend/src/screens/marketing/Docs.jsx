import MarketingLayout, { Container } from './MarketingLayout.jsx'
import { KICK, SERIF } from '../../data/tokens.js'
import { useDocumentTitle } from '../../data/useDocumentTitle.js'

const SECTIONS = [
  { id: 'quick-start', label: 'Quick start' },
  { id: 'your-sheet', label: 'Connect your own Google Sheet' },
  { id: 'sign-in', label: 'Adding Google Sign-In' },
  { id: 'ai', label: 'AI features' },
  { id: 'deploying', label: 'Deploying' },
]

function Code({ children }) {
  return (
    <pre style={{
      background: 'var(--surface-2)', border: '1px solid var(--line)', borderRadius: 6,
      padding: '12px 16px', fontSize: 12.5, lineHeight: 1.7, overflowX: 'auto',
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
    }}>
      {children}
    </pre>
  )
}

function Section({ id, title, children }) {
  return (
    <section id={id} style={{ paddingTop: 48, scrollMarginTop: 90 }}>
      <div style={{ fontFamily: SERIF, fontSize: 22, fontWeight: 600, marginBottom: 16 }}>{title}</div>
      <div style={{ fontSize: 14.5, color: 'var(--ink-2)', lineHeight: 1.8 }}>{children}</div>
    </section>
  )
}

export default function Docs() {
  useDocumentTitle('Documentation — Kosh')
  return (
    <MarketingLayout>
      <Container style={{ paddingTop: 64, paddingBottom: 16, textAlign: 'center' }}>
        <div style={{ ...KICK, marginBottom: 14 }}>Documentation</div>
        <h1 style={{ fontFamily: SERIF, fontSize: 38, fontWeight: 600, margin: '0 0 16px', letterSpacing: '-.01em' }}>
          Self-hosting Kosh
        </h1>
        <p style={{ fontSize: 15, color: 'var(--ink-3)', lineHeight: 1.7, maxWidth: 560, margin: '0 auto' }}>
          Runs entirely on infrastructure you control — your own Google account, your own spreadsheet,
          your own server. This is the overview; the{' '}
          <a href="https://github.com/jenish-jain/kosh/blob/master/docs/SELF_HOSTING.md" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-ink)', fontWeight: 600 }}>
            full reference guide
          </a>{' '}
          on GitHub has every tab schema and environment variable in detail.
        </p>
      </Container>

      <Container style={{ maxWidth: 680, paddingTop: 32, paddingBottom: 72 }}>
        <div style={{ border: '1px solid var(--line)', borderRadius: 8, padding: '18px 20px', marginBottom: 8 }}>
          <div style={{ ...KICK, marginBottom: 10 }}>On this page</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {SECTIONS.map(s => (
              <a key={s.id} href={`#${s.id}`} style={{ fontSize: 13.5, fontWeight: 600, color: 'var(--ink-2)', textDecoration: 'none' }}>
                {s.label}
              </a>
            ))}
          </div>
        </div>

        <Section id="quick-start" title="Quick start">
          <p>
            Local dev mode needs nothing beyond the repo itself — it serves sample data and skips auth
            entirely, so you can try the full UI in under a minute.
          </p>
          <Code>{`git clone https://github.com/jenish-jain/kosh.git
cd kosh
make install   # first time only — Go + npm dependencies
make dev       # Go API on :8080, Vite on :5173`}</Code>
          <p>
            Open <strong>localhost:5173</strong>. Nothing you do here touches a real spreadsheet —
            mutations are no-ops in dev mode.
          </p>
        </Section>

        <Section id="your-sheet" title="Connect your own Google Sheet">
          <p>This is the step that turns Kosh from a demo into your actual ledger.</p>
          <ol style={{ paddingLeft: 20, margin: 0 }}>
            <li style={{ marginBottom: 10 }}>Create a spreadsheet in Google Sheets — Kosh auto-creates the required tabs (with correct headers) on first run against an empty sheet.</li>
            <li style={{ marginBottom: 10 }}>In <a href="https://console.cloud.google.com" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-ink)' }}>Google Cloud Console</a>, enable the Sheets API and create a service account with Editor access. Download its JSON key as <code>backend/credentials.json</code>.</li>
            <li style={{ marginBottom: 10 }}>Share your spreadsheet with the service account's email address.</li>
            <li>Set <code>SPREADSHEET_ID</code> in <code>backend/.env</code> (the long ID in the sheet's URL), then <code>make dev</code> again — a <code>✓ Connected to Google Sheets</code> line replaces the dev-mode banner.</li>
          </ol>
        </Section>

        <Section id="sign-in" title="Adding Google Sign-In">
          <p>
            Without sign-in, anyone who can reach your server can see and edit your data. For a private
            deploy reachable from the internet, restrict access to specific accounts — yourself and your
            family:
          </p>
          <Code>{`GOOGLE_CLIENT_ID=xxxxxxxxxx.apps.googleusercontent.com
SESSION_SECRET=$(openssl rand -base64 32)
ALLOWED_EMAILS=you@gmail.com,partner@gmail.com`}</Code>
          <p>
            Auth is entirely optional — leave these unset to run open on a trusted machine. Whenever
            sign-in is enabled, the login screen also offers a read-only <strong>Try the demo</strong> mode
            with no Google account needed — no extra setup required, it's what powers this site's own demo.
          </p>
        </Section>

        <Section id="ai" title="AI features">
          <p>
            Two optional capabilities are powered by Claude, both reading only your own data: extracting
            fields from uploaded FD/insurance/NPS documents, and generating tax-saving recommendations
            grounded in your actual holdings. Both need an{' '}
            <a href="https://console.anthropic.com" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-ink)' }}>Anthropic API key</a>:
          </p>
          <Code>{'ANTHROPIC_API_KEY=sk-ant-...'}</Code>
          <p>Without it, both features are simply unavailable — the rest of the app works normally.</p>
        </Section>

        <Section id="deploying" title="Deploying">
          <p>
            Kosh ships as a single Docker image — a static frontend served by the same Go binary that
            serves the API, so there's exactly one process to run anywhere that runs a Dockerfile
            (Cloud Run, Fly.io, Render, a VPS).
          </p>
          <Code>{`docker build -t kosh .
docker run -p 8080:8080 \\
  -e SPREADSHEET_ID=your_spreadsheet_id \\
  -e GOOGLE_CREDENTIALS_B64=$(base64 -i backend/credentials.json) \\
  kosh`}</Code>
          <p>
            The full guide covers Cloud Run specifically (including a GitHub Actions example for
            continuous deployment) and the complete environment variable reference.
          </p>
        </Section>
      </Container>
    </MarketingLayout>
  )
}
