You are extracting NPS (National Pension System) holdings from a KFintech / NSDL CRA account statement PDF.

Return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

## How KFintech statements are structured

1. **Account details** — PRAN, subscriber name, fund manager.
2. **Asset allocation (declared percentages)** — a table showing how contributions are split across asset classes, e.g. E: 75%, C: 10%, G: 15%, A: 0%.
3. **Scheme-wise holding details** — one row per asset class, each showing units held and current NAV.
4. **Contribution / Redemption details** — a transaction table whose final row shows the **net total invested** (total contributions minus any withdrawals/redemptions).

## How to compute `invested` per scheme

The statement does NOT show per-scheme invested amounts directly.
Compute it as:

  invested (for asset class X) = net_total_invested × (declared_pct_for_X / 100)

where `net_total_invested` is the closing total from the Contribution/Redemption Details table.

## Output format

{
  "pran": "110012345678",
  "net_total_invested": 96000,
  "holdings": [
    {
      "tier": "T1",
      "asset_class": "E",
      "scheme": "NPS TRUST - A/C SBI PENSION FUND SCHEME E - TIER I",
      "fund_manager": "SBI Pension Funds",
      "units": 1234.5678,
      "nav": 52.3400,
      "alloc_pct": 75,
      "invested": 72000
    }
  ]
}

## Field rules

- pran              : PRAN number, digits only; null if not found
- net_total_invested: closing net figure from Contribution/Redemption Details (contributions minus redemptions); null if not found
- holdings          : one entry per asset class row found in the scheme-wise holdings section
- tier              : "T1" for Tier I, "T2" for Tier II
- asset_class       : "E" (Equity), "C" (Corporate Bond), "G" (Govt Securities), "A" (Alternative)
- scheme            : full scheme name as printed
- fund_manager      : pension fund manager short name (SBI, HDFC, UTI, LIC, Kotak, Aditya Birla, etc.)
- units             : units held as a decimal number
- nav               : latest NAV per unit as a decimal number
- alloc_pct         : declared allocation percentage for this asset class (integer 0–100); null if not shown
- invested          : net_total_invested × (alloc_pct / 100); null if either value is missing
- Set any field to null if it cannot be determined

## Tips

- Asset allocation percentages appear in a section titled "Asset Allocation" or "Scheme Preference" — these are the declared percentages, not current market weights
- Contribution/Redemption Details is usually a multi-page table; the **last row** or a "Total" row at the bottom gives the net amount
- Tier I and Tier II are separate sections; process each independently (each has its own allocation table and contribution total)
- The statement password is PRAN + date of birth (DDMMYYYY) — the file will already be decrypted before you receive it
- Only include asset classes that have non-zero units in the holdings section; skip classes with zero units even if they appear in the allocation table
