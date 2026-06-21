# Kosh Refactor Plan

Status: **DRAFT — pending review/ack**. No code changes have been made yet.

Based on the test-generation-report findings plus a fresh read of all frontend
screens (`Investments.jsx`, `SIPs.jsx`, `Expenses.jsx`, `Family.jsx`, `Tax.jsx`,
`App.jsx`, `Dashboard.jsx`, `Login.jsx`) and the backend (`data.go`, `main.go`,
`mutations.go`, `upload.go`). Each phase is independently shippable, builds on
the existing 128-test safety net, and can be ack'd separately.

---

## Guiding domain model (first principles)

The data has 4 recurring shapes, currently duplicated per-entity:

| Concept | Entities | Shared shape |
|---|---|---|
| **Holding** | MF, Stock, Metal, Fixed, Insurance, NPS | `{ Invested, CurrentValue, Member }` |
| **Liability** | Loan | `{ Principal, Outstanding, Rate, Member }` |
| **Recurring commitment** | SIP, Loan EMI, Insurance Premium | `{ Amount, DayOfMonth/Frequency, NextDue() }` |
| **Sheet-backed record** | All 11 entity types | `{ ID, ...fields }` ↔ one Sheets row, column order = struct field order |

The single biggest structural problem on the backend is that the "sheet-backed
record" shape is expressed in **4 separate places per entity** (`koshTabs` in
main.go, `sheetColumns` in mutations.go, positional `ColStr/ColFloat/ColInt`
reads in `fetchFromSheets`, and the Go struct itself) — confirmed all 11 are
currently consistent, but every new field requires editing all 4. A generic,
struct-tag-driven repository collapses this to **1 place**.

On the frontend, the "tabbed table + inline EditCell + dirty-state + SaveBar +
Add modal" pattern is duplicated 8 times (6 tables in `Investments.jsx` +
`SIPs`/`Expenses`), with `scope()`/`memberOf()`/`KICK` style copy-pasted across
6 files.

---

## Backend plan

### Target package layout

```
backend/
  main.go                    — composition root only (~60 lines)
  internal/
    config/                  — env loading → Config struct
    models/                  — domain types + pure business logic, zero I/O
      types.go                 Member, MFRow, Stock, Metal, Fixed, Insurance, Loan, SIP, NPS, Lumpsum, History, Config, Data
      finance.go               FDCurrentValue, RDCurrentValue, YearsSince, MonthsBetween, ComputeFixedValue(s)  — exported, pure
      networth.go              NetWorthTotal
      finance_test.go, networth_test.go
    store/
      api.go                 — SheetsAPI interface (ReadSheet, AppendRow, UpdateRow, DeleteRow, EnsureTabs)
      client.go              — existing sheets.Client, now implements SheetsAPI
      repository.go          — generic Repository[T] (struct-tag column mapping)
      repository_test.go     — tested against an in-memory FakeSheetsAPI
    drive/
      api.go                 — DriveAPI interface (EnsureFolder, Upload)
      client.go              — existing client, implements DriveAPI
    auth/
      session.go             — token Sign/Verify (pure, already tested)
      googleverify.go        — TokenVerifier interface + GoogleTokenVerifier (injectable httpClient)
      handler.go             — Login, DemoLogin, Me, Logout, Require middleware
    documents/
      pdf.go                 — DecryptPDF, MimeFromFilename (exported, pure)
      claude.go              — DocumentParser interface + ClaudeParser (injectable httpClient)
      handler.go             — orchestrates DriveAPI + DocumentParser + pdf helpers
    api/
      server.go              — router + middleware chain (cors, security headers, auth)
      data.go                — GetData: assembles models.Data from repositories + computed fields
      mutations.go           — generic AddRow/UpdateRow/DeleteRow over a repository registry
      history.go             — maybeSnapshotHistory
```

### The centerpiece: generic `Repository[T]`

Each entity struct gets a `sheet:"..."` tag per field, in the exact order the
Sheets columns are in today:

```go
type MFRow struct {
    ID       string  `sheet:"id"`
    Name     string  `sheet:"name"`
    Plan     string  `sheet:"plan"`
    Platform string  `sheet:"platform"`
    Member   string  `sheet:"member"`
    Invested float64 `sheet:"invested"`
    Current  float64 `sheet:"current"`
    SIP      float64 `sheet:"sip"`
    Notes    string  `sheet:"notes"`
}
```

