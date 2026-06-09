# Kosh Test Generation Report

**Date:** 2026-06-09
**Project:** Kosh — Go + React wealth tracker
**Scope:** Backend (Go) + Frontend (Vitest)

---

## Executive Summary

| Layer | Test files | Test cases | Overall coverage |
|---|---|---|---|
| Backend (Go) | 4 | 57 | ~23% (handlers), ~12% (sheets) |
| Frontend (Vitest) | 5 | 94 | Mixed — see per-file table below |

The test suite covers the pure/utility layer thoroughly. The low overall numbers are driven entirely by untested I/O boundaries: Google Sheets API, Google Drive API, OAuth HTTP calls, and the Anthropic API call in the upload flow. None of those gaps require writing more tests today — they require targeted interface extraction refactors first (see Flagged section).

---

## Coverage Numbers

### Backend — per function

| File | Function | Coverage |
|---|---|---|
| `drive/client.go` | NewClient | 0% |
| `drive/client.go` | NewClientFromToken | 0% |
| `drive/client.go` | EnsureFolder | 0% |
| `drive/client.go` | findOrCreate | 0% |
| `drive/client.go` | Upload | 0% |
| `handlers/auth.go` | NewAuthHandler | 100% |
| `handlers/auth.go` | Login | 0% |
| `handlers/auth.go` | DemoLogin | 100% |
| `handlers/auth.go` | Me | 87.5% |
| `handlers/auth.go` | IsDemoSession | 100% |
| `handlers/auth.go` | Logout | 100% |
| `handlers/auth.go` | Require | 100% |
| `handlers/auth.go` | emailFromCookie | 100% |
| `handlers/auth.go` | signToken | 100% |
| `handlers/auth.go` | verifyToken | 80% |
| `handlers/data.go` | NewHandler | 75% |
| `handlers/data.go` | servingSampleData | 100% |
| `handlers/data.go` | sampleData | 80% |
| `handlers/data.go` | loadDevData | 40% |
| `handlers/data.go` | GetData | 66.7% |
| `handlers/data.go` | maybeSnapshotHistory | 0% |
| `handlers/data.go` | netWorthTotal | 0% |
| `handlers/data.go` | fetchFromSheets | 0% |
| `handlers/data.go` | parseConfig | 0% |
| `handlers/data.go` | parseFloat64 | 0% |
| `handlers/data.go` | writeJSON | 100% |
| `handlers/data.go` | computeFixedValues | 50% |
| `handlers/data.go` | computeFixedValue | 0% |
| `handlers/data.go` | fdCurrentValue | 0% |
| `handlers/data.go` | rdCurrentValue | 0% |
| `handlers/data.go` | yearsSince | 0% |
| `handlers/data.go` | monthsBetween | 0% |
| `handlers/mutations.go` | AddRow | 20% |
| `handlers/mutations.go` | UpdateRow | 33.3% |
| `handlers/mutations.go` | DeleteRow | 66.7% |
| `handlers/mutations.go` | sheetFromPath | 100% |
| `handlers/mutations.go` | normalizeSheet | 0% |
| `handlers/mutations.go` | init | 100% |
| `handlers/upload.go` | NewUploadHandler | 0% |
| `handlers/upload.go` | Handle | 0% |
| `handlers/upload.go` | parseWithClaude | 0% |
| `handlers/upload.go` | mimeFromFilename | 0% |
| `handlers/upload.go` | decryptPDF | 0% |
| `sheets/client.go` | NewClient | 0% |
| `sheets/client.go` | EnsureTabs | 0% |
| `sheets/client.go` | ReadSheet | 0% |
| `sheets/client.go` | AppendRow | 0% |
| `sheets/client.go` | UpdateRow | 0% |
| `sheets/client.go` | DeleteRow | 0% |
| `sheets/client.go` | ColStr | 100% |
| `sheets/client.go` | ColFloat | 100% |
| `sheets/client.go` | ColInt | 100% |
| `sheets/client.go` | EnvOrDefault | 100% |

**Package totals:** `kosh/sheets` 11.8% · `kosh/handlers` 23.4%

### Frontend — per file

