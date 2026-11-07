# Plan: Emergency info + family access (secure sharing)

Goal: let family members (parents, spouse, nominees) open a read-only view of
the household report — and, in an emergency, the encashment info for each
asset — from a link or QR on their phone, **without a password or a Google
account**, while keeping it accessible only to people the owner has
explicitly enrolled.

Two parts:
1. **Emergency info** — per-asset document links and liquidation runbooks
   (the "more details" that get shared).
2. **Family access** — per-recipient secret link + authenticator-app (TOTP)
   code, replacing the passphrase gate from the original version of this plan.

---

## Part 1 — Emergency info: data model

Add two fields to each liquidatable asset type:

- **`doc_link`** — Drive URL to the policy/statement/certificate PDF.
  ✅ **Done for Fixed and Insurance** (commit 72ef5f1): dedicated `doc_link`
  column, populated by the upload flow, link icon in `FixedTable.jsx` /
  `InsuranceTable.jsx`. Still to do: MF, Stocks, Metals.
- **`emergency_steps`** — free-text liquidation runbook: agent/RM name &
  phone, folio/policy/account number, branch/helpline, claim form name,
  documents required (death certificate, KYC, nominee proof), expected
  turnaround.

Where (columns are now derived from `sheet:"..."` struct tags):
- Structs in `backend/internal/models/types.go` (`MF`, `Stock`, `Metal`,
  `Fixed`, `Insurance`)
- Seed data in `backend/dev_data.json`
- Frontend: an expandable "Emergency info" section per row in
  `frontend/src/screens/investments/tables/*`, with a small form for
  `emergency_steps` and `doc_link` (wired to the existing `UploadZone`).

---

## Part 2 — Family access: secret link + authenticator code

### Decisions (confirmed with user 2026-11-07)
- **TOTP (authenticator app), not SMS OTP.** SMS needs a paid provider
  (Twilio etc.), India DLT sender/template registration, sends family phone
  numbers and access events to a third party (contradicts Kosh's privacy
  pitch), and is exposed to SIM-swap. TOTP needs no new service and works
  offline. Cost: one-time in-person enrolment for each family member.
- **Supersedes the earlier passphrase design** (approved 2026-06-08). A
  shared passphrase can't be revoked per person and gets written down; a
  per-recipient TOTP secret can.
- **One link per recipient**, not one shared link. The "whitelist" is a list
  of named recipients ("Mom", "Dad"), each with their own secret, so access
  can be scoped, logged and revoked per person.
- **Live view, not a snapshot** — recipients always see current numbers.

Defaults chosen (change at implementation time if needed):
- Remember-device cookie: **7 days**, then the code is asked again.
- Scope: **per recipient** — sections + member filter set individually
  (e.g. Dad sees only his own holdings).

### Two different artefacts — never confuse them
| | Share link / QR | Enrolment QR |
|---|---|---|
| Content | `https://<host>/s/<random-token>` | `otpauth://totp/Kosh:<name>?secret=…` |
| Grants access alone? | No — identifies the recipient only | No — but link + this = full access |
| How it travels | WhatsApp, print, fridge — fine | **Shown once, scanned in person.** Never sent over chat |

### Infrastructure constraint
Kosh has no database, sessions are stateless HMAC cookies
(`internal/auth/session.go`), and Cloud Run may run multiple instances and
scale to zero. **In-memory rate limits / lockouts silently reset**, so all
security state is persisted in the Sheet (family scale is far below Sheets
write quotas).

### 1. Data model (new Sheet tabs)
- **`Shares`** — one row per recipient: `id`, `name`, `relation`,
  `token_hash` (SHA-256 of the link token; the raw token is never stored),
  `totp_secret_enc` (AES-GCM, key from new env var `SHARE_ENCRYPTION_KEY` —
  anyone with view access to the Sheet must not be able to copy secrets),
  `sections`, `member_scope`, `expires`, `revoked`, `failed_attempts`,
  `locked_until`, `last_totp_step` (replay protection), `created`.
- **`ShareAccess`** — access log: `share_id`, timestamp, outcome
  (ok / bad code / locked / revoked), coarse IP / user agent.

### 2. Backend
- `internal/share` package: TOTP verify per RFC 6238 (30 s step, ±1 step
  clock-drift window, reject steps ≤ `last_totp_step`), AES-GCM secret
  encryption, token generation (32 random bytes, base64url) and hashing.
- **Session scoping:** add a scope/type field to the signed token. Share
  sessions use a separate cookie name, `Path=/s/` + `/api/share/`, and are
  rejected by the existing `protect` middleware, so a recipient can never
  reach `/api/data`, mutations, AI or upload endpoints.
- Owner endpoints (behind `protect`, disabled in demo mode):
  - `POST /api/shares` → create recipient; returns link + enrolment QR
    **once**
  - `GET /api/shares` → list (never returns secrets/tokens)
  - `PUT /api/shares/{id}` → edit scope/expiry, revoke, re-enrol
    (new secret), regenerate link
  - `GET /api/shares/{id}/access` → access log
- Recipient endpoints (unauthenticated, separate code path):
  - `POST /api/share/verify` `{token, code}` → on success sets share cookie
  - `GET /api/share/report` → read-only JSON for **only** the recipient's
    sections/member scope, built server-side via
    `report.BuildTables(data, member, sections)` — never filtered in the
    frontend
  - `GET /api/share/report/export?format=pdf` → PDF watermarked with the
    recipient's name
- Lockout: 5 wrong codes → locked 15 min (persisted). Generic error
  messages that don't reveal whether a link is valid/revoked/expired.
- Headers on all share routes/pages: `Cache-Control: no-store`,
  `Referrer-Policy: no-referrer` (token is in the URL), `X-Robots-Tag:
  noindex`; tokens redacted from logs.

### 3. Frontend — owner "Sharing" screen (behind normal login)
- Add recipient: name, relation, sections, member scope, optional expiry
- Show enrolment QR + share link/QR once, with a "scan this together now"
  instruction; copy-link button
- List recipients with last access, status; revoke / re-enrol / regenerate
- Access log per recipient
- Mobile-first: entry in the bottom tab bar (not only the sidebar),
  full-width modals on ≤768px

### 4. Frontend — recipient view (`/s/<token>`, phone-first)
- Minimal code-entry page (6-digit numeric input, large tap targets)
- Read-only report: allowed sections only, no nav, no edit controls, no
  other Kosh data; emergency runbooks + doc links once Part 1 lands
- "Download PDF" button

### 5. Recovery
Lost phone / new authenticator → owner re-enrols that recipient (new secret,
old one dead). No self-service recovery, by design.

### 6. Tests
Wrong code, replayed code, clock-drift edges, expired/revoked share,
lockout persistence, share cookie rejected on `/api/*`, out-of-scope
sections never returned, demo mode blocks owner endpoints.

### 7. Docs (same PR as the feature)
`SHARE_ENCRYPTION_KEY` + sharing setup in `docs/SELF_HOSTING.md`; feature
mention in `README.md` and `frontend/src/screens/marketing/Docs.jsx`.

---

## Later
- **Passkeys (WebAuthn)** as an alternative to TOTP — Face ID / fingerprint,
  nothing to type; best UX for older family members
- Optional "notify me when someone opens the share"
- `doc_link` / `emergency_steps` for MF, Stocks, Metals

## Status
- Part 1: `doc_link` shipped for Fixed + Insurance; rest pending.
- Part 2: plan revised to TOTP family access on 2026-11-07. Not yet
  implemented.
