You are extracting data from an Indian gold or silver purchase invoice, receipt, or hallmark certificate PDF or image.

Extract the following fields and return ONLY a valid JSON object — no markdown fences, no explanation, no extra text.

{
  "type": "Gold",
  "grams": 11.5,
  "buy_rate": 9500,
  "date_purchased": "2025-03-29",
  "place": "Jeweller or store name",
  "notes": "Hallmark details, purity (e.g. 22K / 916), receipt number, making charges if noted separately, etc."
}

Rules:
- type           : must be exactly "Gold" or "Silver"
- grams          : net weight in grams, decimal number (e.g. 11.5)
- buy_rate       : price per gram in rupees paid by the buyer — calculate as total ÷ grams if only total is shown
                   include making charges in the rate only if they are bundled into the per-gram price
- date_purchased : purchase date, ISO format YYYY-MM-DD
- place          : jeweller or store name as printed on the invoice
- notes          : purity, hallmark BIS number, making charge rate if separate, any other useful detail
- If a required field cannot be found, set it to null

Tips:
- "Gross weight" includes stone/impurities; prefer "net weight" for grams
- Common purities: 24K = 999 fine, 22K = 916, 18K = 750
- Silver invoices may show weight in grams or kilograms — always convert to grams
- If the invoice covers both gold and silver items, extract the dominant metal or the first item listed
