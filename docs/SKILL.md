---
name: kosh-migrate
description: "Migrate existing financial data from any Google Sheet or CSV into the Kosh sheet format, tab by tab. Handles field mapping, ID generation, member-ID linking, and date normalisation. Produces paste-ready rows for each tab."
trigger: /kosh-migrate
---

# /kosh-migrate

Migrate existing financial data into the Kosh Google Sheet format without manual re-entry.

## What this skill does

Given a user's existing spreadsheet data (pasted rows, CSV export, or a description of their sheet), this skill:
1. Starts with the **Members** tab (because every other tab references `member` IDs).
2. Works through each asset tab in order: **MF → Stocks → Metals → Fixed → Insurance → SIPs → Lumpsums → History → Config**.
3. Maps the user's column names to Kosh's schema intelligently.
4. Auto-generates stable IDs, normalises dates to YYYY-MM-DD, and fills sensible defaults.
5. Outputs a **paste-ready table** for each tab (tab-separated, copy → paste directly into Google Sheets).

---

## How to run

When `/kosh-migrate` is invoked:

1. **Greet and orient.** Tell the user:
   > "I'll help you migrate your data into Kosh's Google Sheet format tab by tab. We'll start with Members — the IDs defined there are referenced by every other tab. For each tab, paste your existing rows (or describe what you have) and I'll produce rows ready to paste into your new sheet."

2. **Work tab by tab in this order:**
   Members → MF → Stocks → Metals → Fixed → Insurance → SIPs → Lumpsums → History → Config

3. **For each tab:**
   a. Show the required schema (column headers + types).
   b. Ask the user to paste their existing data for that tab, or describe what they have.
   c. If they have nothing for a tab, offer an example row and move on.
   d. Map their columns → Kosh columns (ask about ambiguous mappings).
   e. Output the migrated rows as a paste-ready table.
   f. Confirm before moving to the next tab.

4. **At the end**, summarise what was migrated and list any tabs that were skipped or left empty.

---

## Tab schemas and migration rules

### MEMBERS
**Purpose:** One row per family member. IDs here are referenced in every other tab.

**Headers (row 1):**
```
id | name | full_name | relation | slab | color
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Short, stable, lowercase. Suggest: `you` for self, `mom`, `dad`, etc. Never spaces. |
| name | string | Short display name (e.g. `Jenish`, `Mom`). Used in chips and tags. |
| full_name | string | Full display name (e.g. `Jenish (You)`, `Mom`). Used in headers. |
| relation | string | `Self` / `Mother` / `Father` / `Spouse` / `Child` |
| slab | number | Income tax slab % as integer: `5`, `20`, or `30` |
| color | hex string | Assign a distinct hex color per member. Suggestions: `#1C4A3A`, `#C98A5E`, `#7E8AA2`, `#A87C2A` |

**Migration rules:**
- Always create the self-member row first with `id = you`.
- If the source has no color column, assign colors in order from the suggestions above.
- If slab is missing, default to `30` for self and `5` for parents (senior citizens).

**Example output:**
```
id	name	full_name	relation	slab	color
you	Jenish	Jenish (You)	Self	30	#1C4A3A
mom	Mom	Mom	Mother	5	#C98A5E
dad	Dad	Dad	Father	5	#7E8AA2
```

---

### MF (Mutual Funds)
**Purpose:** One row per mutual fund holding.

**Headers (row 1):**
```
id | name | plan | platform | member | invested | current | sip | notes
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `mf1`, `mf2`, … in order |
| name | string | Full fund name as shown on the platform |
| plan | string | e.g. `Direct · Growth`. If unknown, use `Direct · Growth` |
| platform | string | e.g. `Groww`, `ET Money`, `Kuvera`, `MF Central` |
| member | string | Must match a Members.id exactly |
| invested | number | Total amount invested (cost basis). No ₹ or commas. |
| current | number | Current market value. If unknown, use the same as invested. |
| sip | number | Monthly SIP amount for this fund. `0` if no active SIP. |
| notes | string | Optional. Leave empty if nothing to say. |

**Migration rules:**
- If the source has a single "Amount" column with no invested/current split, use that value for both.
- If the source has multiple SIP rows for the same fund, sum them into one row.
- Strip ₹ symbols and commas from all numeric fields.
- If member is not specified in source, default to `you`.

---

### STOCKS
**Purpose:** One row per stock holding.

**Headers (row 1):**
```
id | name | ticker | qty | avg_price | last_price | member
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `st1`, `st2`, … |
| name | string | Full company name |
| ticker | string | NSE/BSE ticker symbol (e.g. `RELIANCE`, `HDFCBANK`) |
| qty | number | Number of shares held |
| avg_price | number | Average buy price per share |
| last_price | number | Current market price per share. If unknown, use avg_price. |
| member | string | Must match a Members.id |

