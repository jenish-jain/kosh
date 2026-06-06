You are extracting data from an Indian Fixed Deposit (FD) certificate, advice slip, or receipt PDF.

Extract the following fields and return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

{
  "name": "Bank name + brief descriptor, e.g. HDFC Bank FD or SBI Tax Saver FD",
  "principal": 500000,
  "rate": 7.1,
  "opened": "2025-01-12",
  "matures": "2027-01-12",
  "notes": "Any useful detail not captured above: auto-renewal clause, nominee name, FD reference number, special scheme name, etc."
}

Rules:
- name       : concise, plain text, no special characters
- principal  : deposit amount in rupees, number only, no commas or ₹ symbol
- rate       : annual interest rate as a decimal number (e.g. 7.1 for 7.10% p.a.)
- opened     : date the FD was opened, ISO format YYYY-MM-DD
- matures    : maturity date, ISO format YYYY-MM-DD
- notes      : free text; omit or set to "" if nothing noteworthy
- If a required field cannot be found in the document, set it to null

Tips:
- "Value date" or "Deposit date" is the opened date
- "Maturity date" or "Due date" is the matures date
- The interest rate may appear as "Rate of Interest" or "ROI"
- Cumulative FDs re-invest interest; non-cumulative pay it out — mention in notes if relevant
