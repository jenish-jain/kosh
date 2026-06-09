You are extracting NPS (National Pension System) holdings from a KFintech / NSDL CRA account statement PDF.

A statement may contain multiple scheme rows — one per asset class (E, C, G, A) per tier (T1/T2).
Extract ALL holdings found and return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

{
  "pran": "110012345678",
  "holdings": [
    {
      "tier": "T1",
      "asset_class": "E",
      "scheme": "NPS TRUST - A/C SBI PENSION FUND SCHEME E - TIER I",
      "fund_manager": "SBI Pension Funds",
      "units": 1234.5678,
      "nav": 52.3400,
      "invested": 58000
    }
  ]
}

Rules:
- pran         : PRAN number, digits only (no spaces or dashes); null if not found
- holdings     : array of ALL scheme rows found; include every asset class present
- tier         : "T1" for Tier I accounts, "T2" for Tier II accounts
- asset_class  : "E" (Equity), "C" (Corporate Bond), "G" (Govt Securities), or "A" (Alternative Assets)
- scheme       : full scheme name as printed (e.g. "NPS TRUST - A/C HDFC PENSION MANAGEMENT CO. LTD. - SCHEME E - TIER I")
- fund_manager : pension fund manager short name (e.g. "SBI", "HDFC", "UTI", "LIC", "Kotak", "Aditya Birla")
- units        : units held, as a decimal number
- nav          : latest NAV per unit, as a decimal number
- invested     : total contributions to this scheme in rupees (employee + employer + voluntary combined);
                 if shown as a table of yearly contributions, sum them; null if not determinable
- Set any field to null if it cannot be found in the document

Tips:
- Asset class codes appear as "Asset Class E", "Class C", "Scheme E", or embedded in the scheme name
- Tier I and Tier II appear as separate sections — assign each row the correct tier
- "Market Value" = units × NAV — use the raw units and NAV figures from the statement
- The statement password is typically PRAN + date of birth (DDMMYYYY) — the file will already be decrypted before you receive it
- Some older statements omit the PRAN; set it to null rather than guessing
