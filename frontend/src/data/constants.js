// Shared constants — asset class metadata, display orders, urgency thresholds

export const CLASS_META = {
  mf:        { label: 'Mutual Funds',       color: 'var(--accent)' },
  stocks:    { label: 'Stocks',             color: '#B5603E' },
  metals:    { label: 'Gold & Silver',      color: '#B0822C' },
  fixed:     { label: 'FD / RD',            color: '#9AA79E' },
  insurance: { label: 'Insurance & Plans',  color: '#5E7D72' },
  nps:       { label: 'NPS',               color: '#6B7A99' },
}

export const ED_ORDER = ['mf', 'stocks', 'fixed', 'nps', 'insurance', 'metals']
export const ED_COL = {
  mf: 'var(--accent)', stocks: '#B5603E', fixed: '#9AA79E', nps: '#6B7A99', insurance: '#5E7D72', metals: '#B0822C'
}
export const ED_LABEL = {
  mf: 'Mutual funds', stocks: 'Stocks', fixed: 'FD / RD', nps: 'NPS', insurance: 'Insurance', metals: 'Gold & silver'
}

// NPS asset class labels and colors (used in Investments.jsx NPSTable)
export const AC_LABEL = { E: 'Equity', C: 'Corp Bond', G: 'Govt Sec', A: 'Alternative' }
export const AC_COLOR = { E: 'var(--accent)', C: '#6B7A99', G: '#9AA79E', A: '#B0822C' }

// Outflow type display (used in Dashboard.jsx upcoming outflows widget)
export const OUTFLOW_COLOR = { sip: 'var(--accent)', insurance: '#5E7D72', loan: '#B5603E' }
export const OUTFLOW_LABEL = { sip: 'SIP debit', insurance: 'Insurance premium', loan: 'Loan EMI' }

// Urgency thresholds — days before due date that we highlight in warning colour
export const INSURANCE_URGENCY_DAYS = 14
export const FIXED_URGENCY_DAYS = 90
