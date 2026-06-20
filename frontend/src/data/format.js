// Formatting helpers — Indian numbering, ₹ INR, percentages, dates

function groupIndian(intStr) {
  if (intStr.length <= 3) return intStr
  const last3 = intStr.slice(-3)
  let other = intStr.slice(0, -3)
  other = other.replace(/\B(?=(\d{2})+(?!\d))/g, ',')
  return other + ',' + last3
}

export function fmtINR(n) {
  const neg = n < 0
  const v = Math.round(Math.abs(n))
  const s = '₹' + groupIndian(v.toString())
  return neg ? '−' + s : s
}

// ₹19.6L, ₹1.06Cr, ₹62K
export function fmtCompact(n) {
  const neg = n < 0
  const v = Math.abs(n)
  let out
  if (v >= 1e7) out = (v / 1e7).toFixed(2).replace(/\.?0+$/, '') + 'Cr'
  else if (v >= 1e5) out = (v / 1e5).toFixed(2).replace(/\.?0+$/, '') + 'L'
  else if (v >= 1e3) out = (v / 1e3).toFixed(1).replace(/\.?0+$/, '') + 'K'
  else out = Math.round(v).toString()
  return (neg ? '−₹' : '₹') + out
}

export function fmtPct(n, dp = 1) {
  return (n >= 0 ? '+' : '') + n.toFixed(dp) + '%'
}

export function fmtDate(d) {
  if (!d) return ''
  const dt = typeof d === 'string' ? new Date(d) : d
  const m = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']
  return dt.getDate() + ' ' + m[dt.getMonth()] + ' ' + dt.getFullYear()
}
