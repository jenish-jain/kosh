# Kosh — Family Wealth Tracker

A calm, editorial-design wealth dashboard backed by Google Sheets.

---

## Quick start (dev mode)

Dev mode uses `backend/dev_data.json` — no Google credentials needed.

```
make install   # first time only
make dev       # starts Go on :8080 + Vite on :5173
```

Open http://localhost:5173

---

## Connect to Google Sheets

### 1. Create the spreadsheet

Open [Google Sheets](https://sheets.google.com) and create a new spreadsheet.
Add **10 tabs** with these exact names (case-sensitive):

```
Members  MF  Stocks  Metals  Fixed  Insurance  SIPs  Lumpsums  History  Config
```

Then copy the headers below into row 1 of each tab.

---

### 2. Tab schemas

Every tab has a **header row** (row 1) followed by data rows.  
The `id` column must be unique within the tab — use any stable string (e.g. `mf1`, `st2`).  
The `member` column references a `Members.id` value.

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
| name | string | `HDFC NIFTY 50 Index Fund` |
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
| place | string | `Kalamandir` — where purchased |
| member | string | `you` |

---

#### Fixed
| A | B | C | D | E | F | G | H | I | J |
|---|---|---|---|---|---|---|---|---|---|
| id | kind | name | member | principal | rate | current_value | opened | matures | monthly |

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

---

#### Insurance
| A | B | C | D | E | F | G | H | I | J |
|---|---|---|---|---|---|---|---|---|---|
| id | name | type | member | premium | freq | paid | value | cover | maturity |

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

---

#### SIPs
| A | B | C | D | E | F | G | H |
|---|---|---|---|---|---|---|---|
| id | fund | member | amount | day | status | start_date | platform |

| Column | Type | Example |
|--------|------|---------|
| id | string | `sip1` |
| fund | string | `HDFC NIFTY 50 Index` — short display name |
| member | string | `you` |
| amount | number | `15000` — monthly debit amount |
| day | number | `20` — debit day of month (1–28) |
| status | string | `active` / `paused` |
| start_date | date string | `2023-09-18` |
| platform | string | `Groww` |

---

#### Lumpsums
| A | B | C | D | E |
|---|---|---|---|---|
| id | fund | member | amount | date |

| Column | Type | Example |
|--------|------|---------|
| id | string | `lp1` |
| fund | string | `Quant Small Cap Fund` |
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

**Monthly workflow** — on the last day of each month:
1. Open the History tab.
2. Add a new row: month label (`Jul 26`) and the total net worth figure from the Dashboard.
3. Also update `current` in the MF tab, `last_price` in Stocks, and `today_price` in Metals with latest prices.
4. Update `Config.capital_gains_this_year` if you have realised gains.

---

#### Config
| A | B |
|---|---|
| key | value |

This tab is a **key–value list**, not a row-per-record table. Add one row per config key.

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

---

### 3. Create a Google Cloud service account

1. Go to [console.cloud.google.com](https://console.cloud.google.com) → create or select a project.
2. **APIs & Services → Enable APIs** → search for **Google Sheets API** → Enable.
3. **APIs & Services → Credentials → Create Credentials → Service account**.
   - Name it anything (e.g. `kosh-backend`).
   - Role: **Viewer** is enough for read-only; add **Editor** if you want write-back from the app.
4. Open the service account → **Keys → Add Key → JSON** → download the file.
5. Rename it to `credentials.json` and place it inside `backend/`.

   ```
   kosh/
   └── backend/
       └── credentials.json   ← here
   ```

6. Back in your Google Sheet, click **Share** → paste the service account email  
   (looks like `kosh-backend@your-project.iam.gserviceaccount.com`) → **Viewer** or **Editor**.

---

### 4. Create `backend/.env`

```bash
cp backend/.env.example backend/.env
```

Open `backend/.env` and fill in:

```
SPREADSHEET_ID=your_spreadsheet_id_here
```

The spreadsheet ID is the long string in the sheet URL:  
`https://docs.google.com/spreadsheets/d/`**`1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms`**`/edit`

---

### 5. Run against the real sheet

```
make dev
```

The backend will detect `credentials.json` and `SPREADSHEET_ID` and switch out of dev mode automatically. You'll see no `[dev]` log lines on startup.

---

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SPREADSHEET_ID` | Yes (prod) | — | Google Sheet ID |
| `PORT` | No | `8080` | Port for the Go server |

`GOOGLE_APPLICATION_CREDENTIALS` is **not** needed — the backend reads `credentials.json` from its working directory directly.

---

## Project structure

```
kosh/
├── Makefile
├── backend/
│   ├── main.go              # HTTP server, routing
│   ├── handlers/
│   │   ├── data.go          # GET /api/data — reads all tabs
│   │   └── mutations.go     # POST/PUT/DELETE per sheet
│   ├── sheets/
│   │   └── client.go        # Google Sheets API wrapper
│   ├── dev_data.json        # Seed data for dev mode
│   ├── credentials.json     # ← add this (gitignored)
│   └── .env                 # ← add this (gitignored)
└── frontend/
    ├── src/
    │   ├── screens/         # Dashboard, Investments, SIPs, Family, Tax
    │   ├── components/      # Icons, Primitives (Avatar, Modal, SaveBar…)
    │   └── data/            # api.js, helpers.js, context.jsx
    └── dist/                # built by `make build`
```
