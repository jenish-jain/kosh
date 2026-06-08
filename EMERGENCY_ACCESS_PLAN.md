# Plan: Emergency encashment info + nominee access

Goal: attach PDF links and step-by-step liquidation/encashment instructions to
investments (MF, Stocks, FDs, Insurance, Metals) so that in an emergency,
declared nominees (parents/spouse) can access them in one click — without
needing to learn the Kosh app or have a Google account.

## Part 1 — Data model: what to capture, and where

Add two new fields to each liquidatable asset type:

- **`doc_link`** — Drive URL to the policy/statement/certificate PDF. Kosh
  already has Drive upload plumbing (`backend/handlers/upload.go`,
  `frontend/src/components/UploadZone.jsx`, returns `DriveURL`/`drive_url`).
  Currently the URL gets appended into the `notes` column as a workaround
  (`Investments.jsx` ~line 560) — instead, route it into a dedicated column.
- **`emergency_steps`** — free-text liquidation runbook: agent/RM name & phone,
  folio/policy/account number, branch/helpline, claim form name, documents
  required (death certificate, KYC, nominee proof), expected turnaround.

Backend changes:
- Extend `koshTabs` column defs in `backend/main.go` (~line 31-43)
- Extend `sheetColumns` map in `backend/handlers/mutations.go` (~line 11-22)
- Extend the relevant structs + `ReadSheet` parsing in `backend/handlers/data.go`
  (~lines 40-98, 276-339)

Frontend changes:
- Add an expandable "Emergency info" section / modal per row in
  `InsuranceTable`/`FixedTable`/etc. in `frontend/src/screens/Investments.jsx`
  with a small form for `doc_link` (wired to the existing `UploadZone`) and
  `emergency_steps`.

Current sheetColumns reference (as of writing):
```
MF:        ["id", "name", "plan", "platform", "member", "invested", "current", "sip", "notes"]
Stocks:    ["id", "name", "ticker", "qty", "avg_price", "last_price", "member"]
Metals:    ["id", "type", "date_purchased", "grams", "buy_rate", "today_price", "place", "member"]
Fixed:     ["id", "kind", "name", "member", "principal", "rate", "current_value", "opened", "matures", "monthly"]
Insurance: ["id", "name", "type", "member", "premium", "freq", "paid", "value", "cover", "maturity", "due_date"]
```

## Part 2 — Nominee access: secret link + passphrase gate

Decision (confirmed with user): use a **secret read-only link gated by a
passphrase** (not OTP — OTP would require adding a new email/SMS-sending
dependency the project doesn't have; a passphrase needs nothing new).

### 1. New "Emergency" tab/config (Sheets-backed)
- A long random **access token** (e.g. 32-byte hex) forming the secret URL:
  `https://kosh.app/emergency/<token>`
- A **passphrase**, stored hashed (bcrypt) — never in plaintext
- Optional list of nominee names/relations, just for display
  ("Prepared for: Mom, Dad, Spouse")

### 2. Backend — unauthenticated read-only route `/emergency/<token>`
- Bypasses normal Google-login auth entirely (separate code path from `auth.go`)
- First visit: shows a minimal passphrase-entry page
- On correct passphrase: issues a short-lived signed cookie scoped only to this
  view, then renders the emergency content
- **Rate-limiting / lockout** after N wrong attempts (in-memory counter keyed
  by IP+token) — brute-force protection even if the URL leaks
- Defense in depth: unguessable token is the first line, passphrase the second

### 3. Emergency content shown (read-only, stripped down)
- Grouped by asset class (FDs, insurance, MF, stocks): name, account/folio/
  policy number, current value, the `emergency_steps` runbook text, and a
  clickable `doc_link` to the supporting PDF in Drive
- No edit controls, no nav, no other Kosh data — just what's needed to act

### 4. New "Emergency" settings screen (in Kosh, behind normal login)
- Button to **generate/regenerate** the link + set the passphrase
- Shows the current secret URL (copy button) to store in a password manager /
  share with nominees
- **Last-accessed log** (timestamp + rough location/IP) — peace of mind and a
  tamper signal
- One-click **revoke & regenerate** if the link is ever suspected to have leaked

## Status
Plan approved by user on 2026-06-08. Not yet implemented — revisit when ready
to build.
