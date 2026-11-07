import { useEffect, useState } from 'react'
import { Modal } from './Primitives.jsx'
import { KICK } from '../data/tokens.js'
import { getFeaturePolicy, saveFeaturePolicy } from '../data/hostedApi.js'

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b)

// Operator only: switch each feature on or off per tier (free, trial,
// paid) for every hosted household. The server enforces the result.
export default function FeatureTiersModal({ onClose, onSaved, showToast }) {
  const [data, setData] = useState(null)
  const [draft, setDraft] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    getFeaturePolicy().then(d => { setData(d); setDraft(d.policy) }).catch(e => setError(e.message))
  }, [])

  const toggle = (tier, feature) =>
    setDraft(d => ({ ...d, [tier]: { ...d[tier], [feature]: !d[tier][feature] } }))

  const save = async () => {
    setError(null)
    setBusy(true)
    try {
      const d = await saveFeaturePolicy(draft)
      setData(d)
      setDraft(d.policy)
      showToast?.('Feature tiers saved')
      onSaved?.()
    } catch (e) { setError(e.message) } finally { setBusy(false) }
  }

  const dirty = data && !same(draft, data.policy)

  return (
    <Modal title="Feature tiers" onClose={onClose} width={600}>
      {error && <div className="share-error">{error}</div>}
      {!data ? (!error && <div style={KICK}>Loading…</div>) : (
        <>
          <p className="share-intro">
            Choose what households get on each tier. Households move between tiers as their trial
            or subscription starts and ends. Viewing and exporting data, billing, deleting a household
            and revoking share links are always available.
          </p>
          <div className="tiers-scroll">
            <table className="tiers-table">
              <thead>
                <tr>
                  <th scope="col">Feature</th>
                  {data.tiers.map(t => <th key={t.key} scope="col" title={t.help}>{t.label}</th>)}
                </tr>
              </thead>
              <tbody>
                {data.features.map(f => (
                  <tr key={f.key}>
                    <th scope="row">
                      <div className="cell-strong">{f.label}</div>
                      <div className="cell-sub">{f.help}</div>
                    </th>
                    {data.tiers.map(t => {
                      const on = !!draft[t.key]?.[f.key]
                      const changed = on !== !!data.defaults[t.key]?.[f.key]
                      return (
                        <td key={t.key}>
                          <label className={`tier-toggle${changed ? ' changed' : ''}`} title={changed ? 'Differs from the default' : undefined}>
                            <input type="checkbox" checked={on} disabled={busy} onChange={() => toggle(t.key, f.key)}
                              aria-label={`${f.label} on ${t.label}`} />
                          </label>
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="hosted-fine">
            {data.updated_by
              ? <>Last changed by {data.updated_by} on {new Date(data.updated_at).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit' })}. </>
              : <>Using the defaults. </>}
            Highlighted boxes differ from the defaults. Saved changes reach every server within 30 seconds.
          </p>
          <div className="share-actions">
            <button className="btn" disabled={busy || same(draft, data.defaults)} onClick={() => setDraft(data.defaults)}>Back to defaults</button>
            <button className="btn primary" disabled={busy || !dirty} onClick={save}>{busy ? 'Saving…' : 'Save'}</button>
          </div>
        </>
      )}
    </Modal>
  )
}