```go
type Repository[T any] struct {
    api     store.SheetsAPI
    sheet   string
    columns []string // derived once via reflection from `sheet` tags, in field order
}

func NewRepository[T any](api store.SheetsAPI, sheet string) *Repository[T]
func (r *Repository[T]) All() ([]T, error)
func (r *Repository[T]) Add(item T) (int, error)
func (r *Repository[T]) Update(id string, patch map[string]any) error
func (r *Repository[T]) Delete(id string) error
func (r *Repository[T]) Columns() []string // feeds EnsureTabs, replaces koshTabs entries
```

This:
- Replaces `koshTabs` (main.go), `sheetColumns` (mutations.go), and all 11
  hand-written `ColStr/ColFloat/ColInt` blocks in `fetchFromSheets` — **one
  source of truth: the struct definition.**
- `Fixed.CurrentValue` stays a normal tagged field (read from the sheet like
  anything else); `models.ComputeFixedValues(fixed)` overwrites it post-read —
  no special "skip" tag needed, repository stays fully generic.
- `Repository[T]` is tested once against a `FakeSheetsAPI`, and every entity
  gets read/write coverage for free — directly resolves test-report items
  **#3, #6, #8**.
- HTTP mutation handlers become a thin dispatch over a `map[string]AnyRepository`
  registry built in `main.go` — no more per-sheet `switch`.

### Other backend changes

- **`auth`**: split pure `Sign`/`Verify` (already well-tested) from
  `GoogleTokenVerifier` (injectable `httpClient`) — resolves **#1**.
- **`documents`**: export `DecryptPDF`/`MimeFromFilename` as pure functions,
  wrap Claude calls behind a `DocumentParser` interface — resolves **#5**.