**Migration rules:**
- If source has "Investment value" and "Qty" but no avg_price, compute: `avg_price = investment / qty`.
- If source has multiple buy lots for the same stock, combine: `total_qty`, weighted avg_price, and latest last_price.
- Ticker may need to be looked up if only company name is provided — ask the user to confirm.

---

### METALS
**Purpose:** One row per physical metal purchase.

**Headers (row 1):**
```
id | type | date_purchased | grams | buy_rate | today_price | place | member
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `me1`, `me2`, … |
| type | string | `Gold` or `Silver` |
| date_purchased | date | YYYY-MM-DD. Convert from DD/MM/YYYY or any other format. |
| grams | number | Weight in grams |
| buy_rate | number | ₹ per gram at time of purchase |
| today_price | number | Current ₹ per gram. If unknown, use buy_rate. |
| place | string | Where purchased (jeweller name, platform, etc.) |
| member | string | Must match a Members.id |

**Migration rules:**
- If source has total value rather than per-gram rate, compute: `buy_rate = total_value / grams`.
- Sovereign Gold Bonds count as Gold — note `SGBs` in the place field.
- Gold ETF/FOF units are MF holdings, not Metals — suggest moving those to the MF tab instead.

---

### FIXED
**Purpose:** One row per fixed-income instrument (FD, RD, PPF, Bonds, etc.).

**Headers (row 1):**
```
id | kind | name | member | principal | rate | current_value | opened | matures | monthly
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `fd1`, `fd2`, … |
| kind | string | `FD` / `RD` / `PPF` / `Bonds` / `NPS` / `Provident Fund` |
| name | string | Descriptive name (e.g. `HDFC Bank FD`, `GPF Account`) |
| member | string | Must match a Members.id |
| principal | number | Original amount deposited / invested |
| rate | number | Interest rate % p.a. (e.g. `7.1`) |
| current_value | number | Current value with accrued interest. If unknown, use principal. |
| opened | date | YYYY-MM-DD — date account/FD was opened |
| matures | date | YYYY-MM-DD — maturity date |
| monthly | number | Monthly interest payout. `0` for cumulative / non-paying instruments. |

---

### INSURANCE
**Purpose:** One row per insurance policy.

**Headers (row 1):**
```
id | name | type | member | premium | freq | paid | value | cover | maturity
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `in1`, `in2`, … |
| name | string | Policy name (e.g. `LIC Jeevan Anand`, `HDFC Click 2 Protect`) |
| type | string | `Term` / `Endowment` / `ULIP` / `Health` / `Accident` |
| member | string | Life insured — must match a Members.id |
| premium | number | Premium amount per frequency cycle |
| freq | string | `annual` / `monthly` / `quarterly` / `half-yearly` |
| paid | number | Total premiums paid so far (premium × number of payments) |
| value | number | Current surrender value or fund value. `0` for term plans. |
| cover | number | Sum assured (death benefit). `0` for health-only plans. |
| maturity | number | Year of maturity as 4-digit integer (e.g. `2038`). `0` if no maturity. |

---

### SIPs
**Purpose:** One row per SIP mandate (active or paused).

**Headers (row 1):**
```
id | fund | member | amount | day | status | start_date | platform
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `sip1`, `sip2`, … |
| fund | string | Short fund name used in the SIP calendar (can abbreviate — e.g. `HDFC NIFTY 50 Index`) |
| member | string | Must match a Members.id |
| amount | number | Monthly debit amount |
| day | number | Debit day of month (1–28) |
| status | string | `active` or `paused` |
| start_date | date | YYYY-MM-DD — when the SIP was first set up |
| platform | string | Platform where the SIP is registered (e.g. `Groww`, `ET Money`) |

**Migration rules:**
- If source doesn't have day of month, ask the user; default to `5` if they don't know.
- Cancelled/completed SIPs should be omitted entirely (they are not the same as paused).
- If source has a single "SIP amount" column for the whole portfolio without per-fund breakdown, ask the user to confirm amounts per fund before generating rows.

---

### LUMPSUMS
**Purpose:** One row per one-off top-up investment (outside the monthly SIP cycle).

**Headers (row 1):**
```
id | fund | member | amount | date
```

| Column | Type | Rules |
|--------|------|-------|
| id | string | Auto-generate: `lp1`, `lp2`, … |
| fund | string | Fund name (matches the name used in MF tab) |
| member | string | Must match a Members.id |
| amount | number | Amount of the one-off investment |
| date | date | YYYY-MM-DD — date of the investment |