| File | Statements | Branches | Functions |
|---|---|---|---|
| `src/data/api.js` | 100% | 100% | 100% |
| `src/data/context.jsx` | 100% | 100% | 100% |
| `src/data/helpers.js` | 100% | 72% | 100% |
| `src/components/Icons.jsx` | 100% | 100% | 100% |
| `src/components/Primitives.jsx` | 46.7% | 95.45% | 61.53% |
| `src/data/driveAuth.js` | 0% | 0% | 0% |
| `src/App.jsx` | 0% | 0% | 0% |
| `src/components/UploadZone.jsx` | 0% | 0% | 0% |
| `src/screens/Dashboard.jsx` | 0% | 0% | 0% |
| `src/screens/Investments.jsx` | 0% | 0% | 0% |
| `src/screens/SIPs.jsx` | 0% | 0% | 0% |
| `src/screens/Expenses.jsx` | 0% | 0% | 0% |
| `src/screens/Family.jsx` | 0% | 0% | 0% |
| `src/screens/Tax.jsx` | 0% | 0% | 0% |
| `src/screens/Login.jsx` | 0% | 0% | 0% |

---

## What Is Tested

### Backend

**`backend/sheets/client_test.go`** (34 cases)
Covers the four pure column-extraction helpers (`ColStr`, `ColFloat`, `ColInt`, `EnvOrDefault`) exhaustively: nil rows, out-of-bounds indices, type-assertion failures, empty string coercion, and environment variable fallback behaviour.

**`backend/handlers/auth_test.go`** (12 tests)
Covers the demo-mode auth surface: `DemoLogin` issues a valid JWT and sets the cookie; `Me` returns the correct identity for a demo session and 401s without a cookie; `Logout` clears the cookie; `IsDemoSession` and `Require` middleware chain correctly.

**`backend/handlers/mutations_test.go`** (7 tests)
Covers `AddRow`, `UpdateRow`, `DeleteRow` in demo mode (no live Sheets call) and bad-path rejection via `sheetFromPath`. Validates HTTP status codes and JSON error shapes.

**`backend/handlers/data_test.go`** (4 tests)
Smoke-tests `GetData` in demo mode: verifies the response is valid JSON, contains expected top-level keys, and returns HTTP 200. Does not assert financial values — that math is in the private helpers flagged below.

### Frontend

**`src/data/helpers.test.js`** (51 tests)
Full coverage of every exported helper: `fmtINR`, `fmtCompact`, `fmtPct`, `fmtDate`, `holdingsFor`, `classTotals`, `memberTotal`, `netWorth`, `nextPremiumDue`, `nextEmiDue`, and `upcomingOutflows`. Uses fixture data shaped after real sheet rows; edge cases include zero holdings, missing members, overdue premiums, and multiple upcoming outflows.

**`src/data/api.test.js`** (10 tests)
Mocks `fetch` globally; verifies `fetchData`, `getCached`, `addRow`, `updateRow`, and `deleteRow` construct correct URLs, send the right HTTP methods/bodies, and surface API error messages to the caller.

**`src/data/context.test.jsx`** (9 tests)
Mounts `DataProvider` with a mocked `api.js`; asserts the `useData` hook returns loading state, then populated data, then reflects optimistic mutations before the server round-trip completes.

**`src/components/Icons.test.jsx`** (7 tests)
Renders each named icon variant and checks the SVG is present in the DOM with the correct `aria-hidden` attribute.

**`src/components/Primitives.test.jsx`** (17 tests)
Covers `Avatar` (initials extraction, colour stability), `GainPill` (positive/negative/zero formatting), `Modal` (open/close, focus trap), `Field` (label association, validation message), `Toast` (auto-dismiss timer), and `SaveBar` (enabled/disabled state, save callback).

---

## Flagged for Refactoring

Each item below is a concrete blocker preventing test coverage. The recommended fix is the minimal change needed to make the code testable — not a full architecture rewrite.

---

### 1. `handlers/auth.go: Login` (0%)

**Problem:** `Login` makes a direct `http.Get("https://oauth2.googleapis.com/tokeninfo?...")` call using `http.DefaultClient`, which is not injectable. Any test would hit the real Google endpoint.

**Recommended fix:** Add an optional `httpClient` field (type `*http.Client`) to `AuthHandler`. Default it to `http.DefaultClient` in `NewAuthHandler`. Tests construct an `httptest.NewServer` that serves a canned tokeninfo JSON response and pass `server.Client()` to the handler.

