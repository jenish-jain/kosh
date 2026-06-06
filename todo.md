# Kosh — Executable Todo

**Stack**: Go backend (Google Sheets API) + React/Vite frontend  
**Structure**: `backend/` and `frontend/` as sibling folders  
**Auth**: Service account credentials (JSON file, gitignored)  
**Save**: Explicit "Save changes" button per screen section  

---

## Stage 1 — Project Scaffold
- [x] 1.1 Create `backend/` and `frontend/` directory structure
- [x] 1.2 Initialize Go module (`backend/go.mod`, `backend/main.go`)
- [x] 1.3 Initialize React + Vite app (`frontend/`)
- [x] 1.4 Add `.gitignore` (exclude credentials, node_modules, dist, .env)
- [x] 1.5 Verify Go server starts (`go run ./backend`)
- [x] 1.6 Verify Vite dev server starts (`npm run dev` in `frontend/`)

## Stage 2 — Google Sheets Setup *(user action required before Stage 3 works live)*
- [ ] 2.1 Create a new Google Sheet named "Kosh"
- [ ] 2.2 Add tabs: Members, MF, Stocks, Metals, Fixed, Insurance, SIPs, Lumpsums, History, Config
- [ ] 2.3 Add headers to each tab (see README for exact column names)
- [ ] 2.4 Populate seed data (your real SIP + metal data from the design file)
- [ ] 2.5 Create a Google Cloud project → enable Sheets API
- [ ] 2.6 Create a service account → download `credentials.json` → place in `backend/`
- [ ] 2.7 Share the Google Sheet with the service account email (Editor role)
- [ ] 2.8 Copy `.env.example` to `.env` and fill in `SPREADSHEET_ID`

## Stage 3 — Go Backend
- [x] 3.1 Add Google Sheets API dependency (`google.golang.org/api/sheets/v4`)
- [x] 3.2 Google Sheets client (`backend/sheets/client.go`) — reads all tabs into structs
- [x] 3.3 `GET /api/data` — returns full JSON payload (members, mf, stocks, metals, fixed, insurance, sips, lumpsums, history, config)
- [x] 3.4 `POST /api/{sheet}` — append a new row, returns row with generated id
- [x] 3.5 `PUT /api/{sheet}/{id}` — update a specific row by id column
- [x] 3.6 `DELETE /api/{sheet}/{id}` — delete a row by id
- [x] 3.7 CORS middleware (allow frontend dev server origin)
- [x] 3.8 Static file handler — serve `../frontend/dist` at `/`
- [x] 3.9 Dev mode fallback — if `credentials.json` missing, load `backend/dev_data.json` (seed data from prototype)

## Stage 4 — Frontend: Design System
- [x] 4.1 Port `styles.css` from prototype (editorial theme: paper background, hairlines, serif numerals)
- [x] 4.2 Import Google Fonts (Hanken Grotesk + Newsreader) in `index.html`
- [x] 4.3 App shell layout component (sidebar + topbar + `<main>`)
- [x] 4.4 Sidebar: brand mark (क), nav links, household net worth footer
- [x] 4.5 Topbar: member switcher chips (Whole family / You / Mom / Dad)
- [x] 4.6 Icon component (SVG stroke icons for all nav + action items)
- [x] 4.7 Avatar component
- [x] 4.8 Modal component (backdrop, close on Escape/click-outside)
- [x] 4.9 Field + inputStyle helpers for forms
- [x] 4.10 EditCell component (click to edit inline, blur to confirm)

## Stage 5 — Frontend: Data & Charts
- [x] 5.1 `frontend/src/api.js` — fetch wrapper for Go backend (`/api/data`, mutations)
- [x] 5.2 Formatters: `fmtINR`, `fmtCompact`, `fmtPct`, `fmtDate` (Indian numbering)
- [x] 5.3 Aggregation helpers: `classTotals(memberId)`, `memberTotal(memberId)`, `holdingsFor(memberId)`
- [x] 5.4 `DataContext` — global React context wrapping fetched data + mutation functions
- [x] 5.5 `AreaChart` — SVG sparkline (ResizeObserver-based, responsive)
- [x] 5.6 `Donut` — SVG donut chart for allocation
- [x] 5.7 `EdStack` — editorial segmented allocation bar
- [x] 5.8 `GainPill` — colored gain/loss pill with arrow icon

