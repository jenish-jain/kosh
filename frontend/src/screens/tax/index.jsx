import { useState } from 'react'
import { fmtINR, fmtCompact } from '../../data/format.js'
import { todayDisplay } from '../../data/schedule.js'
import { KICK, SERIF } from '../../data/tokens.js'
import { EdRule } from '../../components/Primitives.jsx'
import {
  computeTax, slabLabel, effectiveRate, currentFY, activeRuleSet,
} from './taxMath.js'
import PotentialInflowsTile from './PotentialInflows.jsx'
import RecommendationsPanel from './RecommendationsPanel.jsx'
import AdvanceTaxSchedule from './AdvanceTaxSchedule.jsx'
import TaxRulesAdmin from './TaxRulesAdmin.jsx'

// ── Surcharge runway bar ──────────────────────────────────────
function SurchargeBar({ income }) {
  const lo = 5000000, hi = 10000000
  const pct = Math.min(Math.max((income - lo) / (hi - lo), 0), 1)
  const inRange = income > lo && income < hi
  const past = income >= hi
  const below = income <= lo

  return (
    <div style={{ marginTop: 6 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: 'var(--ink-3)', fontWeight: 700, letterSpacing: '.04em', marginBottom: 6 }}>
        <span>₹50L — 10% surcharge starts</span>
        <span>₹1Cr — 15% surcharge</span>
      </div>
      <div style={{ height: 10, background: 'var(--line)', borderRadius: 3, position: 'relative', overflow: 'hidden' }}>
        <div style={{
          position: 'absolute', left: 0, top: 0, bottom: 0,
          width: below ? '0%' : past ? '100%' : `${pct * 100}%`,
          background: inRange ? 'var(--warn)' : past ? 'var(--neg)' : 'var(--accent)',
          borderRadius: 3,
          transition: 'width .4s ease'
        }} />
      </div>
      <div style={{ fontSize: 12, marginTop: 7, color: 'var(--ink-3)', fontWeight: 600 }}>
        {below
          ? `${fmtCompact(lo - income)} below the 10% surcharge threshold — no surcharge`
          : past
          ? `Income above ₹1Cr — 15% surcharge applies`
          : `${fmtCompact(income - lo)} into the 10% surcharge zone`}
      </div>
    </div>
  )
}

// ── Regime switcher ───────────────────────────────────────────
function RegimeSwitcher({ value, onChange }) {
  return (
    <div className="seg" style={{ display: 'inline-flex' }}>
      {[['old', 'Old regime'], ['new', 'New regime']].map(([v, label]) => (
        <button key={v} className={value === v ? 'active' : ''} onClick={() => onChange(v)}>
          {label}
        </button>
      ))}
    </div>
  )
}