**Approach:**
1. Add `httpClient *http.Client` to `AuthHandler` struct.
2. In `NewAuthHandler`, set `httpClient: http.DefaultClient`.
3. Replace `http.Get(...)` in `Login` with `h.httpClient.Get(...)`.
4. In `auth_test.go`, spin up `httptest.NewServer`, inject the client, and test the success path (valid token → cookie set) and failure paths (tokeninfo 400, email mismatch).

**Effort:** S

---

### 2. `handlers/data.go`: private financial math functions (all 0%)

**Problem:** `fdCurrentValue`, `rdCurrentValue`, `yearsSince`, `monthsBetween`, `parseConfig`, `parseFloat64`, `netWorthTotal`, and `computeFixedValue` are all private. They contain the core financial calculation logic but cannot be invoked from a `_test.go` file in a different package. `netWorthTotal` mirrors the frontend `netWorth` helper — divergence between the two would be invisible until a user sees wrong numbers.

**Recommended fix:** Either export them (rename to `FdCurrentValue`, etc.) within the `handlers` package, or — better — move them into a new `internal/finance` package with exported symbols. The `internal/` convention keeps them unavailable to external callers while making them directly testable.

**Approach:**
1. Create `backend/internal/finance/calc.go` with exported versions of the functions.
2. Update `handlers/data.go` to call the `finance` package.
3. Write `backend/internal/finance/calc_test.go` with table-driven tests for each function: known principal/rate/duration → expected maturity value; known birth date → expected years; known net-worth fixture → expected total.
4. Add a parity test that feeds the same fixture to both `finance.NetWorthTotal` and the frontend `netWorth` helper output (via a golden file) to catch drift.

**Effort:** S

---

### 3. `handlers/data.go: fetchFromSheets` (0%)

**Problem:** `fetchFromSheets` takes a concrete `*sh.Client` parameter (or uses the one stored on `Handler`). There is no seam to substitute a fake implementation in tests.

**Recommended fix:** Extract a `SheetsReader` interface:

```go
type SheetsReader interface {
    ReadSheet(name string) ([][]interface{}, error)
}
```

Inject it into `Handler` instead of the concrete `*sh.Client`. Write a `fakeSheetsReader` in `data_test.go` that returns fixture rows. `GetData` in live mode is then fully testable without a network connection.

**Approach:**
1. Define `SheetsReader` in `handlers/data.go` (or a shared `handlers/interfaces.go`).
2. Update `NewHandler` to accept `SheetsReader` instead of `*sh.Client`.
3. `sheets.Client` already satisfies the interface — no change needed in `sheets/`.
4. In `data_test.go`, implement `fakeSheetsReader` returning three or four fixture sheet payloads; assert that `GetData` returns correctly shaped JSON with the expected computed values.

**Effort:** M

---

### 4. `handlers/data.go: maybeSnapshotHistory` (0%)

**Problem:** `maybeSnapshotHistory` calls `AppendRow` on the live `*sheets.Client`. It has no test seam.

**Recommended fix:** Unblocked by item 3. Extend the interface from item 3 into a `SheetsReadWriter` (or add a separate `SheetsWriter` interface with `AppendRow`). Inject it into `Handler`. In tests, use a `fakeReadWriter` that records `AppendRow` calls; assert the snapshot is triggered on the correct cadence and that the row written contains the expected net-worth value.

**Approach:**
1. Add `AppendRow(sheet, row string) error` to the interface (or a separate writer interface).
2. Update `maybeSnapshotHistory` to use the interface.
3. In `data_test.go`, drive `GetData` twice with a time gap that crosses the snapshot threshold; verify `AppendRow` was called once with correct arguments.

**Effort:** M (same PR as item 3)

---

### 5. `handlers/upload.go` — all functions (0%)

**Problem:** Four distinct issues:
- `mimeFromFilename` and `decryptPDF` are private but have no I/O — pure string/byte operations that are trivially testable once exported.
- `parseWithClaude` uses `http.DefaultClient` directly for the Anthropic API call — same injectable-client pattern needed as item 1.
- `Handle` orchestrates Drive + Anthropic; it is fully testable once both client fixes land.

**Recommended fix:**
1. Export `mimeFromFilename` → `MimeFromFilename` and `decryptPDF` → `DecryptPDF`. Test them immediately with no other changes.
2. Add `httpClient *http.Client` to `UploadHandler`, inject via `NewUploadHandler`, use in `parseWithClaude`.
3. For `Handle`, mock Drive via the `DriveService` interface (item 7) and mock the Anthropic HTTP call via the injected client.

