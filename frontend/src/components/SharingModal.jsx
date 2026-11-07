import { useCallback, useEffect, useState } from 'react'
import { Modal, Field } from './Primitives.jsx'
import { Icon } from './Icons.jsx'
import QRCode from './QRCode.jsx'
import { KICK } from '../data/tokens.js'
import {
  listShares, createShare, resetShare, revokeShare, deleteShare, shareAccessLog,
} from '../data/shareApi.js'

const STATUS_PILL = { active: 'pos', revoked: 'neg', expired: 'neutral', locked: 'gold' }
const EVENT_LABEL = {
  verified: 'Unlocked with code', viewed: 'Viewed report', pdf: 'Downloaded PDF',
  bad_code: 'Wrong code', locked: 'Blocked (locked out)', denied: 'Blocked (revoked/expired)',
}

const fmtWhen = iso => iso
  ? new Date(iso).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit' })
  : 'Never'

// Owner-side management of family share links. View switches between the
// recipient list, the add form, the one-time enrolment screen and a
// recipient's access log. canCreate is false when the hosted plan doesn't
// include family sharing: existing links can still be revoked or removed.
export default function SharingModal({ onClose, showToast, canCreate = true }) {
  const [view, setView] = useState({ name: 'list' })
  const [shares, setShares] = useState(null)
  const [error, setError] = useState(null)

  const refresh = useCallback(() => {
    listShares().then(setShares).catch(e => setError(e.message))
  }, [])
  useEffect(refresh, [refresh])

  const run = async (fn, onOk) => {
    setError(null)
    try { onOk(await fn()) } catch (e) { setError(e.message) }
  }

  const title = {
    list: 'Family sharing', add: 'Add family member', enrol: 'Set up access', log: 'Access log',
  }[view.name]

  return (
    <Modal title={title} onClose={onClose} width={520}>
      {error && <div className="share-error">{error}</div>}

      {view.name === 'list' && (
        <ShareList
          shares={shares}
          canCreate={canCreate}
          onAdd={() => setView({ name: 'add' })}
          onLog={s => setView({ name: 'log', share: s })}
          onReset={s => {
            if (!window.confirm(`Reset access for ${s.name}? Their old link and authenticator entry stop working, and you'll need to set them up again.`)) return
            run(() => resetShare(s.id), enr => { setView({ name: 'enrol', enrolment: enr }); refresh() })
          }}
          onRevoke={s => {
            if (!window.confirm(`Revoke ${s.name}'s access? Their link stops working immediately.`)) return
            run(() => revokeShare(s.id), () => { showToast?.(`${s.name}'s access revoked`); refresh() })
          }}
          onDelete={s => {
            if (!window.confirm(`Remove ${s.name} from family sharing?`)) return
            run(() => deleteShare(s.id), () => { showToast?.(`${s.name} removed`); refresh() })
          }}
        />
      )}

      {view.name === 'add' && (
        <AddForm
          onCancel={() => setView({ name: 'list' })}
          onSubmit={form => run(() => createShare(form), enr => { setView({ name: 'enrol', enrolment: enr }); refresh() })}
        />
      )}

      {view.name === 'enrol' && (
        <Enrolment enrolment={view.enrolment} onDone={() => setView({ name: 'list' })} showToast={showToast} />
      )}

      {view.name === 'log' && (
        <AccessLog share={view.share} onBack={() => setView({ name: 'list' })} />
      )}
    </Modal>
  )
}

function ShareList({ shares, canCreate, onAdd, onLog, onReset, onRevoke, onDelete }) {
  return (
    <>
      <p className="share-intro">
        Give family members a read-only view of the whole-family report. Each person gets
        their own link and unlocks it with a 6-digit code from an authenticator app on their
        phone. No password, and no Google account needed. They're asked for a code again after a day.
      </p>
      {shares === null ? (
        <div style={KICK}>Loading…</div>
      ) : shares.length === 0 ? (
        <div className="share-empty">No one has access yet.</div>
      ) : (
        <ul className="share-list">
          {shares.map(s => (
            <li key={s.id} className="share-row">
              <div className="share-row-main">
                <div className="cell-strong">
                  {s.name}{s.relation && <span className="cell-sub" style={{ marginLeft: 6 }}>{s.relation}</span>}
                </div>
                <div className="cell-sub">
                  <span className={`pill ${STATUS_PILL[s.status] || 'neutral'}`} style={{ marginRight: 8 }}>{s.status}</span>
                  Last opened: {fmtWhen(s.last_opened)}
                  {s.expires && <> · until {s.expires}</>}
                </div>
              </div>
              <div className="share-row-actions">
                <button className="btn ghost sm" onClick={() => onLog(s)}>Log</button>
                {canCreate && <button className="btn ghost sm" onClick={() => onReset(s)}>Reset</button>}
                {s.status !== 'revoked' && <button className="btn ghost sm" onClick={() => onRevoke(s)}>Revoke</button>}
                <button className="btn ghost sm" onClick={() => onDelete(s)} title="Remove"><Icon name="trash" size={14} /></button>
              </div>
            </li>
          ))}
        </ul>
      )}
      {canCreate ? (
        <div className="share-actions">
          <button className="btn primary" onClick={onAdd}><Icon name="plus" size={14} /> Add family member</button>
        </div>
      ) : (
        <p className="hosted-fine">Adding people to family sharing isn't included in your current plan. Existing links keep working, and you can still revoke them.</p>
      )}
    </>
  )
}