// ── Income split planner ──────────────────────────────────────
function SplitPlanner({ data, grossIncome, taxFn }) {
  const parents = (data.members || []).filter(m =>
    (m.relation === 'Father' || m.relation?.startsWith('Father') || m.relation === 'Mother') && m.id !== 'you'
  )
  const [shifted, setShifted] = useState(0)
  const parentCount = parents.length || 1
  const cfg = data.config || {}

  const myIncome = grossIncome - shifted
  const perParent = shifted / parentCount

  const baseMyTax = taxFn(grossIncome)
  const myTax = taxFn(myIncome)
  const parentTax = parents.length > 0 ? parents.reduce((a, m) => a + taxFn(perParent), 0) : taxFn(perParent)
  const saved = baseMyTax - myTax - parentTax

  const sliderMax = Math.min(grossIncome * 0.5, 4000000)

  const capacities = {
    mom: cfg.parents_capacity_mom || 500000,
    dad: cfg.parents_capacity_dad || 500000,
  }

  return (
    <div>
      <div style={{ ...KICK, marginBottom: 14 }}>Income-split planner</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', gap: 32, alignItems: 'center' }}>
        <div>
          <div style={KICK}>Shift to family</div>
          <div className="num serif-num" style={{ fontSize: 38, marginTop: 8 }}>{fmtINR(shifted)}</div>
          <input type="range" min="0" max={Math.round(sliderMax)} step="10000" value={shifted}
            onChange={e => setShifted(Number(e.target.value))}
            style={{ width: '100%', marginTop: 12, accentColor: 'var(--accent)' }} />
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: 'var(--ink-3)', fontWeight: 700 }}>
            <span>₹0</span><span>{fmtCompact(sliderMax)}</span>
          </div>
        </div>
        <div style={{ width: 1, background: 'var(--line)', alignSelf: 'stretch' }} />
        <div>
          <div style={KICK}>Tax saved vs. no split</div>
          <div className="num serif-num" style={{ fontSize: 38, marginTop: 8, color: saved > 0 ? 'var(--pos)' : 'var(--ink-3)' }}>
            {saved > 0 ? '+' : ''}{fmtINR(Math.max(0, saved))}
          </div>
          <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 6 }}>
            Your effective tax: {fmtINR(myTax)}
            {parents.length > 0 && ` · parents pay ${fmtINR(parentTax)}`}
          </div>
        </div>
      </div>

      {parents.length > 0 && shifted > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: `repeat(${parents.length},1fr)`, gap: 18, marginTop: 24 }}>
          {parents.map(m => {
            const cap = m.relation?.startsWith('Mother') ? capacities.mom : capacities.dad
            const used = perParent
            const pct = Math.min(used / cap * 100, 100)
            return (
              <div key={m.id}>
                <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 4 }}>{m.full_name || m.name}</div>
                <div style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 600, marginBottom: 10 }}>
                  Receives {fmtCompact(perParent)} · slab {m.slab}%
                </div>
                <div style={{ height: 7, background: 'var(--line)', borderRadius: 2, overflow: 'hidden' }}>
                  <div style={{ height: '100%', width: `${pct}%`, background: pct > 90 ? 'var(--warn)' : 'var(--accent)', borderRadius: 2, transition: 'width .3s' }} />
                </div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 5 }}>
                  {fmtCompact(used)} of {fmtCompact(cap)} estimated capacity
                </div>
              </div>
            )
          })}
        </div>
      )}

      {parents.length === 0 && (
        <div className="empty-hint" style={{ marginTop: 12 }}>Add family members with a lower tax slab (Father / Mother) to use this planner.</div>
      )}
    </div>
  )
}