**Approach:**
1. Rename private functions, run existing tests to confirm no regression.
2. Write `upload_test.go`: table-driven tests for `MimeFromFilename` (pdf/jpg/png/unknown extensions); `DecryptPDF` with a minimal encrypted-PDF fixture vs. a plaintext PDF.
3. Spin up `httptest.NewServer` serving a canned Anthropic JSON response; wire it into `parseWithClaude` tests.
4. Full `Handle` integration test after Drive interface lands.

**Effort:** M

---

### 6. `sheets/client.go` — API methods (all 0%)

**Problem:** `NewClient`, `EnsureTabs`, `ReadSheet`, `AppendRow`, `UpdateRow`, `DeleteRow` all instantiate or call live Google Sheets API objects. There is no way to inject a fake service.

**Recommended fix:** Extract a `SheetsService` interface that wraps the methods on `*sheets.Service` used by `Client`. Implement a `fakeSheetsService` in `sheets/client_test.go`. Inject `SheetsService` into `Client` via constructor.

**Approach:**
1. Identify the exact `*sheets.Service` methods called (`spreadsheets.values.get`, `batchUpdate`, `append`, `update`, `clear`).
2. Define `SheetsService` with those method signatures.
3. Add `svc SheetsService` field to `Client`; update `NewClient` to populate it.
4. Write `fakeSheetsService` returning fixture responses; test `ReadSheet` row parsing, `EnsureTabs` idempotency (sheet already exists → no create call), `AppendRow` payload construction, `ColStr`/`ColFloat`/`ColInt` edge cases in a round-trip.

**Effort:** M

---

### 7. `drive/client.go` — all functions (all 0%)

**Problem:** Every function requires a live `*drive.Service`. `EnsureFolder` has meaningful tree-walking logic (find or create by name under a parent) that is worth testing independently of the network.

**Recommended fix:** Same interface-extraction pattern as item 6. Define a `DriveService` interface wrapping the Drive API methods used (`files.list`, `files.create`). Inject into `Client` via constructor.

**Approach:**
1. Extract `DriveService` interface with `List(query)` and `Create(file)` signatures.
2. Implement `fakeDriveService` in `drive/client_test.go`.
3. Test `EnsureFolder`: (a) folder exists → returns existing ID, no create call; (b) folder missing → create called with correct metadata; (c) nested path → two sequential find-or-create calls.
4. Test `Upload`: verify the file metadata, MIME type, and parent folder ID sent to the fake.

**Effort:** M

---

### 8. `handlers/mutations.go` — live-mode paths (partially covered)

**Problem:** `AddRow`, `UpdateRow`, `DeleteRow` are tested only in demo mode (7 tests). The live-mode paths — which call `AppendRow`/`UpdateRow`/`DeleteRow` on `*sheets.Client` — are at 20–67% because the live branches are unreachable without a real Sheets client.

**Recommended fix:** Unblocked by item 3/6. Once `SheetsReader`/`SheetsWriter` interfaces exist, inject them into the mutations handler. Then test: column ordering in `AddRow`, JSON-to-row serialisation, unknown sheet rejection, and partial-update semantics in `UpdateRow`.

**Approach:**
1. After interface work lands, add a `fakeSheetsWriter` to `mutations_test.go`.
2. Table-drive `AddRow` with each known sheet name; assert the row slice matches the expected column order.
3. Test `UpdateRow` with a partial body; assert only the specified columns are updated.
4. Test `DeleteRow` with a valid and an out-of-range row index.

**Effort:** S (after interface work from items 3 and 6)

---

### 9. `handlers/mutations.go: normalizeSheet` (0%)

**Problem:** `normalizeSheet` is a private helper referenced only in a comment inside `init()`. It is not called from any exported path in the current codebase — the routing in `main.go` never wires it up. It is dead code.

**Recommended fix:** Decision required before writing tests: either (a) wire it into the routing and test it as part of `sheetFromPath`, or (b) delete it. Do not write tests for dead code.

**Approach:**
1. Search `main.go` and all handler registration sites for calls to `normalizeSheet`.
2. If the function is genuinely unused, open a PR to delete it and update `init()` comment accordingly.
3. If it was meant to be used, wire it up and cover it via `sheetFromPath` tests.

**Effort:** S

---

### 10. `src/data/driveAuth.js` (0%)

