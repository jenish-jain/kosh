import { todayStr } from '../../data/schedule.js'

// extractedToForm — maps AI-extracted fields to form shape for each investment tab
export function extractedToForm(tab, fields) {
  if (!fields) return {}
  const num = k => (fields[k] != null && fields[k] !== '' ? Number(fields[k]) : '')
  if (tab === 'fixed') {
    const opened  = fields.opened  || todayStr()
    const matures = fields.matures || ''
    const tenure  = opened && matures
      ? Math.round((new Date(matures) - new Date(opened)) / (365.25 / 12 * 864e5))
      : ''
    return { fdKind: 'FD', name: fields.name || '', rate: num('rate'), opened, principal: num('principal'), tenure }
  }
  if (tab === 'insurance') {
    return {
      insType:  fields.type     || 'Endowment',
      name:     fields.name     || '',
      premium:  num('premium'),
      freq:     fields.freq     || 'annual',
      cover:    num('cover'),
      maturity: num('maturity'),
      dueDate:  fields.due_date || '',
    }
  }
  if (tab === 'metals') {
    return {
      metalType: fields.type === 'Silver' ? 'Silver' : 'Gold',
      grams:     num('grams'),
      buyRate:   num('buy_rate'),
      place:     fields.place  || '',
    }
  }
  return {}
}