// ── Main screen ──────────────────────────────────────────────
export default function Tax({ data, memberId }) {
  const cfg = data.config || {}
  const selfMember = (data.members || []).find(m => m.id === 'you') || {}
  const fy = currentFY()

  // Default regime from Config sheet (regime=new/old); falls back to old
  const [regime, setRegime] = useState(() => (cfg.regime || '').toLowerCase() === 'new' ? 'new' : 'old')
  const isNew = regime === 'new'
  const oldRules = activeRuleSet(data.tax_rules, fy.label, 'old')
  const newRules = activeRuleSet(data.tax_rules, fy.label, 'new')
  const rules = isNew ? newRules : oldRules
  const taxFn = income => computeTax(rules, income)

  // Derive annual gross from Income tab (latest period × 12), fall back to Config
  const MONTHS = {Jan:0,Feb:1,Mar:2,Apr:3,May:4,Jun:5,Jul:6,Aug:7,Sep:8,Oct:9,Nov:10,Dec:11}
  const parsePeriod = p => { const [m, y] = (p || '').split(' '); return new Date(+y, MONTHS[m] ?? 0) }
  const incomeRows = data.income || []
  let grossIncome = cfg.gross_income || 0
  let latestPeriod = null
  if (incomeRows.length > 0) {
    let latestDate = null
    incomeRows.forEach(r => {
      const t = parsePeriod(r.period)
      if (!latestDate || t > latestDate) { latestDate = t; latestPeriod = r.period }
    })
    if (latestPeriod) {
      const monthlyGross = incomeRows
        .filter(r => r.period === latestPeriod)
        .reduce((sum, r) => sum + (r.gross || 0), 0)
      if (monthlyGross > 0) grossIncome = monthlyGross * 12
    }
  }

  const taxPayable = taxFn(grossIncome)
  const slab = slabLabel(rules, grossIncome)
  const effRate = effectiveRate(grossIncome, taxFn)
  const savedByFiling = cfg.saved_by_filing || 0

  // Old regime deductions from Config
  const deduction80c = cfg.deduction_80c || 0
  const deduction80d = cfg.deduction_80d || 0
  const otherDeductions = cfg.other_deductions || 0
  const totalOldDeductions = deduction80c + deduction80d + otherDeductions
  const taxableIncomeOld = Math.max(0, grossIncome - totalOldDeductions)
  const taxAfterOldDeductions = computeTax(oldRules, taxableIncomeOld)

  // New regime: standard deduction only
  const newStdDeduction = newRules.stdDeduction || 0
  const taxableIncomeNew = Math.max(0, grossIncome - newStdDeduction)
  const rebateEligible = taxableIncomeNew <= (newRules.rebateThreshold || 0)

  const [showTaxRulesAdmin, setShowTaxRulesAdmin] = useState(false)

  const memberName = memberId
    ? ((data.members || []).find(m => m.id === memberId)?.full_name || '').replace(' (You)', '')
    : (selfMember.full_name || 'You').replace(' (You)', '')

  return (
    <div className="fade-in">
      <div className="stmt-band" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
        <div>
          <div style={{ ...KICK, letterSpacing: '.18em' }}>Tax position &amp; planning</div>
          <div className="stmt-meta">{memberName} · {fy.label} · {todayDisplay()}</div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <button className="btn ghost sm" onClick={() => setShowTaxRulesAdmin(true)}>Tax rules</button>
          <RegimeSwitcher value={regime} onChange={setRegime} />
        </div>
      </div>
      <EdRule thick />

      {/* 4 headline figures */}
      <div className="tax-tiles">
        <div className="tax-tile">
          <div style={KICK}>Gross income</div>
          <div className="num serif-num" style={{ fontSize: 36, marginTop: 10 }}>{fmtINR(grossIncome)}</div>
          <div style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
            {latestPeriod ? `${latestPeriod} × 12 (from Income tab)` : 'before deductions'}
          </div>
        </div>
        <div className="tax-tile">
          <div style={KICK}>Current slab</div>
          <div className="num serif-num" style={{ fontSize: 36, marginTop: 10 }}>{slab}</div>
          <div style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>{effRate}% effective rate</div>
        </div>
        <PotentialInflowsTile data={data} />
        <div className="tax-tile">
          <div style={KICK}>Saved by splitting</div>
          <div className="num serif-num" style={{ fontSize: 36, marginTop: 10, color: savedByFiling > 0 ? 'var(--pos)' : 'var(--ink)' }}>
            {savedByFiling > 0 ? '+' + fmtINR(savedByFiling) : '—'}
          </div>
          <div style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>vs. no family routing</div>
        </div>
      </div>

      <EdRule />

      {/* Surcharge runway */}
      <div style={{ ...KICK, marginBottom: 12 }}>Surcharge runway</div>
      <SurchargeBar income={grossIncome} />

      <EdRule />

      {/* Two columns: tax payable + deductions */}
      <div className="tax-cols">
        <div>
          <div style={KICK}>Tax payable ({isNew ? 'new' : 'old'} regime)</div>
          <div className="num serif-num" style={{ fontSize: 48, marginTop: 10 }}>{fmtINR(taxPayable)}</div>

          {isNew ? (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginTop: 20 }}>
              {[
                { label: 'Effective rate', value: `${effRate}%` },
                { label: 'Std deduction', value: fmtCompact(newStdDeduction) },
                { label: '87A rebate', value: rebateEligible ? 'Eligible' : 'Not eligible', color: rebateEligible ? 'var(--pos)' : 'var(--warn)' },
                { label: 'Net take-home', value: fmtCompact(grossIncome - taxPayable) },
              ].map((r, i) => (
                <div key={i}>
                  <div style={KICK}>{r.label}</div>
                  <div className="num serif-num" style={{ fontSize: 20, marginTop: 6, color: r.color || 'var(--ink)' }}>{r.value}</div>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginTop: 20 }}>
              {[
                { label: 'Effective rate', value: `${effRate}%` },
                { label: 'After deductions', value: fmtCompact(taxAfterOldDeductions) },
                { label: '4% cess', value: fmtCompact(Math.round(taxPayable * 0.04 / 1.04)) },
                { label: 'Net take-home', value: fmtCompact(grossIncome - taxAfterOldDeductions) },
              ].map((r, i) => (
                <div key={i}>
                  <div style={KICK}>{r.label}</div>
                  <div className="num serif-num" style={{ fontSize: 20, marginTop: 6 }}>{r.value}</div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ background: 'var(--line)', alignSelf: 'stretch' }} />

        <div>
          {isNew ? (
            <>
              <div style={KICK}>Deductions (new regime)</div>
              <div style={{ marginTop: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <span style={{ fontSize: 13, fontWeight: 700 }}>Standard deduction</span>
                  <span className="num serif-num" style={{ fontSize: 14, color: 'var(--pos)' }}>{fmtINR(newStdDeduction)}</span>
                </div>
                <div style={{ height: 6, background: 'var(--line)', borderRadius: 2, overflow: 'hidden', marginBottom: 6 }}>
                  <div style={{ height: '100%', width: '100%', background: 'var(--pos)', borderRadius: 2 }} />
                </div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginBottom: 20 }}>
                  Auto-applied · no documentation needed
                </div>

                <div style={{ background: 'var(--surface)', border: '1px solid var(--line)', borderRadius: 6, padding: '12px 14px', fontSize: 12.5, color: 'var(--ink-3)', lineHeight: 1.6 }}>
                  <strong style={{ color: 'var(--ink-2)' }}>Not available under new regime:</strong><br />
                  80C (ELSS, PF, LIC) · 80D (health insurance) · HRA · LTA · NPS · home loan interest
                </div>

                {totalOldDeductions > 0 && taxAfterOldDeductions < taxPayable && (
                  <div style={{ marginTop: 14, padding: '10px 14px', borderRadius: 6, background: 'var(--accent-soft)', fontSize: 12.5, lineHeight: 1.6 }}>
                    <span style={{ fontWeight: 700 }}>Old regime saves {fmtCompact(taxPayable - taxAfterOldDeductions)} more</span>
                    {' '}with your {fmtCompact(totalOldDeductions)} in deductions —{' '}
                    <button onClick={() => setRegime('old')} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--accent)', fontWeight: 700, fontSize: 12.5, padding: 0 }}>
                      switch to compare
                    </button>
                  </div>
                )}
              </div>
            </>
          ) : (
            <>
              <div style={KICK}>Deduction breakdown</div>
              <div style={{ marginTop: 12 }}>
                {[
                  { label: '80C (ELSS · PF · LIC)', cap: oldRules.deductionCaps?.section80C || 150000, used: deduction80c },
                  { label: '80D (health insurance)', cap: oldRules.deductionCaps?.section80DSelf || 25000, used: deduction80d },
                  { label: 'Other (NPS · LTA etc.)', cap: 150000, used: otherDeductions },
                ].map(d => {
                  const pct = d.cap > 0 ? Math.min(d.used / d.cap * 100, 100) : 0
                  return (
                    <div key={d.label} style={{ marginBottom: 18 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 5 }}>
                        <span style={{ fontSize: 12.5, fontWeight: 600 }}>{d.label}</span>
                        <span className="num serif-num" style={{ fontSize: 13 }}>{fmtINR(d.used)} / {fmtCompact(d.cap)}</span>
                      </div>
                      <div style={{ height: 6, background: 'var(--line)', borderRadius: 2, overflow: 'hidden' }}>
                        <div style={{ height: '100%', width: `${pct}%`, background: pct >= 100 ? 'var(--pos)' : 'var(--accent)', borderRadius: 2 }} />
                      </div>
                      <div style={{ fontSize: 11, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
                        {pct >= 100 ? 'Maxed out' : `${fmtCompact(d.cap - d.used)} room remaining`}
                      </div>
                    </div>
                  )
                })}
                <div style={{ borderTop: '1px solid var(--line)', paddingTop: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                  <div style={KICK}>Total deductions</div>
                  <div className="num serif-num" style={{ fontSize: 22 }}>{fmtINR(totalOldDeductions)}</div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      <EdRule />

      {/* Advance tax instalment schedule */}
      <AdvanceTaxSchedule taxPayable={isNew ? taxPayable : taxAfterOldDeductions} fy={fy} />

      <EdRule />

      {/* Tax-saving recommendations */}
      <RecommendationsPanel data={data} regime={regime} />

      <EdRule />

      {/* Income split planner */}
      <SplitPlanner data={data} grossIncome={grossIncome} taxFn={taxFn} />

      <EdRule />

      <div style={{ fontFamily: SERIF, fontSize: 13.5, fontStyle: 'italic', color: 'var(--ink-3)', lineHeight: 1.7 }}>
        Indicative figures under {isNew ? 'new' : 'old'} regime · no marginal relief applied · consult a qualified CA before acting on any strategy.
        {isNew && ` New regime: standard deduction ${fmtCompact(newStdDeduction)} auto-applied; 87A rebate if taxable income ≤ ${fmtCompact(newRules.rebateThreshold || 0)}.`}
      </div>

      {showTaxRulesAdmin && (
        <TaxRulesAdmin data={data} onClose={() => setShowTaxRulesAdmin(false)} />
      )}
    </div>
  )
}