**Problem:** `driveAuth.js` depends on `window.google.accounts.oauth2`, a global injected by the GIS SDK script tag at runtime. JSDOM does not provide it. Additionally, `inFlight` and `cachedToken` are module-level variables — their state leaks between test cases if not reset.

**Recommended fix:** Wrap the GIS SDK behind a thin interface. Expose a module-level setter `setTokenClientFactory(fn)` that tests can call to inject a fake. The real implementation calls `window.google.accounts.oauth2.initTokenClient`; the fake immediately invokes the callback with a fixture token.

**Approach:**
1. Extract the `google.accounts.oauth2.initTokenClient` call into a factory function; export `setTokenClientFactory` for test injection.
2. In `driveAuth.test.js`, call `setTokenClientFactory` with a jest/vitest stub before each test.
3. Add `afterEach(() => { resetModuleState() })` — export a `_resetForTesting()` function that zeroes `inFlight` and `cachedToken`.
4. Test: first call → token requested and cached; second call before expiry → cached token returned without new request; expired token → new request made.

**Effort:** S

---

### 11. `src/data/helpers.js: upcomingOutflows` — SIP branch date dependency (branch coverage 72%)

**Problem:** The SIP calculation inside `upcomingOutflows` uses module-level constants `TODAY`, `TODAY_YEAR`, `TODAY_MONTH` computed once at import time. This makes SIP date math non-deterministic in tests — it depends on when the test suite runs. The insurance and EMI branches accept a `from` parameter; the SIP branch does not.

**Recommended fix:** Add an optional `from` parameter to `upcomingOutflows` (defaulting to `new Date()`) and thread it through the SIP date arithmetic, replacing the module-level constants.

**Approach:**
1. Change the signature to `upcomingOutflows(data, opts = {})` where `opts.from` defaults to `new Date()`.
2. Derive `todayYear`, `todayMonth` from `opts.from` inside the function instead of from module-level constants.
3. Update the 3 SIP-specific branches in `helpers.test.js` with a fixed `from` date; assert exact outflow counts and values.
4. The 72% branch coverage in helpers.js will reach 100% once these branches are deterministically exercisable.

**Effort:** S

---

### 12. `src/components/Primitives.jsx` — AreaChart, EdStack, EditCell, MemberTag (uncovered)

**Problem:** `AreaChart`, `EdStack`, `EditCell`, and `MemberTag` have no tests. `AreaChart` computes SVG path geometry; `EditCell` has a click-to-edit lifecycle with keyboard and blur handling. These contain enough logic to warrant independent tests.

**Recommended fix:** Extend `Primitives.test.jsx` with targeted render + interaction tests for each component.

**Approach:**
1. `AreaChart`: render with a 5-point fixture data array; assert an SVG `<path>` element is present and its `d` attribute is non-empty and starts with `M`.
2. `EdStack`: render with mock column definitions; assert columns render in order.
3. `EditCell`: render a cell in read mode; `userEvent.click` → assert input appears with current value; type new value + `Enter` → assert `onChange` called with new value; `Escape` → assert original value restored without calling `onChange`.
4. `MemberTag`: render with a member fixture; assert initials and colour class match.

**Effort:** S–M

---

### 13. `src/components/UploadZone.jsx` (0%)

**Problem:** `UploadZone` combines three untestable-by-default dependencies: the browser `fetch` API, the `driveAuth` module (GIS SDK), and file input interaction. None of these work in JSDOM without mocking.

**Recommended fix:** Mock all three at the test boundary.

**Approach:**
1. `vi.stubGlobal('fetch', vi.fn())` — configure it to return a canned upload response.
2. `vi.mock('../data/driveAuth', () => ({ getAccessToken: vi.fn().mockResolvedValue('fake-token') }))`.
3. Render `<UploadZone />`; select the hidden file input via `screen.getByTestId('file-input')` (add `data-testid` if not present); use `Object.defineProperty(input, 'files', { value: [new File(['%PDF'], 'test.pdf', { type: 'application/pdf' })] })` then `fireEvent.change(input)`.
4. Test states: idle → uploading (spinner visible) → success (file name shown) → error (error message shown on fetch rejection).
5. Test the password field: when a 400 response body contains `"password_required"`, the password input appears; submitting it retries the fetch with the password included.

**Effort:** M

---

### 14. `src/screens/*.jsx` — all screen components (all 0%)

