# Kosh — Implementation Plan

**Design**: Editorial (serif numerals, hairline rules, ivory paper)  
**Database**: Google Sheets (data lives in your sheet; app reads/writes via Google Apps Script)  
**Stack**: Vanilla HTML + React (no build tools — open `index.html` in a browser or serve statically)

---

## Architecture

```
Google Sheets (source of truth)
    ↕ read/write
Google Apps Script Web App  ← deployed URL stored in config.js
    ↕ JSON fetch
index.html (React app, editorial design)
    ↑ served from: GitHub Pages / Netlify / local file / GAS HTML Service
```

**Data flows:**
- App loads → fetch all data from Apps Script → render
- User edits a cell inline → PATCH to Apps Script → updates Sheet row
- User adds a holding/SIP → POST to Apps Script → appends Sheet row
- Sheet edits (direct in Sheets) reflect on next app load/refresh

---

## Google Sheets Schema (one tab per entity)

| Tab | Columns |
|-----|---------|
| `Members` | id, name, full_name, relation, slab, color |
| `MF` | id, name, plan, platform, member, invested, current, sip, notes |
| `Stocks` | id, name, ticker, qty, avg_price, last_price, member |
| `Metals` | id, type, date_purchased, grams, buy_rate, today_price, place, member |
| `Fixed` | id, kind, name, member, principal, rate, current_value, opened, matures, monthly |
| `Insurance` | id, name, type, member, premium, freq, paid, value, cover, maturity |
| `SIPs` | id, fund, member, amount, day, status, start_date, platform |
| `Lumpsums` | id, fund, member, amount, date |
| `Config` | key, value (stores tax data: gross_income, regime, capital_gains, shifted_to_parents, etc.) |

---

## Stage 1 — Project Structure

Files to create in the project root:

```
kosh/
├── index.html          ← entry point (loads React, styles, scripts)
├── styles.css          ← editorial design system (copied + extended from prototype)
├── config.js           ← Apps Script URL + app config (not committed with secrets)
├── data.js             ← data fetcher: calls Apps Script, exposes window.Kosh
├── components.jsx      ← shared UI primitives (Icon, Avatar, Chart, Modal, etc.)
├── dashboard.jsx       ← editorial dashboard screen
├── investments.jsx     ← MF / Stocks / Metals / Insurance editable tables
├── sips.jsx            ← SIP scheduler + calendar + lumpsum modal
├── members.jsx         ← family overview + member drill-down
├── tax.jsx             ← tax slab + surcharge runway + income-split planner
├── app.jsx             ← app shell, sidebar, member switcher, routing
├── apps-script/
│   └── Code.gs         ← Google Apps Script (deploy as Web App)
└── plan.md             ← this file
```

**Deliverable**: Empty scaffolding compiles and shows a loading screen.

---

## Stage 2 — Google Apps Script Backend (`Code.gs`)

A single `doGet` / `doPost` handler deployed as a Web App (Anyone can access).

```
GET  /exec              → returns { members, mf, stocks, metals, fixed, insurance, sips, lumpsums, config }
POST /exec  action=add  → appends a row to specified sheet, returns new row id
POST /exec  action=edit → updates a specific row by id
POST /exec  action=del  → deletes a row by id
```

The script reads data from the spreadsheet and returns JSON.  
CORS headers included so the browser can fetch it directly.

**Deliverable**: Deployed Web App URL that returns your data as JSON.

---

## Stage 3 — Data Layer (`data.js`)

Replace hardcoded prototype data with live fetch from Apps Script.

- On load: `fetch(APPS_SCRIPT_URL)` → parse → store in `window.Kosh`
- Formatters (`fmtINR`, `fmtCompact`, `fmtPct`) stay the same
- Aggregation helpers (`classTotals`, `memberTotal`, `holdingsFor`) stay the same
- Add `Kosh.editRow(sheet, id, patch)` → calls Apps Script POST
- Add `Kosh.addRow(sheet, data)` → calls Apps Script POST
- Loading/error state exposed for the UI

**Deliverable**: App loads live data from your Google Sheet.

---

## Stage 4 — Editorial Design System (`styles.css` + `index.html`)

Port the editorial theme from the prototype verbatim:

- CSS variables: `--paper:#FBF8F1`, `--ink:#221F1A`, `--ink2:#6B665C`, `--line:#DAD3C5`, `--accent:#1C4A3A`
- Typography: `Hanken Grotesk` (UI) + `Newsreader` (numerals/display)
- App shell: sidebar + topbar + content area
- `theme-editorial` class applied to `<html>` at startup (no toggle needed — editorial only)

**Deliverable**: App shell renders with correct editorial styling.

---

## Stage 5A — Dashboard Screen (`dashboard.jsx`)

Editorial dashboard layout:
- Statement header: "Statement of net worth · As on [date] · [scope]"
- Thick hairline rule
- Hero: large serif net worth + invested + gain
- Slim area sparkline (12-month history from `Config` sheet or hardcoded)
- Allocation bar (5 asset classes) + per-class figures
- Thin hairline rule
- Two-column: family member list (left) | monthly commitment + protection (right)
- Italic footnote

**Deliverable**: Dashboard renders with live data, member switcher works.

---

## Stage 5B — Investments Screen (`investments.jsx`)

