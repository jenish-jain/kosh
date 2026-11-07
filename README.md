# Kosh (कोश) — Family Wealth Tracker

A calm, editorial-design wealth dashboard backed by Google Sheets — built for
tracking a household's mutual funds, stocks, FDs, insurance, loans, gold, and
more in one place, with a tax-aware view across family members.

There's no database to manage and no third-party service holding your
financial data — Kosh reads and writes a Google Sheet you own, and that's it.

---

## Try it without an account

Open the app and click **"Try the demo"** on the sign-in screen — it drops
you straight into a fully working dashboard backed by sample data
(`backend/dev_data.json`), read-only, no Google sign-in required. It's the
fastest way to see whether Kosh's approach to tracking wealth fits how you
think about money.

---

## Two ways to use it

- **Hosted Kosh**: sign up with Google and you're in, with a 14-day free
  trial and then ₹149/month or ₹1,499/year for the whole household. Your
  data still lives in a Google Sheet in *your* Drive. See
  [pricing](https://kosh.jenishjain.in/pricing).
- **Self-hosted**: free and open source (AGPL-3.0). Run it on your own
  Google Cloud project; everything below is about this route.

---

## Quick start (run it locally)

Dev mode uses `backend/dev_data.json` — no Google credentials needed, and
nothing you do touches a real spreadsheet.

```bash
make install   # first time only
make dev       # starts Go on :8080 + Vite on :5173
```

Open http://localhost:5173

---

## Make it yours

To track your *actual* finances, point Kosh at a Google Sheet you own —
no separate database, no vendor lock-in, your data stays in your account.

**→ See [docs/SELF_HOSTING.md](docs/SELF_HOSTING.md)** for the full guide:
connecting your spreadsheet, adding Google Sign-In so only your family can
access it, enabling AI-assisted document uploads, and deploying it (Docker /
Cloud Run) so it's reachable from anywhere.

---

## How it's built

- **Backend**: Go, talking directly to the Google Sheets API (no ORM, no
  intermediate database)
- **Frontend**: React + Vite, hand-rolled editorial design (serif numerals,
  hairline rules, ivory paper — no component library)
- **Storage**: a Google Sheet you own — viewable, editable, and exportable
  with tools you already trust
- **Auth**: optional Google Sign-In restricted to an email allow-list — or
  run it open on a trusted machine
- **Family sharing**: optional read-only report links for family members,
  unlocked with an authenticator-app code (no passwords, no SMS) — see
  [docs/SELF_HOSTING.md](docs/SELF_HOSTING.md#family-sharing-optional)

See [docs/SELF_HOSTING.md § Project structure](docs/SELF_HOSTING.md#project-structure)
for a tour of the codebase.

---

## A note on privacy

If you self-host Kosh, your financial data stays entirely within your own
Google account — nothing passes through the project author's servers.
The only third parties involved are Google itself (Sheets/Drive, and
Sign-In if you enable it) and, optionally, Anthropic (for AI-assisted
document parsing, if you turn that on).

On hosted Kosh the financial data still lives in a Sheet in the customer's
own Drive; the service keeps only account and billing records (see
[docs/HOSTED.md](docs/HOSTED.md)).

---

## Tests

```bash
# Backend (Go)
cd backend && go test ./...

# Frontend (Node)
cd frontend && npm test
```

---

## License

Copyright © 2026 Jenish Jain. Licensed under the
[GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0-only).

Use it, fork it, self-host it for your family — all free. The one condition:
if you run a **modified** Kosh as a service for other people, you must offer
them your modified source under the same license.

Releases before November 2026 were published under the MIT License, and
copies obtained under those terms keep them.
