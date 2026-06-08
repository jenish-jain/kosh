# Self-hosting Kosh

Kosh is designed to run entirely on infrastructure you control: your own
Google account, your own spreadsheet, your own server. This guide walks
through every layer — from a local dev instance to a deployed, password-free
"family only" install.

Each section is independent — you can stop after **Local dev mode** if you
just want to try Kosh, or go all the way to **Deploying** if you want a
private install your family can reach from anywhere.

---

## 1. Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 20+
- A Google account (for Sheets, and optionally Sign-In + Drive uploads)

---

## 2. Local dev mode

Dev mode needs nothing beyond the repo itself — it serves
`backend/dev_data.json` (sample data) and disables auth, so you can explore
the full UI immediately.

```bash
make install   # first time only — installs Go + npm dependencies
make dev       # starts the Go API on :8080 and Vite on :5173
```

Open **http://localhost:5173**. Mutations in dev mode are no-ops (logged as
`"sample data — not persisted"`) — nothing you do here touches a real
spreadsheet.

---

## 3. Connecting your own Google Sheet

This is the step that turns Kosh from a demo into your actual ledger — your
data lives in a spreadsheet you own, and Kosh reads/writes it via a Google
Cloud service account.

### 3.1 Create the spreadsheet

Open [Google Sheets](https://sheets.google.com) and create a new spreadsheet.
Add **11 tabs** with these exact names (case-sensitive):

```
Members  MF  Stocks  Metals  Fixed  Insurance  Loans  SIPs  Lumpsums  History  Config
```

Then copy the headers from [Tab schemas](#4-tab-schemas) below into row 1 of
each tab.

> Tip: if you'd rather not do this by hand, run Kosh once against an *empty*
> spreadsheet with a service account that has **Editor** access — it
> auto-creates any missing tabs with the correct headers on startup.

### 3.2 Create a Google Cloud service account

1. Go to [console.cloud.google.com](https://console.cloud.google.com) →
   create or select a project.
2. **APIs & Services → Enable APIs** → search for **Google Sheets API** →
   Enable.
3. **APIs & Services → Credentials → Create Credentials → Service account**.
   - Name it anything (e.g. `kosh-backend`).
   - Role: **Viewer** is enough for read-only; add **Editor** if you want
     write-back (adding/editing rows) from the app.
4. Open the service account → **Keys → Add Key → JSON** → download the file.
5. Rename it to `credentials.json` and place it inside `backend/`:

   ```
   kosh/
   └── backend/
       └── credentials.json   ← here (gitignored — never commit this)
   ```

6. Back in your Google Sheet, click **Share** → paste the service account
   email (looks like `kosh-backend@your-project.iam.gserviceaccount.com`) →
   **Viewer** or **Editor** (matching the role above).

### 3.3 Configure the backend

```bash
cp backend/.env.example backend/.env
```

Open `backend/.env` and set:

```
SPREADSHEET_ID=your_spreadsheet_id_here
```

The spreadsheet ID is the long string in the sheet URL:
`https://docs.google.com/spreadsheets/d/`**`1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms`**`/edit`

### 3.4 Run against the real sheet

```bash
make dev
```

The backend detects `credentials.json` + `SPREADSHEET_ID` and switches out of
dev mode automatically — you'll see no `[dev]` log lines on startup, and a
`✓ Connected to Google Sheets (…)` line instead.

---

## 4. Tab schemas

Every tab has a **header row** (row 1) followed by data rows.
The `id` column must be unique within the tab — use any stable string (e.g.
`mf1`, `st2`). The `member` column references a `Members.id` value.

> ⚠️ **Column order matters.** Kosh reads each row positionally by column
> index, not by header text. If you add a custom column, append it at the
> **end** of the row — never insert it in the middle, or every column after
> it will be misread.

---

#### Members
| A | B | C | D | E | F |
|---|---|---|---|---|---|
| id | name | full_name | relation | slab | color |

| Column | Type | Example |
|--------|------|---------|
| id | string | `you` |
| name | string | `Aarav` — short display name |
| full_name | string | `Aarav (You)` — used in headers |
| relation | string | `Self` / `Mother` / `Father` |
| slab | number | `30` — income tax slab % |
| color | hex string | `#1C4A3A` — avatar/chart color |

---

#### MF (Mutual Funds)
| A | B | C | D | E | F | G | H | I |
|---|---|---|---|---|---|---|---|---|
| id | name | plan | platform | member | invested | current | sip | notes |

| Column | Type | Example |
|--------|------|---------|
| id | string | `mf1` |
| name | string | `Horizon Nifty 50 Index Fund` |
| plan | string | `Direct · Growth` |
| platform | string | `Groww` |
| member | string | `you` — matches Members.id |
| invested | number | `490000` — total amount put in |
| current | number | `512000` — current market value (update monthly) |
| sip | number | `15000` — monthly SIP amount (0 if none) |
| notes | string | `1 failed txn` — optional |

---

#### Stocks
| A | B | C | D | E | F | G |
|---|---|---|---|---|---|---|
| id | name | ticker | qty | avg_price | last_price | member |

| Column | Type | Example |
|--------|------|---------|
| id | string | `st1` |
| name | string | `Reliance Industries` |
| ticker | string | `RELIANCE` |
| qty | number | `40` — number of shares |
| avg_price | number | `2450` — average buy price per share |
| last_price | number | `2620` — current market price (update manually) |
| member | string | `you` |

---

#### Metals
| A | B | C | D | E | F | G | H |
|---|---|---|---|---|---|---|---|
| id | type | date_purchased | grams | buy_rate | today_price | place | member |

| Column | Type | Example |
|--------|------|---------|
| id | string | `me1` |
| type | string | `Gold` / `Silver` |
| date_purchased | date string | `2026-03-29` (YYYY-MM-DD) |
| grams | number | `11` |
| buy_rate | number | `14680` — ₹ per gram at time of purchase |
| today_price | number | `15200` — current ₹ per gram (update manually) |
| place | string | `Tanishq` — where purchased |
| member | string | `you` |

---

#### Fixed
| A | B | C | D | E | F | G | H | I | J | K |
|---|---|---|---|---|---|---|---|---|---|---|
| id | kind | name | member | principal | rate | current_value | opened | matures | monthly | doc_link |

| Column | Type | Example |
|--------|------|---------|
| id | string | `fd1` |
| kind | string | `FD` / `PPF` / `RD` / `Bonds` |
| name | string | `HDFC Bank FD` |
| member | string | `you` |
| principal | number | `500000` — amount deposited |
| rate | number | `7.1` — interest rate % p.a. |
| current_value | number | `535000` — current value with interest |
| opened | date string | `2025-01-12` |
| matures | date string | `2027-01-12` |
| monthly | number | `0` — monthly interest payout (0 for cumulative) |
| doc_link | string | optional — link to the certificate/statement (e.g. a Drive share link). Shows a small link icon next to the entry when present |

---

#### Insurance
| A | B | C | D | E | F | G | H | I | J | K | L |
|---|---|---|---|---|---|---|---|---|---|---|---|
| id | name | type | member | premium | freq | paid | value | cover | maturity | due_date | doc_link |

| Column | Type | Example |
|--------|------|---------|
| id | string | `in1` |
| name | string | `LIC Jeevan Anand` |
| type | string | `Term` / `Endowment` / `ULIP` / `Health` |
| member | string | `you` |
| premium | number | `48000` — premium per frequency cycle |
| freq | string | `annual` / `monthly` / `quarterly` |
| paid | number | `384000` — total premiums paid so far |
| value | number | `412000` — current surrender / fund value (0 for term) |
| cover | number | `1000000` — sum assured |
| maturity | number | `2038` — year of maturity |
| due_date | date string | `2026-07-05` — next premium due date |
| doc_link | string | optional — link to the policy document. Shows a small link icon next to the entry when present |

---

#### Loans
| A | B | C | D | E | F | G | H | I | J | K |
|---|---|---|---|---|---|---|---|---|---|---|
| id | lender | type | member | principal | outstanding | rate | emi | emi_day | started | tenure_months |

| Column | Type | Example |
|--------|------|---------|
| id | string | `ln1` |
| lender | string | `HDFC Home Loan` |
| type | string | `Home` / `Car` / `Personal` / `Education` |
| member | string | `you` |
| principal | number | `1800000` — original loan amount |
| outstanding | number | `1120000` — remaining balance (update periodically) |
| rate | number | `8.5` — interest rate % p.a. |
| emi | number | `18900` — monthly EMI amount |
| emi_day | number | `5` — EMI debit day of month (1–28) |
| started | date string | `2022-04-01` |
| tenure_months | number | `180` — total loan tenure in months |

Outstanding balances reduce net worth (assets − debt), and EMIs appear
alongside SIPs in "Expected outflows".

---

#### SIPs
| A | B | C | D | E | F | G | H | I |
|---|---|---|---|---|---|---|---|---|
| id | fund | member | amount | day | status | start_date | platform | kind |

| Column | Type | Example |
|--------|------|---------|
| id | string | `sip1` |
| fund | string | `Horizon Nifty 50 Index` — short display name |
| member | string | `you` |
| amount | number | `15000` — monthly debit amount |
| day | number | `20` — debit day of month (1–28) |
| status | string | `active` / `paused` |
| start_date | date string | `2023-09-18` |
| platform | string | `Groww` |
| kind | string | `MF` / `Stock` — what the SIP buys |

---

#### Lumpsums
| A | B | C | D | E |
|---|---|---|---|---|
| id | fund | member | amount | date |

| Column | Type | Example |
|--------|------|---------|
| id | string | `lp1` |
| fund | string | `Bluepeak Small Cap Fund` |
| member | string | `you` |
| amount | number | `49998` |
| date | date string | `2024-12-18` |

---

#### History
| A | B |
|---|---|
| month | value |

| Column | Type | Example |
|--------|------|---------|
| month | string | `Jun 26` — 3-letter month + 2-digit year |
| value | number | `2390000` — total household net worth that month |

Kosh appends a row here automatically once per month when you load the
dashboard (see `maybeSnapshotHistory` in `backend/handlers/data.go`) — no
manual upkeep needed, though you can edit past entries by hand if you ever
need to correct one.

---

#### Config
| A | B |
|---|---|
| key | value |

This tab is a **key–value list**, not a row-per-record table. Add one row per
config key.

| Key | Value | Notes |
|-----|-------|-------|
| gross_income | `6200000` | Annual gross income (salary + other) |
| regime | `New` | `New` or `Old` — for display only |
| capital_gains_this_year | `320000` | Realised cap gains this FY |
| shifted_to_parents | `480000` | Amount gifted / loaned to parents this FY |
| surcharge_from | `5000000` | First surcharge threshold — always 5000000 |
| next_surcharge_at | `10000000` | Second surcharge threshold — always 10000000 |
| parents_capacity_mom | `700000` | Estimated annual income capacity for Mom |
| parents_capacity_dad | `750000` | Estimated annual income capacity for Dad |
| deduction_80c | `150000` | Total 80C deductions claimed |
| deduction_80d | `25000` | Total 80D deductions claimed |
| other_deductions | `50000` | NPS / LTA / other deductions |
| saved_by_filing | `64350` | Tax saved vs. filing everything under self |

**Monthly upkeep** — once a month, update:
1. `current` in MF, `last_price` in Stocks, `today_price` in Metals, and
   `outstanding` in Loans with latest figures.
2. `Config.capital_gains_this_year` if you've realised gains.

(Net worth history is captured automatically — see the History section above.)

---

## 5. Adding Google Sign-In (optional)

Without sign-in, anyone who can reach your server sees and can edit your
data. For a private deploy reachable from the internet, restrict access to
specific Google accounts (e.g. you and your spouse):

1. In [Google Cloud Console](https://console.cloud.google.com), open
   **APIs & Services → Credentials → Create Credentials → OAuth client ID**.
   - Application type: **Web application**.
   - **Authorized JavaScript origins**: add `http://localhost:8080` (for
     local testing) and your production URL (e.g. `https://kosh.example.com`).
2. Copy the generated **Client ID** into `backend/.env`:
   ```
   GOOGLE_CLIENT_ID=xxxxxxxxxx.apps.googleusercontent.com
   SESSION_SECRET=        # generate with: openssl rand -base64 32
   ALLOWED_EMAILS=you@gmail.com,partner@gmail.com
   ```
3. Restart the backend. You'll now see a **Sign in with Google** screen, and
   only the listed emails can get past it.

Auth is entirely optional — leave `GOOGLE_CLIENT_ID` / `SESSION_SECRET` unset
to run open (e.g. for purely local use on a trusted machine).

### Try-the-demo mode

Whenever sign-in is enabled, the login screen also offers a **"Try the
demo"** button — it starts a read-only session backed by the sample data in
`backend/dev_data.json`, with no Google account needed. This is what powers
Kosh's public showcase, and it's on by default for any deployment with auth
configured; there's nothing extra to set up.

---

## 6. Document uploads & AI parsing (optional)

Kosh can extract structured fields from FD certificates, insurance policies,
and metal purchase receipts (PDF/image), and store the original file in
*your own* Google Drive (never the author's).

This needs:
- The same `GOOGLE_CLIENT_ID` as above, with the **Google Drive API** enabled
  in your Cloud project (uploads use the narrow `drive.file` scope — Kosh can
  only see files it creates, nothing else in your Drive).
- An [Anthropic API key](https://console.anthropic.com) for parsing:
  ```
  ANTHROPIC_API_KEY=sk-ant-...
  ```

Without `ANTHROPIC_API_KEY`, the upload feature is simply unavailable — the
rest of the app works normally.

---

## 7. Deploying

Kosh ships as a single Docker image — a static frontend served by the same Go
binary that serves the API, so there's exactly one process to run.

### Build & run with Docker

```bash
docker build -t kosh .
docker run -p 8080:8080 \
  -e SPREADSHEET_ID=your_spreadsheet_id \
  -e GOOGLE_CLIENT_ID=xxxxxxxxxx.apps.googleusercontent.com \
  -e SESSION_SECRET=$(openssl rand -base64 32) \
  -e ALLOWED_EMAILS=you@gmail.com \
  -e GOOGLE_CREDENTIALS_B64=$(base64 -i backend/credentials.json) \
  kosh
```

`GOOGLE_CREDENTIALS_B64` exists because most container platforms don't offer
a clean way to mount a secret *file* — the entrypoint decodes it to
`credentials.json` on disk before starting the server (see the `CMD` line in
`Dockerfile`). If your platform supports mounting files directly, you can
instead mount `credentials.json` and skip this variable.

### Deploying to Railway

This repo includes a `railway.json` pre-configured to build from the
`Dockerfile`. To deploy:

1. Create a new Railway project from this GitHub repo.
2. Set the environment variables listed in [§8](#8-environment-variable-reference)
   under **Settings → Variables** (at minimum `SPREADSHEET_ID` and
   `GOOGLE_CREDENTIALS_B64`; add the auth variables if you want sign-in).
3. Railway builds the Dockerfile and exposes the app on a generated domain —
   add that domain to your OAuth client's **Authorized JavaScript origins**
   (see §5) if you've enabled sign-in.

Any other platform that runs a Dockerfile (Fly.io, Render, a VPS with
`docker run`, …) works the same way — just set the same environment
variables.

---

## 8. Environment variable reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SPREADSHEET_ID` | For live data | — | Your Google Sheet ID. Omit to run in dev mode (sample data, no persistence) |
| `CREDENTIALS_PATH` | For live data | `credentials.json` | Path to the service-account JSON key |
| `PORT` | No | `8080` | Port for the Go HTTP server |
| `FRONTEND_DIST` | No | `../frontend/dist` | Path to the built frontend assets |
| `GOOGLE_CLIENT_ID` | For sign-in & uploads | — | OAuth 2.0 Web client ID from Google Cloud Console |
| `SESSION_SECRET` | For sign-in | — | Random string used to HMAC-sign session cookies (`openssl rand -base64 32`) |
| `ALLOWED_EMAILS` | For sign-in | — | Comma-separated list of Google account emails allowed to sign in |
| `COOKIE_SECURE` | No | `true` | Set to `false` to allow cookies over plain HTTP (local testing only) |
| `ANTHROPIC_API_KEY` | For uploads | — | Enables AI-assisted PDF/image parsing for FD, insurance, and metals |
| `PROMPTS_DIR` | No | `prompts` | Directory of extraction prompt templates (`fd.md`, `insurance.md`, `metals.md`) |
| `GOOGLE_CREDENTIALS_B64` | Docker only | — | Base64-encoded service-account JSON — decoded to `CREDENTIALS_PATH` at container startup, for platforms with no secret-file mounting |

`GOOGLE_APPLICATION_CREDENTIALS` is **not** needed — the backend reads
`credentials.json` from its working directory directly.

---

## Project structure

```
kosh/
├── Makefile
├── Dockerfile
├── backend/
│   ├── main.go              # HTTP server, routing, env config
│   ├── handlers/
│   │   ├── auth.go          # Google Sign-In, sessions, demo mode
│   │   ├── data.go          # GET /api/data — reads all tabs
│   │   ├── mutations.go     # POST/PUT/DELETE per sheet
│   │   └── upload.go        # Document upload + AI parsing
│   ├── sheets/
│   │   └── client.go        # Google Sheets API wrapper
│   ├── drive/               # Google Drive upload (user-owned storage)
│   ├── dev_data.json        # Sample data for dev mode & demo sessions
│   ├── credentials.json     # ← add this (gitignored)
│   └── .env                 # ← add this (gitignored)
└── frontend/
    ├── src/
    │   ├── screens/         # Dashboard, Investments, SIPs, Expenses, Family, Tax
    │   ├── components/      # Icons, Primitives (Avatar, Modal, SaveBar…)
    │   └── data/            # api.js, helpers.js, context.jsx, driveAuth.js
    └── dist/                # built by `make build`
```