- Statement header: "Schedule of holdings · [scope] · As on [date]"
- Four summary tiles (MF / Stocks / Gold & Silver / Insurance) — click to jump to tab
- Tab bar: Mutual Funds | Stocks | Gold & Silver | Insurance & Plans
- **Mutual Funds table**: Fund name + plan + platform | Owner (if whole-family) | Monthly SIP (editable) | Invested (editable) | Current value (editable)
- **Stocks table**: Name + ticker | Owner | Qty (editable) | Avg price (editable) | Last price (editable) | Invested | Current value
- **Metals table**: Type + date + place | Owner | Grams (editable) | Buy rate/g (editable) | Today/g (editable) | Invested | Current value
- **Insurance table**: Plan name | Owner | Type pill | Premium (editable) + freq | Paid so far (editable) | Cover | Value (editable) | Maturity year
- Totals bar at bottom of each table
- "Add" modal for each asset class
- "Import CSV" modal (UI only — column mapping stub)
- Inline edits → auto-save to Google Sheet via Apps Script

**Deliverable**: All four tables render and inline edits persist to Sheets.

---

## Stage 5C — SIPs & Schedule Screen (`sips.jsx`)

- Statement header: "SIP schedule & commitments · [scope]"
- 4 headline figures: Monthly outflow | Active SIPs | Annual commitment | Lumpsums added
- "Lumpsum top-up" button (primary)
- Two-column: Manage SIPs table (left) | June calendar + recent lumpsums (right)
- **SIP table**: Fund | Owner | Amount (edit) | Day (edit) | Status pill | Edit button
- Edit modal: change amount, debit day, pause/resume
- Calendar: 30-day grid, SIP days highlighted with amount
- Recent lumpsums list
- Lumpsum modal: pick fund → enter amount → writes to Sheets

**Deliverable**: SIP screen works, edits persist.

---

## Stage 5D — Family Screen (`members.jsx`)

**Overview view** (whole-family selected):
- Statement header: "Family portfolios · 3 members"
- Hero net worth + member share bar
- Per-member allocation bar
- Member statement rows (click → drill into member)

**Member profile view** (one member selected):
- Statement header with member name + slab
- Hero net worth (serif, large)
- Allocation bar (5 classes)
- Two-column: Active SIPs list (left) | Top MF holdings (right)

**Deliverable**: Family overview and member drill-down both render.

---

## Stage 5E — Tax Screen (`tax.jsx`)

- Statement header: "Tax position & planning · Aarav · FY 2025-26"
- 4 headline figures: Gross income | Slab (30% + surcharge) | Capital gains | Saved by splitting
- Surcharge runway bar (₹50L → ₹1Cr, your position shown)
- Income-split planner (slider → routes investment income to parents → shows tax saved)
- Indicative disclaimer

Tax figures come from `Config` sheet (editable in Sheets directly).

**Deliverable**: Tax screen renders with live config values.

---

## Stage 6 — Persistence & Real-time Sync

- Inline cell edits → debounced POST to Apps Script (300ms after blur)
- Optimistic updates: UI updates immediately, rolls back on error
- Add/edit/delete modals → POST → refresh local cache
- Toast notification on save success/error
- "Last synced" indicator in sidebar footer

**Deliverable**: All changes persist to Google Sheet.

---

## Stage 7 — Deployment & Polish

- **Hosting options** (pick one):
  - **GitHub Pages**: push repo, enable Pages (free, public URL)
  - **Netlify**: drag-and-drop the folder (free, custom domain)
  - **Local**: just open `index.html` in Chrome (no server needed if Apps Script CORS is set)
  - **Apps Script HTML Service**: serve the whole app from the sheet itself (zero external hosting)
- Loading skeleton while fetching data
- Error boundary if Apps Script is unreachable
- `config.js` instructions for first-time setup (paste your Apps Script URL)
- Mobile-responsive sidebar (collapses to bottom tab bar on narrow screens)

---

## Questions Before Starting

Before implementation begins, I need answers to these:

1. **Names**: The design uses "Aarav (You)", "Meena" (Mom), "Ramesh" (Dad). Want to keep these or use real names?
ans: we will use real names can hardcode if needed for now but ideally should come from a members form while initial onboarding (or just added to the sheet directly)

2. **Hosting**: Where should the app live?
   - `A` — Local only (open `index.html` in browser, no deployment needed)
   - `B` — GitHub Pages (push to GitHub, free public URL)
   - `C` — Apps Script HTML Service (app served from inside your Google Sheet — zero external hosting, most self-contained)

ans : option c — serve from Apps Script HTML Service. This way, the entire app (frontend + backend) lives inside the Google Sheet, making it super easy to manage and update without needing an external hosting solution.

3. **Write-back**: When you edit a cell inline (e.g. update "Current value" of a fund), should it:
   - `A` — Immediately save to Google Sheet
   - `B` — Save only when you click a "Save changes" button
   - `C` — In-memory only (manual edits to Sheet stay as source of truth; app is read-only)
ans: option B

4. **Auth**: Should the app require any login/password, or is it open to anyone with the URL?
ans: open to anyone with the URL (since it's protected by the obscurity of the URL and the Google Sheet's own sharing settings) will work on auth later

5. **Current market prices**: For stocks and metals, do you want a "Today's price/g" or "Last price" column in the sheet that you update manually? (No live API — you'd paste the price from your broker.)
ans : no live api for now we will enhance that later if needed

6. **History chart**: The 12-month net-worth sparkline — should this come from a `History` tab in the sheet (you add a row each month), or is a placeholder chart fine for now?
ans: we will use google sheet for this to create agreegated reports which can be used to create the sparkline in the dashboard. add readme steps for this on sheets will assist for steps to create the history tab and how to add data to it on a monthly basis.