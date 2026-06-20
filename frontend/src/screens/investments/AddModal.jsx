import { useState } from 'react'
import { useData } from '../../data/context.jsx'
import { Modal, Field } from '../../components/Primitives.jsx'
import SegmentedButtons from '../../components/shared/SegmentedButtons.jsx'
import FormGrid from '../../components/shared/FormGrid.jsx'
import { fmtINR } from '../../data/format.js'
import { todayStr } from '../../data/schedule.js'
import { computeMatures } from './countdown.js'

export default function AddModal({ tab, memberId, data, onClose, initialForm }) {
  const { add } = useData()
  const defaults = {
    member: memberId || (data.members?.[0]?.id || 'you'),
    freq: 'annual',
    metalType: 'Gold',
    insType: 'Endowment',
    fdKind: 'FD',
    npsTier: 'T1',
    npsAC: 'E',
  }
  const [form, setForm] = useState(initialForm ? { ...defaults, ...initialForm } : defaults)
  const fromUpload = !!initialForm
  const up = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const n = v => Number(v) || 0

  const submit = async () => {
    const id = tab + Date.now()
    if (tab === 'mf') {
      await add('MF', { id, name: form.name || 'New Fund', plan: 'Direct · Growth', platform: form.platform || '—', member: form.member, invested: n(form.invested), current: n(form.current) || n(form.invested), sip: n(form.sip), notes: form.notes || '' })
    } else if (tab === 'stocks') {
      await add('Stocks', { id, name: form.name || 'New Stock', ticker: (form.ticker || form.name || 'NEW').toUpperCase().slice(0, 8), qty: n(form.qty), avg_price: n(form.avg), last_price: n(form.ltp) || n(form.avg), member: form.member })
    } else if (tab === 'metals') {
      await add('Metals', { id, type: form.metalType, date_purchased: new Date().toISOString().slice(0, 10), grams: n(form.grams), buy_rate: n(form.buyRate), today_price: n(form.buyRate), place: form.place || '—', member: form.member })
    } else if (tab === 'insurance') {
      await add('Insurance', { id, name: form.name || 'New plan', type: form.insType, member: form.member, premium: n(form.premium), freq: form.freq, paid: n(form.paid), value: n(form.value), cover: n(form.cover), maturity: n(form.maturity) || 2040, due_date: form.dueDate || '', doc_link: form.docLink || '' })
    } else if (tab === 'fixed') {
      const tenure  = n(form.tenure)
      const monthly = form.fdKind === 'RD' ? n(form.monthly) : 0
      const principal = form.fdKind === 'FD' ? n(form.principal) : monthly * tenure
      const matures = computeMatures(form.opened || todayStr(), tenure)
      await add('Fixed', { id, kind: form.fdKind, name: form.name || 'New deposit', member: form.member, principal, rate: n(form.rate), current_value: principal, opened: form.opened || todayStr(), matures, monthly, doc_link: form.docLink || '' })
    } else if (tab === 'nps') {
      await add('NPS', { id, pran: form.pran || '', member: form.member, tier: form.npsTier || 'T1', asset_class: form.npsAC || 'E', scheme: form.scheme || '', fund_manager: form.fundManager || '', units: n(form.units), nav: n(form.nav), invested: n(form.invested) })
    }
    onClose()
  }

  const memberPicker = (
    <Field label="Owner">
      <SegmentedButtons
        options={(data.members || []).map(m => ({ value: m.id, label: m.name }))}
        value={form.member}
        onChange={v => up('member', v)}
      />
    </Field>
  )

  const titles = {
    mf: 'Add mutual fund',
    stocks: 'Add stock',
    metals: 'Add metal',
    insurance: 'Add insurance / plan',
    fixed: 'Add deposit (FD / RD)',
    nps: 'Add NPS holding',
  }

  return (
    <Modal title={titles[tab]} onClose={onClose}>
      {fromUpload && (
        <div style={{
          margin: '0 0 16px', padding: '9px 14px', borderRadius: 4,
          background: 'var(--accent-soft)', color: 'var(--accent-ink)',
          fontSize: 12.5, fontWeight: 600,
        }}>
          ✦ Pre-filled by Claude from your document — review each field before saving.
        </div>
      )}

      {tab === 'mf' && <>
        <Field label="Fund name">
          <input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. Quant Small Cap Fund" />
        </Field>
        <FormGrid>
          <Field label="Invested so far"><input className="input" type="number" value={form.invested || ''} onChange={e => up('invested', e.target.value)} /></Field>
          <Field label="Current value"><input className="input" type="number" value={form.current || ''} onChange={e => up('current', e.target.value)} /></Field>
        </FormGrid>
        <FormGrid>
          <Field label="Monthly SIP (0 if lumpsum)"><input className="input" type="number" value={form.sip || ''} onChange={e => up('sip', e.target.value)} /></Field>
          <Field label="Platform"><input className="input" value={form.platform || ''} onChange={e => up('platform', e.target.value)} placeholder="e.g. Groww" /></Field>
        </FormGrid>
      </>}

      {tab === 'stocks' && <>
        <Field label="Company">
          <input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. Reliance Industries" />
        </Field>
        <FormGrid>
          <Field label="Ticker"><input className="input" value={form.ticker || ''} onChange={e => up('ticker', e.target.value)} placeholder="e.g. RELIANCE" /></Field>
          <Field label="Qty"><input className="input" type="number" value={form.qty || ''} onChange={e => up('qty', e.target.value)} /></Field>
        </FormGrid>
        <FormGrid>
          <Field label="Avg price"><input className="input" type="number" value={form.avg || ''} onChange={e => up('avg', e.target.value)} /></Field>
          <Field label="Last price"><input className="input" type="number" value={form.ltp || ''} onChange={e => up('ltp', e.target.value)} /></Field>
        </FormGrid>
      </>}

      {tab === 'metals' && <>
        <Field label="Metal">
          <SegmentedButtons
            options={[{ value: 'Gold', label: 'Gold' }, { value: 'Silver', label: 'Silver' }]}
            value={form.metalType}
            onChange={v => up('metalType', v)}
          />
        </Field>
        <FormGrid>
          <Field label="Grams"><input className="input" type="number" value={form.grams || ''} onChange={e => up('grams', e.target.value)} /></Field>
          <Field label="Buy rate / g"><input className="input" type="number" value={form.buyRate || ''} onChange={e => up('buyRate', e.target.value)} /></Field>
        </FormGrid>
        <Field label="Place of purchase">
          <input className="input" value={form.place || ''} onChange={e => up('place', e.target.value)} placeholder="e.g. Kalamandir" />
        </Field>
      </>}

      {tab === 'insurance' && <>
        <Field label="Plan name">
          <input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. LIC Jeevan Anand" />
        </Field>
        <Field label="Type">
          <SegmentedButtons
            options={[
              { value: 'Endowment', label: 'Endowment' },
              { value: 'Term',      label: 'Term' },
              { value: 'ULIP',      label: 'ULIP' },
              { value: 'Income',    label: 'Income' },
            ]}
            value={form.insType}
            onChange={v => up('insType', v)}
          />
        </Field>
        <FormGrid>
          <Field label="Premium"><input className="input" type="number" value={form.premium || ''} onChange={e => up('premium', e.target.value)} /></Field>
          <Field label="Frequency">
            <select className="input" value={form.freq} onChange={e => up('freq', e.target.value)}>
              <option value="annual">Annual</option>
              <option value="monthly">Monthly</option>
              <option value="single">Single</option>
            </select>
          </Field>
        </FormGrid>
        <Field label={form.freq === 'monthly' ? 'Next premium due (sets the day-of-month it recurs on)' : form.freq === 'single' ? 'Premium due date' : 'Next premium due (sets the month + day it recurs on)'}>
          <input className="input" type="date" value={form.dueDate || ''} onChange={e => up('dueDate', e.target.value)} />
        </Field>
        <FormGrid>
          <Field label="Paid so far"><input className="input" type="number" value={form.paid || ''} onChange={e => up('paid', e.target.value)} /></Field>
          <Field label="Sum assured / cover"><input className="input" type="number" value={form.cover || ''} onChange={e => up('cover', e.target.value)} /></Field>
        </FormGrid>
        <FormGrid>
          <Field label="Current / fund value"><input className="input" type="number" value={form.value || ''} onChange={e => up('value', e.target.value)} placeholder="optional" /></Field>
          <Field label="Maturity year"><input className="input" type="number" value={form.maturity || ''} onChange={e => up('maturity', e.target.value)} placeholder="e.g. 2040" /></Field>
        </FormGrid>
      </>}

      {tab === 'fixed' && <>
        <Field label="Kind">
          <SegmentedButtons
            options={[{ value: 'FD', label: 'Fixed Deposit' }, { value: 'RD', label: 'Recurring Deposit' }]}
            value={form.fdKind}
            onChange={v => up('fdKind', v)}
          />
        </Field>
        <Field label="Bank / Institution">
          <input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. SBI, HDFC Bank" />
        </Field>
        <FormGrid>
          <Field label="Interest rate (% p.a.)"><input className="input" type="number" step="0.01" value={form.rate || ''} onChange={e => up('rate', e.target.value)} placeholder="e.g. 7.1" /></Field>
          <Field label="Opened date"><input className="input" type="date" value={form.opened || todayStr()} onChange={e => up('opened', e.target.value)} /></Field>
        </FormGrid>
        {form.fdKind === 'FD' ? (
          <FormGrid>
            <Field label="Principal amount"><input className="input" type="number" value={form.principal || ''} onChange={e => up('principal', e.target.value)} /></Field>
            <Field label="Tenure (months)"><input className="input" type="number" value={form.tenure || ''} onChange={e => up('tenure', e.target.value)} placeholder="e.g. 24" /></Field>
          </FormGrid>
        ) : (
          <FormGrid>
            <Field label="Monthly installment"><input className="input" type="number" value={form.monthly || ''} onChange={e => up('monthly', e.target.value)} /></Field>
            <Field label="Tenure (months)"><input className="input" type="number" value={form.tenure || ''} onChange={e => up('tenure', e.target.value)} placeholder="e.g. 12" /></Field>
          </FormGrid>
        )}
        {(form.tenure > 0) && (
          <div style={{ fontSize: 12, color: 'var(--ink-3)', padding: '8px 12px', background: 'var(--bg-2,#f7f5f0)', borderRadius: 4, lineHeight: 1.6 }}>
            {form.fdKind === 'RD' && form.monthly > 0 && <>Total committed: <strong>{fmtINR(n(form.monthly) * n(form.tenure))}</strong> · </>}
            Matures: <strong>{computeMatures(form.opened || todayStr(), n(form.tenure))}</strong>
          </div>
        )}
      </>}

      {tab === 'nps' && <>
        <FormGrid>
          <Field label="Tier">
            <SegmentedButtons
              options={[{ value: 'T1', label: 'Tier I' }, { value: 'T2', label: 'Tier II' }]}
              value={form.npsTier}
              onChange={v => up('npsTier', v)}
            />
          </Field>
          <Field label="Asset class">
            <SegmentedButtons
              options={[
                { value: 'E', label: 'E' },
                { value: 'C', label: 'C' },
                { value: 'G', label: 'G' },
                { value: 'A', label: 'A' },
              ]}
              value={form.npsAC}
              onChange={v => up('npsAC', v)}
            />
          </Field>
        </FormGrid>
        <Field label="Scheme name">
          <input className="input" value={form.scheme || ''} onChange={e => up('scheme', e.target.value)} placeholder="e.g. NPS TRUST - A/C SBI Pension Fund Scheme E" />
        </Field>
        <Field label="Fund manager">
          <input className="input" value={form.fundManager || ''} onChange={e => up('fundManager', e.target.value)} placeholder="e.g. SBI, HDFC, UTI, LIC, Kotak" />
        </Field>
        <FormGrid cols={3}>
          <Field label="Units"><input className="input" type="number" step="0.0001" value={form.units || ''} onChange={e => up('units', e.target.value)} /></Field>
          <Field label="NAV (₹)"><input className="input" type="number" step="0.01" value={form.nav || ''} onChange={e => up('nav', e.target.value)} /></Field>
          <Field label="Invested (₹)"><input className="input" type="number" value={form.invested || ''} onChange={e => up('invested', e.target.value)} /></Field>
        </FormGrid>
        {n(form.units) > 0 && n(form.nav) > 0 && (
          <div style={{ fontSize: 12, color: 'var(--ink-3)', padding: '8px 12px', background: 'var(--bg-2,#f7f5f0)', borderRadius: 4 }}>
            Current value: <strong>{fmtINR(n(form.units) * n(form.nav))}</strong>
            {n(form.invested) > 0 && (
              <> · Gain: <strong style={{ color: n(form.units) * n(form.nav) >= n(form.invested) ? 'var(--pos)' : 'var(--neg)' }}>
                {n(form.units) * n(form.nav) >= n(form.invested) ? '+' : '−'}{fmtINR(Math.abs(n(form.units) * n(form.nav) - n(form.invested)))}
              </strong></>
            )}
          </div>
        )}
        <Field label="PRAN (optional)">
          <input className="input" value={form.pran || ''} onChange={e => up('pran', e.target.value)} placeholder="e.g. 110012345678" />
        </Field>
      </>}

      {memberPicker}
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit}>Add holding</button>
      </div>
    </Modal>
  )
}