- **`drive`**: add `DriveAPI` interface + fake — resolves **#7**.
- **Delete `normalizeSheet`** (confirmed dead code, item **#9**).
- **`api/server.go`**: replace the repeated
  `cors(w); if OPTIONS return; protect(...)` boilerplate per route with a
  small middleware chain (`Chain(handler, cors, requireAuth)`).
- **`maybeSnapshotHistory`**: becomes a method on the History repository —
  resolves **#4**.

---

## Frontend plan

### Target layout

```
frontend/src/
  data/
    api.js, context.jsx        — unchanged
    format.js                  — fmtINR, fmtCompact, fmtPct, fmtDate
    aggregate.js                — holdingsFor, classTotals, memberTotal, netWorth, scope, memberOf
    schedule.js                  — nextPremiumDue, nextEmiDue, upcomingOutflows, daysLeft, fmtCountdown
                                   (fixes TODAY coupling: takes optional `from` param, item #11)
    constants.js                  — CLASS_META, ED_ORDER/COL/LABEL, AC_COLOR/LABEL, OUTFLOW_COLOR/LABEL, urgency thresholds
    tokens.js                      — KICK, SERIF, spacing/type scale (currently copy-pasted in 6 files)
    driveAuth.js                    — refactored: createDriveAuth(googleAccounts) factory, no module-level state (item #10)
  auth/
    useAuth.js                       — auth state machine extracted from App.jsx
    googleSDK.js                      — shared GIS SDK loader (used by Login.jsx + driveAuth.js)
  components/
    Icons.jsx, Primitives.jsx          — slimmed (Avatar, Modal, Field, Toast, SaveBar, EdRule, EdStack, GainPill, AreaChart)
    UploadZone.jsx
    shared/
      EditableTable.jsx                  — column-config-driven table + dirty-state + totals row
      SegmentedButtons.jsx                 — categorical toggle (used 4+ places in Add modals)
      FormGrid.jsx                          — N-column field grid wrapper
      StatBox.jsx, SectionHead.jsx, DataListItem.jsx, ColorLabel.jsx
  components/layout/
    Sidebar.jsx, MemberSwitcher.jsx, AppShell.jsx, LoadingScreen.jsx, ErrorScreen.jsx
  screens/
    investments/
      index.jsx, SummaryTiles.jsx
      tables/{MF,Stock,Fixed,NPS,Metal,Insurance}Table.jsx
      AddModal/{index,MFForm,StockForm,MetalForm,InsuranceForm,FixedForm,NPSForm}.jsx
      NPSImportModal.jsx, formMappers.js, countdown.js
    Dashboard.jsx, SIPs.jsx, Expenses.jsx, Family.jsx, Tax.jsx, Login.jsx  — refactored to use shared/* + tokens.js
  App.jsx                               — slimmed to composition (useAuth + AppShell)
```

### Key new abstraction: `EditableTable`

Every one of the 6 Investments tables + `Expenses`'s `LoanTable` follows: rows
→ columns (some `EditCell`-editable, some computed) → member tag → totals row.
A column-config component replaces all 7:

```jsx
<EditableTable
  rows={mfRows}
  columns={[
    { key: 'name', label: 'Fund', render: r => <><strong>{r.name}</strong><div className="cell-sub">{r.plan}</div></> },
    showOwner && { key: 'member', label: 'Owner', render: r => <MemberTag member={memberOf(data, r.member)} /> },
    { key: 'sip', label: 'Monthly SIP', editable: true, format: fmtINR },
    { key: 'invested', label: 'Invested', editable: true, format: fmtINR },
    { key: 'current', label: 'Current value', editable: true, format: fmtINR },
  ]}
  totals={{ inv: r => r.invested, cur: r => r.current, label: 'Mutual funds' }}
  dirty={dirty} onMark={mark}
/>
```

`StockTable`/`MetalTable`/`NPSTable` keep their *computed* columns
(`qty × avg_price` etc.) via a `compute` fn per column — the table component
itself stays generic.

### Other frontend changes

- **`AddModal`** (currently one 185-line component with 6 `if(tab===...)`
  branches) splits into one form component per asset class under
  `AddModal/`, sharing `FormGrid` + `SegmentedButtons` + the member picker.
- **App.jsx** (214 → ~60 lines): `Sidebar`, `MemberSwitcher`, `LoadingScreen`,
  `ErrorScreen`, `AppShell` move to `components/layout/`; the auth state
  machine (`authState`/`user`/`clientId`/login/logout handlers) moves to
  `auth/useAuth.js` — resolves **#15**.
- **`Login.jsx`**: GIS SDK polling extracted to `auth/googleSDK.js`, shared
  with `driveAuth.js`.
- **`driveAuth.js`**: factory pattern removes module-level
  `inFlight`/`cachedToken`, makes it mockable — resolves **#10**.
- **Dedup `scope()`/`memberOf()`/`KICK`**: currently copy-pasted in
  `Investments`, `SIPs`, `Expenses`, `Family`, `Dashboard`, `Login` → move to
  `data/aggregate.js` and `data/tokens.js`.
- **`Tax.jsx`** and `Family.jsx` are genuinely different (calculator /
  read-only dashboard) — they get tokens/StatBox/DataListItem treatment but
  **not** EditableTable.

---

## Suggested sequencing

| Phase | Scope | Risk | Resolves report items |
|---|---|---|---|
| **B1** | Backend: extract `internal/models` (types + pure finance fns), export `FDCurrentValue` etc. | Low — pure functions, no I/O change | #2 |
| **B2** | Backend: `SheetsAPI`/`DriveAPI` interfaces + generic `Repository[T]`, replace `koshTabs`/`sheetColumns`/`fetchFromSheets`, delete `normalizeSheet` | Medium-high — touches I/O layer, but interface+fake gives full test coverage | #3, #6, #7, #8, #9 |
| **B3** | Backend: split `handlers` → `auth`/`documents`/`api`, injectable Google/Claude HTTP clients, middleware chain, `main.go` → composition root | Medium — mostly mechanical moves | #1, #4, #5 |
| **F1** | Frontend: split `helpers.js` → `format/aggregate/schedule/constants/tokens`, fix `TODAY` coupling, dedupe `scope`/`memberOf`/`KICK`, refactor `driveAuth.js` | Low — pure refactor, fully testable | #10, #11 |
| **F2** | Frontend: build `shared/` component library (`EditableTable`, `SegmentedButtons`, `FormGrid`, `StatBox`, `SectionHead`, `DataListItem`, `ColorLabel`) | Medium — new components need their own tests before adoption | #12 |
| **F3** | Frontend: decompose `Investments.jsx` into `screens/investments/`; migrate `SIPs`/`Expenses` tables to `EditableTable`; extract `App.jsx` layout + `auth/useAuth` | High effort, do screen-by-screen | #13, #14, #15 |

Each phase keeps the app behaviorally identical (128 tests + manual smoke-test
as a regression net), so they can land as separate PRs.

---

## Open questions to resolve before starting

1. For `helpers.js` — update all ~10 import sites directly to the new split
   modules, or keep `helpers.js` as a temporary re-export barrel during
   migration? -> update all
2. Backend `Repository.Update` semantics — should it stay "merge a JSON patch
   into the existing row" (current behavior) or move to "replace the full
   row"? This affects the generic interface shape. -> stay merge/patch
3. Tackle all of B1–B3 and F1–F3, or start with a subset (e.g., just
   B1+B2+F1, the highest-value/lowest-risk wins) and re-evaluate? -> tackle all
