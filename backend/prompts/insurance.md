You are extracting data from an Indian insurance policy document, schedule, or premium receipt PDF.

Extract the following fields and return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

{
  "name": "Policy name as printed, e.g. LIC Jeevan Anand or HDFC Click 2 Protect",
  "type": "Term",
  "premium": 48000,
  "freq": "annual",
  "cover": 10000000,
  "maturity": 2040,
  "due_date": "2026-06-18",
  "notes": "Riders attached, nominee name, policy number, insurer, any exclusion worth noting, etc."
}

Rules:
- name     : as printed on the policy; include insurer name if not obvious from policy name
- type     : must be exactly one of — Term, Endowment, ULIP, Income
             Term      = pure life cover, no maturity benefit
             Endowment = life cover + guaranteed maturity corpus
             ULIP      = market-linked investment + life cover
             Income    = regular payout / money-back plans
- premium  : premium amount in rupees per payment, number only
- freq     : payment frequency — exactly one of: annual, monthly, single
- cover    : sum assured / life cover in rupees, number only
- maturity : year the policy matures or cover ends, 4-digit integer (e.g. 2040)
- due_date : the next/anchor premium due date in YYYY-MM-DD — for annual or
             monthly plans this just needs to be ANY date with the right
             month+day (annual) or day-of-month (monthly) that the premium
             recurs on; if unclear, use the policy's commencement/issue date
- notes    : free text; set to "" if nothing noteworthy
- If a required field cannot be found, set it to null

Tips:
- "Sum Assured" or "Basic Sum Assured" is the cover
- "Premium Paying Term" gives frequency context; "Policy Term" gives maturity year
- Money-back and pension plans should be typed as "Income"
- ULIP plans often say "Unit Linked" explicitly
- "Date of Commencement", "Risk Commencement Date", or "Due Date" on a
  premium receipt are good sources for due_date
