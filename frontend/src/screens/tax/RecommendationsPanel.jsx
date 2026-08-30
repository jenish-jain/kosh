import { useState } from 'react'
import { fmtINR, fmtCompact } from '../../data/format.js'
import { KICK } from '../../data/tokens.js'
import { Icon } from '../../components/Icons.jsx'
import { useData } from '../../data/context.jsx'
import { generateTaxRecommendations } from '../../data/api.js'

const STATUS_PILL = { new: 'pill accent', actioned: 'pill pos', dismissed: 'pill neutral', superseded: 'pill neutral' }
const STATUS_LABEL = { new: 'New', actioned: 'Actioned', dismissed: 'Dismissed', superseded: 'Superseded' }

function RecommendationCard({ rec, onUpdate, busy }) {
  // A row keeps whatever status it had (new/actioned/dismissed) when a later
  // generation cycle superseded it — superseded_by is set independently of
  // status. Surface that here rather than showing a stale "New" pill on
  // advice that's no longer current.
  const superseded = !!rec.superseded_by
  const displayStatus = superseded ? 'superseded' : rec.status

  return (
    <div style={{ padding: '16px 0', borderBottom: '1px solid var(--line)', opacity: superseded ? 0.6 : 1 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
        <div style={{ flex: 1, minWidth: 220 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6, flexWrap: 'wrap' }}>
            <span style={KICK}>{rec.category}</span>
            <span className={STATUS_PILL[displayStatus] || 'pill neutral'}>{STATUS_LABEL[displayStatus] || displayStatus}</span>
          </div>
          <div style={{ fontSize: 15, fontWeight: 700 }}>{rec.headline}</div>
          <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 500, marginTop: 6, lineHeight: 1.6 }}>{rec.rationale}</div>
        </div>
        <div style={{ textAlign: 'right', minWidth: 130 }}>
          {rec.suggested_amount > 0 && (
            <div className="num serif-num" style={{ fontSize: 17 }}>{fmtINR(rec.suggested_amount)}</div>
          )}
          {rec.potential_tax_saving > 0 && (
            <div style={{ fontSize: 12, color: 'var(--pos)', fontWeight: 700, marginTop: 2 }}>
              +{fmtCompact(rec.potential_tax_saving)} tax saved
            </div>
          )}
        </div>
      </div>

      {rec.status === 'new' && !superseded && (
        <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
          <button className="btn ghost sm" disabled={busy} onClick={() => onUpdate(rec.id, 'actioned')}>
            <Icon name="check" size={14} /> Mark actioned
          </button>
          <button className="btn ghost sm" disabled={busy} onClick={() => onUpdate(rec.id, 'dismissed')}>
            <Icon name="x" size={14} /> Dismiss
          </button>
        </div>
      )}
    </div>
  )
}

export default function RecommendationsPanel({ data, regime }) {
  const { update, reload } = useData()
  const [generating, setGenerating] = useState(false)
  const [updatingId, setUpdatingId] = useState(null)
  const [error, setError] = useState(null)
  const [note, setNote] = useState(null)

  const recommendations = [...(data.tax_recommendations || [])].sort((a, b) =>
    (b.generated_date || '').localeCompare(a.generated_date || '')
  )

  const handleGenerate = async () => {
    setGenerating(true)
    setError(null)
    setNote(null)
    try {
      const result = await generateTaxRecommendations(regime)
      if (result?.note) {
        setNote(result.note)
      } else {
        await reload()
      }
    } catch (e) {
      setError(e.message || 'Failed to generate recommendations.')
    } finally {
      setGenerating(false)
    }
  }

  const handleUpdate = async (id, status) => {
    setUpdatingId(id)
    try {
      await update('TaxRecommendations', id, { status })
    } finally {
      setUpdatingId(null)
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12, marginBottom: 14 }}>
        <div style={KICK}>Tax-saving recommendations</div>
        <button className="btn primary sm" onClick={handleGenerate} disabled={generating}>
          <Icon name="refresh" size={14} />
          {generating ? 'Generating…' : recommendations.length > 0 ? 'Regenerate' : 'Generate recommendations'}
        </button>
      </div>

      {error && (
        <div style={{ marginBottom: 14, padding: '10px 14px', borderRadius: 6, background: 'var(--neg-soft)', color: 'var(--neg)', fontSize: 12.5, lineHeight: 1.6 }}>
          {error}
        </div>
      )}

      {note && (
        <div style={{ marginBottom: 14, padding: '10px 14px', borderRadius: 6, background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink-2)', fontSize: 12.5, lineHeight: 1.6 }}>
          {note}
        </div>
      )}

      {recommendations.length === 0 && !generating && (
        <div className="empty-hint">
          No recommendations yet — click "Generate recommendations" for actionable, fact-grounded tax and investment suggestions based on your actual FDs, deductions, and holdings.
        </div>
      )}

      {recommendations.map(rec => (
        <RecommendationCard key={rec.id} rec={rec} onUpdate={handleUpdate} busy={updatingId === rec.id} />
      ))}

      <div style={{ fontSize: 11, color: 'var(--ink-3)', fontStyle: 'italic', marginTop: recommendations.length > 0 ? 14 : 0 }}>
        Generated from your actual data — regenerating reviews prior recommendations rather than starting over. Not financial advice; consult a qualified CA before acting.
      </div>
    </div>
  )
}
