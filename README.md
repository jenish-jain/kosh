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
Railway) so it's reachable from anywhere.

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

See [docs/SELF_HOSTING.md § Project structure](docs/SELF_HOSTING.md#project-structure)
for a tour of the codebase.

---

## A note on privacy

If you self-host Kosh, your financial data stays entirely within your own
Google account — nothing passes through the project author's servers.
The only third parties involved are Google itself (Sheets/Drive, and
Sign-In if you enable it) and, optionally, Anthropic (for AI-assisted
document parsing, if you turn that on).

---

## Tests & Coverage

### Run tests

```bash
# Backend (Go)
cd backend && go test ./...

# Backend with coverage report
cd backend && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Frontend (Node)
cd frontend && npm test

# Frontend with coverage report
cd frontend && npm run test:coverage
```

### Current coverage

**Backend**

| Package | Coverage |
|---|---|
| `kosh/handlers` | 23.4% |
| `kosh/sheets` | 11.8% |
| `kosh/drive` | 0% |

Low coverage in `handlers` and `sheets` reflects that the Google Sheets and Drive API clients require live credentials and are not yet wrapped in interfaces for injection. See [`test-generation-report.md`](test-generation-report.md) for the refactoring plan.

**Frontend**

| File | Stmts | Branch | Funcs |
|---|---|---|---|
| `data/api.js` | 100% | 100% | 100% |
| `data/context.jsx` | 100% | 100% | 100% |
| `data/helpers.js` | 100% | 72% | 100% |
| `components/Icons.jsx` | 100% | 100% | 100% |
| `components/Primitives.jsx` | 47% | 95% | 62% |
| `data/driveAuth.js` | 0% | 0% | 0% |
| `components/UploadZone.jsx` | 0% | 0% | 0% |
| `screens/*` | 0% | 0% | 0% |
| `App.jsx` | 0% | 0% | 0% |

Zero coverage in screens and `App.jsx` is expected — they depend on the full context + Google Sign-In integration. The data and pure-logic layer is fully covered. See [`test-generation-report.md`](test-generation-report.md) for the screen-testing plan.

---

## License

[MIT](LICENSE) — use it, fork it, self-host it, make it yours.
