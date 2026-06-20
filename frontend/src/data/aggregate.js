// Aggregation and lookup helpers — portfolio totals, member lookups

export function holdingsFor(data, memberId) {
  const f = arr => memberId ? arr.filter(x => x.member === memberId) : arr
  return {
    mf:        f(data.mf        || []),
    stocks:    f(data.stocks    || []),
    metals:    f(data.metals    || []),
    fixed:     f(data.fixed     || []),
    insurance: f(data.insurance || []),
    loans:     f(data.loans     || []),
    nps:       f(data.nps       || []),
  }
}

export function classTotals(data, memberId) {
  const h = holdingsFor(data, memberId)

  const mfCur  = h.mf.reduce((a, x) => a + (x.current  || 0), 0)
  const mfInv  = h.mf.reduce((a, x) => a + (x.invested || 0), 0)

  const stCur  = h.stocks.reduce((a, x) => a + (x.qty || 0) * (x.last_price || 0), 0)
  const stInv  = h.stocks.reduce((a, x) => a + (x.qty || 0) * (x.avg_price  || 0), 0)

  const meCur  = h.metals.reduce((a, x) => a + (x.grams || 0) * (x.today_price || 0), 0)
  const meInv  = h.metals.reduce((a, x) => a + (x.grams || 0) * (x.buy_rate    || 0), 0)

  const fiCur  = h.fixed.reduce((a, x) => a + (x.current_value || 0), 0)
  const fiInv  = h.fixed.reduce((a, x) => a + (x.principal    || 0), 0)

  const inCur  = h.insurance.reduce((a, x) => a + (x.value || 0), 0)
  const inInv  = h.insurance.reduce((a, x) => a + (x.paid  || 0), 0)

  const npsCur = h.nps.reduce((a, x) => a + (x.units || 0) * (x.nav || 0), 0)
  const npsInv = h.nps.reduce((a, x) => a + (x.invested || 0), 0)

  const loanOutstanding = h.loans.reduce((a, x) => a + (x.outstanding || 0), 0)
  const loanEmiMonthly  = h.loans.reduce((a, x) => a + (x.emi || 0), 0)

  return {
    mf:        { cur: mfCur, inv: mfInv },
    stocks:    { cur: stCur, inv: stInv },
    metals:    { cur: meCur, inv: meInv },
    fixed:     { cur: fiCur, inv: fiInv },
    insurance: { cur: inCur, inv: inInv },
    nps:       { cur: npsCur, inv: npsInv },
    liabilities: { cur: loanOutstanding, monthly: loanEmiMonthly },
    total:     {
      cur: mfCur + stCur + meCur + fiCur + inCur + npsCur,
      inv: mfInv + stInv + meInv + fiInv + inInv + npsInv,
    },
  }
}

// Gross asset total for a scope — used for allocation percentages, etc.
export function memberTotal(data, memberId) {
  return classTotals(data, memberId).total.cur
}

// True net worth — gross assets minus outstanding loan balances.
export function netWorth(data, memberId) {
  const c = classTotals(data, memberId)
  return c.total.cur - c.liabilities.cur
}

// Member display name — "Whole family" if no memberId, otherwise member's name
export function scope(data, memberId) {
  if (!memberId) return 'Whole family'
  const m = data.members?.find(m => m.id === memberId)
  return m ? (m.full_name || m.name).replace(' (You)', '') : '—'
}

// Find member by id
export function memberOf(data, id) {
  return (data.members || []).find(m => m.id === id)
}