**Problem:** All seven screen components (`Dashboard`, `Investments`, `SIPs`, `Expenses`, `Family`, `Tax`, `Login`) are at 0% coverage. They are large, stateful, and depend on the full `DataProvider` context plus `react-router` routing.

**Recommended fix:** Create a `renderWithProviders` test helper (in `src/test-utils.jsx`) that wraps the component under test in `DataProvider` with a mocked `api.js` module and a `MemoryRouter`. This is the same pattern already used in `context.test.jsx` — extract it and share it.

**Approach:**
1. Extract `renderWithProviders(ui, { initialData })` into `src/test-utils.jsx`.
2. Start with `Login.jsx`: no data dependency; test that the Google sign-in button is rendered and that clicking it calls the auth flow.
3. `Family.jsx`: render with a fixture containing 2 members; assert both member names appear; test add-member form submission calls `addRow` with the correct sheet and payload.
4. `Investments.jsx`, `SIPs.jsx`, `Expenses.jsx`: render with fixture data; assert summary totals match fixture values computed by `helpers.js`.
5. `Dashboard.jsx`, `Tax.jsx`: render with fixture data; assert key derived values (net worth, tax liability) appear in the DOM.
6. Each screen: test the empty-state (no data) render path separately.

**Effort:** L overall (M per screen)

---

### 15. `src/App.jsx` (0%)

**Problem:** `App.jsx` integrates Google Sign-In via the GIS SDK, performs a cookie-based auth check on mount (`/api/auth/me`), and handles screen routing. It depends on both the GIS global and the `fetch` API.

**Recommended fix:** Fully testable once items 10 (`driveAuth.js` refactor) and 14 (screen helpers + `renderWithProviders`) are in place. The remaining blocker is mocking the two startup fetch calls.

**Approach:**
1. `vi.stubGlobal('fetch', mockFetch)` — configure `mockFetch` to return a 401 for `/api/auth/me` (unauthenticated state) or a 200 with a user object (authenticated state).
2. Mock the GIS SDK via the factory injected in item 10.
3. Test: unauthenticated → `Login` screen rendered; authenticated → `Dashboard` screen rendered; sign-out → `Login` screen returned.
4. Test the `config` fetch (`/api/auth/config`) that gates whether Google sign-in is shown vs. demo mode.

**Effort:** M (after items 10 and 14)

---

## Priority Order

Items are ranked by the ratio of coverage unlocked to implementation effort. Items that unblock other items are ranked above their dependents.

| Rank | Item | Effort | Coverage unlocked | Blocks |
|---|---|---|---|---|
| 1 | **#2** — Export private financial math | S | 8 functions in data.go; enables parity test with frontend | — |
| 2 | **#11** — `upcomingOutflows` `from` param | S | 3 SIP branches → helpers.js 100% branch | — |
| 3 | **#1** — Injectable `httpClient` in `AuthHandler` | S | `Login` fully covered | #15 (partially) |
| 4 | **#10** — `driveAuth.js` GIS SDK abstraction | S | driveAuth.js 100% | #15 |
| 5 | **#9** — Resolve dead `normalizeSheet` | S | Cleanup; removes false 0% signal | — |
| 6 | **#3 + #4** — `SheetsReader` interface in data.go | M | `fetchFromSheets`, `maybeSnapshotHistory`, `GetData` live mode | #8 |
| 7 | **#6** — `SheetsService` interface in sheets/ | M | All 6 sheets API methods | #3, #4, #8 |
| 8 | **#7** — `DriveService` interface in drive/ | M | All 5 drive functions | #5, #13, #15 |
| 9 | **#5** — Export + inject in upload.go | M | All upload functions | depends on #1, #7 |
| 10 | **#8** — Mutations live-mode tests | S | AddRow/UpdateRow/DeleteRow live paths | depends on #3, #6 |
| 11 | **#12** — Primitives: AreaChart, EditCell, EdStack, MemberTag | S–M | Primitives.jsx → ~80%+ stmts | — |
| 12 | **#13** — `UploadZone.jsx` tests | M | UploadZone 100% | depends on #10 |
| 13 | **#14** — Screen component tests | L | All 7 screens (largest coverage gain overall) | depends on #10 |
| 14 | **#15** — `App.jsx` tests | M | App.jsx 100% | depends on #10, #14 |

**Recommended sprint order:** complete items 1–5 (all S effort, no dependencies) first. Then tackle items 6 and 7 in parallel (the two interface-extraction refactors). Everything else becomes unblocked once those land.
