You are extracting Indian income-tax rules for ONE tax regime (old or new) from an official document — a Union Budget memorandum, Finance Bill excerpt, or CBDT circular.

Extract the following fields and return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

{
  "regime": "new",
  "fy": "FY 2026-27",
  "slabs": [
    {"upto": 400000, "rate": 0},
    {"upto": 800000, "rate": 0.05},
    {"rate": 0.30}
  ],
  "stdDeduction": 75000,
  "rebateThreshold": 1200000,
  "rebateAmount": 60000,
  "surcharge": [
    {"above": 5000000, "rate": 0.10},
    {"above": 10000000, "rate": 0.15}
  ],
  "cessRate": 0.04,
  "deductionCaps": {
    "section80C": 150000,
    "section80DSelf": 25000,
    "section80DSenior": 50000,
    "nps80CCD1B": 50000
  },
  "source": "Union Budget 2026 Memorandum, Part A",
  "notes": "Any caveat worth a human reviewer's attention — e.g. marginal relief provisions this schema doesn't model, or a figure you're not fully confident about."
}

Rules:
- regime          : "old" or "new" — whichever regime this document's figures describe. If the document covers both, extract the one discussed first/most prominently and say which in "notes".
- fy              : the financial year these rules take effect for, as "FY YYYY-YY" (e.g. "FY 2026-27")
- slabs           : income-tax slabs applied to TAXABLE income (after standard deduction, if any), in ascending order. Each band's "upto" is its upper bound in rupees; the LAST band must omit "upto" entirely (unbounded top rate) rather than using 0, null, or a huge number. "rate" is a decimal fraction (5% = 0.05), never a percentage integer.
- stdDeduction    : the standard deduction in rupees, 0 if this regime has none
- rebateThreshold : the taxable-income ceiling (rupees) under which Section 87A fully zeroes out tax, 0 if this regime's rules don't specify one you can find
- rebateAmount    : the maximum rupee value of the 87A rebate. If the law fully zeroes tax up to the threshold, use a very large number (e.g. 999999999) rather than a specific rebate cap that might understate it
- surcharge       : rupee thresholds (gross income) above which a surcharge applies, ascending by "above". Omit entirely if none found rather than guessing.
- cessRate        : the health & education cess as a decimal fraction (4% = 0.04)
- deductionCaps   : only applicable to the OLD regime typically — extract whatever old-regime deduction limits (80C, 80D self/senior, NPS 80CCD(1B)) this document mentions. If the document is about the NEW regime and doesn't discuss these, leave them at their prior known values or 0 if genuinely unknown — do not fabricate a number.
- source          : a short citation to where in the document these figures came from (section/para name if visible)
- notes           : free text; omit or set to "" if nothing noteworthy

If you cannot confidently determine a field from the document, do not guess a plausible-sounding number — set it to 0 (or an empty array for slabs/surcharge) and explain why in "notes". A human reviews every extraction before it takes effect, so a flagged gap is far better than a fabricated figure.