function AddForm({ onCancel, onSubmit }) {
  const [form, setForm] = useState({ name: '', relation: '', expires: '' })
  const [busy, setBusy] = useState(false)
  const set = k => e => setForm(f => ({ ...f, [k]: e.target.value }))
  const submit = async e => {
    e.preventDefault()
    if (!form.name.trim()) return
    setBusy(true)
    await onSubmit(form)
    setBusy(false)
  }
  return (
    <form onSubmit={submit}>
      <Field label="Name">
        <input className="input" value={form.name} onChange={set('name')} placeholder="e.g. Mom" autoFocus required />
      </Field>
      <Field label="Relation (optional)">
        <input className="input" value={form.relation} onChange={set('relation')} placeholder="e.g. Mother" />
      </Field>
      <Field label="Access until (optional)">
        <input className="input" type="date" value={form.expires} onChange={set('expires')} />
      </Field>
      <p className="share-intro">
        Next you'll get a setup QR code. Have them nearby with their phone, because it's shown only once.
      </p>
      <div className="share-actions">
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
        <button className="btn primary" disabled={busy || !form.name.trim()}>{busy ? 'Creating…' : 'Continue'}</button>
      </div>
    </form>
  )
}

function Enrolment({ enrolment, onDone, showToast }) {
  const link = window.location.origin + enrolment.link_path
  const name = enrolment.share.name
  const canShare = typeof navigator !== 'undefined' && !!navigator.share

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link)
      showToast?.('Link copied')
    } catch {
      window.prompt('Copy this link:', link)
    }
  }
  const send = () => navigator.share({
    title: 'Kosh family report',
    text: `${name}, here's your Kosh link. Open it and enter the code from your authenticator app.`,
    url: link,
  }).catch(() => {})

  return (
    <>
      <section className="enrol-step">
        <div className="enrol-step-head"><span className="enrol-num">1</span> Set up {name}'s authenticator, in person</div>
        <p className="share-intro">
          On their phone, open <strong>Google Authenticator</strong> (or Microsoft Authenticator, 1Password…),
          tap <strong>+</strong> → <strong>Scan a QR code</strong>, and scan this.
        </p>
        <div className="enrol-qr">
          <QRCode value={enrolment.otpauth_uri} label={`Authenticator setup code for ${name}`} />
        </div>
        <div className="enrol-secret">
          Can't scan? Enter this key manually: <code>{enrolment.secret.match(/.{1,4}/g).join(' ')}</code>
        </div>
        <div className="share-warn">
          This setup code is shown only once. Never send it over WhatsApp, SMS or email: anyone with it
          and the link can open your report.
        </div>
      </section>

      <section className="enrol-step">
        <div className="enrol-step-head"><span className="enrol-num">2</span> Give them their link</div>
        <p className="share-intro">
          It's safe to send on chat, because it's useless without the code from their phone.
        </p>
        <div className="enrol-link">
          <input className="input" value={link} readOnly onFocus={e => e.target.select()} />
          <button className="btn" onClick={copy}>Copy</button>
          {canShare && <button className="btn" onClick={send}>Send…</button>}
        </div>
        <details className="enrol-linkqr">
          <summary>Show link as QR (to print or scan)</summary>
          <div className="enrol-qr"><QRCode value={link} size={150} label={`Share link for ${name}`} /></div>
        </details>
      </section>

      <div className="share-actions">
        <button className="btn primary" onClick={onDone}><Icon name="check" size={14} /> Done</button>
      </div>
    </>
  )
}

function AccessLog({ share, onBack }) {
  const [logs, setLogs] = useState(null)
  const [error, setError] = useState(null)
  useEffect(() => { shareAccessLog(share.id).then(setLogs).catch(e => setError(e.message)) }, [share.id])
  return (
    <>
      <div style={{ ...KICK, marginBottom: 12 }}>{share.name}: last 50 events</div>
      {error && <div className="share-error">{error}</div>}
      {logs === null && !error ? <div style={KICK}>Loading…</div> : logs?.length === 0 ? (
        <div className="share-empty">No activity yet.</div>
      ) : (
        <ul className="share-list">
          {(logs || []).map(l => (
            <li key={l.id} className="share-row">
              <div className="share-row-main">
                <div className={l.event === 'bad_code' || l.event === 'locked' || l.event === 'denied' ? 'cell-strong neg' : 'cell-strong'}>
                  {EVENT_LABEL[l.event] || l.event}
                </div>
                <div className="cell-sub">{fmtWhen(l.at)} · {l.ip}</div>
              </div>
            </li>
          ))}
        </ul>
      )}
      <div className="share-actions">
        <button className="btn" onClick={onBack}>Back</button>
      </div>
    </>
  )
}