**Migration rules:**
- Only include lumpsum entries that are *additional* to SIP instalments. Regular SIP debits are tracked in the SIPs tab.
- If the user has a transaction log, filter for non-SIP purchase transactions.
- If dates are missing, ask the user for approximate dates; do not guess.

---

### HISTORY
**Purpose:** Monthly net-worth snapshots for the Dashboard sparkline.

**Headers (row 1):**
```
month | value
```

| Column | Type | Rules |
|--------|------|-------|
| month | string | 3-letter month + 2-digit year: `Jun 26`, `May 26`, … Oldest first. |
| value | number | Total household net worth that month (all members combined) |

**Migration rules:**
- If the source has a monthly portfolio tracker, use those totals.
- If no history exists, ask the user if they want to start fresh (only the current month) or estimate a few past months from memory.
- If history exists in a different month-label format (e.g. `June 2026`, `06-2026`), convert to `Jun 26`.
- At least 2 rows are needed to show a sparkline. Suggest adding at least 6 months if possible.

---

### CONFIG
**Purpose:** Key-value pairs for the Tax screen and other computed values.

**Format:** This tab is a key-value list — **two columns, no fixed number of rows.**

**Headers (row 1):**
```
key | value
```

Prompt the user for each key in this order. If they don't know, use the default shown.

| key | What to ask | Default |
|-----|-------------|---------|
| `gross_income` | "What is your annual gross income (salary + other, before deductions)?" | `0` |
| `regime` | "Are you filing under New or Old income tax regime?" | `New` |
| `capital_gains_this_year` | "Any realised capital gains so far this financial year?" | `0` |
| `shifted_to_parents` | "Any income gifted or loaned to parents this FY for tax planning?" | `0` |
| `surcharge_from` | Leave as-is | `5000000` |
| `next_surcharge_at` | Leave as-is | `10000000` |
| `parents_capacity_mom` | "Estimated annual income capacity for Mom (before she hits 30% slab)?" | `500000` |
| `parents_capacity_dad` | "Estimated annual income capacity for Dad?" | `500000` |
| `deduction_80c` | "Total 80C deductions this year (PF + ELSS + LIC premiums + PPF)?" | `0` |
| `deduction_80d` | "Total 80D deductions (health insurance premiums for self + parents)?" | `0` |
| `other_deductions` | "Any other deductions (NPS 80CCD(1B), HRA, LTA, etc.)?" | `0` |
| `saved_by_filing` | "Known tax saving vs. filing everything under yourself? Leave 0 if unsure." | `0` |

**Example output:**
```
key	value
gross_income	6200000
regime	New
capital_gains_this_year	320000
shifted_to_parents	480000
surcharge_from	5000000
next_surcharge_at	10000000
parents_capacity_mom	700000
parents_capacity_dad	750000
deduction_80c	150000
deduction_80d	25000
other_deductions	50000
saved_by_filing	64350
```

---

## Output format rules

- Always output rows as **tab-separated values** so the user can paste directly into Google Sheets (Ctrl+V / Cmd+V).
- Include the header row in the first output for each tab.
- After the paste-ready block, add a short plain-English **"What was migrated"** summary for that tab.
- If any field could not be determined and was left as a default or placeholder, call it out explicitly so the user knows what to verify.

---

## Handling partial or missing data

- **Unknown current value:** Use invested/cost as a conservative placeholder. Note it.
- **Unknown dates:** Ask; do not invent. If the user genuinely can't recall, use `2024-01-01` as a visible placeholder they can fix later.
- **Unknown ticker:** Leave the cell with the company name and a `TODO:` prefix. Don't guess.
- **No history data:** Generate a single row for the current month using the sum of the migrated holdings as the value.
- **No insurance data:** Skip the tab and note it.

---

## Completion message

After all tabs are done, output:

```
Migration complete. Here's what was prepared:

✓ Members    — N rows
✓ MF         — N rows
✓ Stocks     — N rows
✓ Metals     — N rows
✓ Fixed      — N rows
✓ Insurance  — N rows
✓ SIPs       — N rows
✓ Lumpsums   — N rows
✓ History    — N rows
✓ Config     — 12 key-value rows

Next steps:
1. Open your Kosh Google Sheet.
2. For each tab above, click the first empty cell in column A (row 2)
   and paste the table. Do not paste over the header row.
3. Verify any fields marked TODO or noted as placeholders.
4. Update today_price (Metals) and last_price (Stocks) with current market prices.
5. Run `make dev` to start Kosh and confirm the Dashboard loads correctly.
```