## Stage 6 — Frontend: Screens

### 6A — Dashboard
- [x] 6A.1 Statement header band (scope + date)
- [x] 6A.2 Thick hairline rule component (`EdRule`)
- [x] 6A.3 Hero block (serif total net worth + invested + gain line)
- [x] 6A.4 12-month area sparkline (data from History tab)
- [x] 6A.5 5-class allocation bar + per-class breakdown grid
- [x] 6A.6 Two-column lower section: member list (left) | monthly commitment + protection (right)
- [x] 6A.7 Italic disclaimer footnote

### 6B — Investments
- [x] 6B.1 Statement header + 4 summary tiles (click tile → switch tab)
- [x] 6B.2 Tab bar: Mutual Funds | Stocks | Gold & Silver | Insurance & Plans
- [x] 6B.3 MF table (editable: SIP amount, Invested, Current value) + totals bar
- [x] 6B.4 Stocks table (editable: Qty, Avg price, Last price) + totals bar
- [x] 6B.5 Metals table (editable: Grams, Buy rate/g, Today/g) + totals bar
- [x] 6B.6 Insurance table (editable: Premium, Paid so far, Value) + totals bar
- [x] 6B.7 "Add holding" modal (per-tab form, member picker)
- [x] 6B.8 "Save changes" button → batch PUT to backend for all dirty rows

### 6C — SIPs & Schedule
- [x] 6C.1 Statement header + 4 headline KPIs
- [x] 6C.2 "Lumpsum top-up" primary button
- [x] 6C.3 Manage SIPs table (editable: amount, day; status pill; edit button)
- [x] 6C.4 Edit SIP modal (amount, day, pause/resume → Save)
- [x] 6C.5 Monthly calendar grid (current month, SIP days highlighted)
- [x] 6C.6 Recent lumpsums list
- [x] 6C.7 Lumpsum modal → POST to backend
- [x] 6C.8 "Save changes" for SIP edits → batch PUT

### 6D — Family
- [x] 6D.1 Overview: hero net worth + member share bar (EdStack)
- [x] 6D.2 Member statement rows (allocation bar + SIP/holdings stats + click to drill)
- [x] 6D.3 Member profile: hero + allocation breakdown + SIPs list + top holdings
- [x] 6D.4 "View portfolio" → sets active member + navigates to Investments

### 6E — Tax
- [x] 6E.1 Statement header (Aarav · FY 2025-26)
- [x] 6E.2 4 headline figures (income, slab, cap gains, saved)
- [x] 6E.3 Surcharge runway bar (₹50L → ₹1Cr)
- [x] 6E.4 Income-split planner (range slider → live tax-saved calculation)
- [x] 6E.5 Per-parent capacity bars
- [x] 6E.6 Indicative disclaimer

## Stage 7 — Persistence
- [x] 7.1 Track "dirty" state per screen (which rows have unsaved edits)
- [x] 7.2 "Save changes" button appears when dirty; disabled + spinner while saving
- [x] 7.3 On save: batch PUT all dirty rows to `/api/{sheet}/{id}`
- [x] 7.4 On success: toast "Saved to Google Sheet ✓"; clear dirty state
- [x] 7.5 On error: toast "Save failed — check connection"; keep dirty state
- [x] 7.6 Add modal → POST → optimistic prepend to local list + refetch
- [x] 7.7 Delete row → DELETE → remove from local list + refetch

## Stage 8 — Docs & Polish
- [ ] 8.1 Loading skeleton (sidebar net worth "—", content shimmer bars)
- [ ] 8.2 Error boundary (friendly message if backend unreachable)
- [ ] 8.3 `README.md` — Google Sheet setup (tabs, headers, service account)
- [ ] 8.4 `README.md` — History tab: monthly workflow (how to append a net-worth snapshot)
- [ ] 8.5 `README.md` — Running locally (`go run ./backend`, open http://localhost:8080)
- [ ] 8.6 `.env.example` with `SPREADSHEET_ID`, `PORT`, `CREDENTIALS_PATH`
- [ ] 8.7 Mobile: sidebar collapses to bottom tab bar on screens < 768px

---

## Current Status
Starting Stage 1 now.
