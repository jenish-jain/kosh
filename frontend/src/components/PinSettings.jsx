import { useState, useRef } from 'react'
import { Modal, Field } from './Primitives.jsx'

const TIMEOUT_OPTIONS = [
  { value: 1,  label: '1 minute' },
  { value: 5,  label: '5 minutes' },
  { value: 15, label: '15 minutes' },
  { value: 30, label: '30 minutes' },
  { value: 0,  label: 'Never' },
]

const inputStyle = {
  width: '100%', padding: '8px 12px', borderRadius: 8,
  border: '1px solid var(--border)', background: 'var(--surface)',
  color: 'var(--ink)', fontSize: 15, boxSizing: 'border-box',
  letterSpacing: 6, textAlign: 'center',
}

export default function PinSettings({ pinLock, onClose }) {
  const { pinConfigured, timeoutMinutes, checkPin, setupPin, clearPin, setTimeoutMinutes } = pinLock
  const [step, setStep] = useState('idle') // idle | verify-current | enter-new | confirm-new
  const [err, setErr] = useState('')
  const [newPin, setNewPin] = useState('')
  const actionRef = useRef(null) // 'setup' | 'change' | 'remove'
  const currentPinRef = useRef('')

  const reset = () => { setStep('idle'); setErr(''); setNewPin(''); currentPinRef.current = ''; actionRef.current = null }

  const handleSetup = () => { actionRef.current = 'setup'; setStep('enter-new') }
  const handleChange = () => { actionRef.current = 'change'; setStep('verify-current') }
  const handleRemove = () => { actionRef.current = 'remove'; setStep('verify-current') }

  const handleVerifyCurrent = async e => {
    e.preventDefault()
    const pin = e.target.pin.value.trim()
    if (pin.length !== 4) { setErr('Enter 4 digits'); return }
    const ok = await checkPin(pin)
    if (!ok) { setErr('Wrong PIN'); e.target.pin.value = ''; return }
    currentPinRef.current = pin
    setErr('')
    if (actionRef.current === 'remove') { clearPin(); onClose(); return }
    setStep('enter-new')
  }

  const handleEnterNew = e => {
    e.preventDefault()
    const pin = e.target.pin.value.trim()
    if (pin.length !== 4 || !/^\d{4}$/.test(pin)) { setErr('Enter 4 digits (numbers only)'); return }
    if (pin === currentPinRef.current) { setErr('New PIN must differ from current'); return }
    setErr(''); setNewPin(pin); setStep('confirm-new')
  }

  const handleConfirmNew = async e => {
    e.preventDefault()
    const pin = e.target.pin.value.trim()
    if (pin !== newPin) { setErr('PINs do not match'); e.target.pin.value = ''; return }
    await setupPin(pin)
    onClose()
  }

  const PinInput = ({ label, hint, onSubmit, submitLabel }) => (
    <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Field label={label}>
        <input
          name="pin"
          type="password"
          inputMode="numeric"
          maxLength={4}
          autoFocus
          autoComplete="off"
          placeholder="••••"
          style={inputStyle}
        />
      </Field>
      {hint && <div style={{ fontSize: 12, color: 'var(--ink-3)', marginTop: -8 }}>{hint}</div>}
      {err && <div style={{ color: 'var(--red)', fontSize: 13 }}>{err}</div>}
      <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
        <button type="button" onClick={reset} style={btnStyle()}>Cancel</button>
        <button type="submit" style={btnStyle(true)}>{submitLabel}</button>
      </div>
    </form>
  )

  return (
    <Modal title="Security" onClose={onClose}>
      {step === 'idle' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <div>
            <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--ink-3)', marginBottom: 8, textTransform: 'uppercase', letterSpacing: .8 }}>Auto-lock after</div>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              {TIMEOUT_OPTIONS.map(o => (
                <button
                  key={o.value}
                  onClick={() => setTimeoutMinutes(o.value)}
                  style={{
                    padding: '6px 14px', borderRadius: 20, fontSize: 13, cursor: 'pointer',
                    background: timeoutMinutes === o.value ? 'var(--accent)' : 'var(--surface)',
                    color: timeoutMinutes === o.value ? '#fff' : 'var(--ink)',
                    border: '1px solid ' + (timeoutMinutes === o.value ? 'var(--accent)' : 'var(--border)'),
                    fontWeight: timeoutMinutes === o.value ? 600 : 400,
                  }}
                >
                  {o.label}
                </button>
              ))}
            </div>
            {!pinConfigured && timeoutMinutes !== 0 && (
              <div style={{ fontSize: 12, color: 'var(--ink-3)', marginTop: 8 }}>Set a PIN below to enable auto-lock.</div>
            )}
          </div>

          <hr style={{ border: 'none', borderTop: '1px solid var(--border)', margin: '4px 0' }} />

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {!pinConfigured ? (
              <button onClick={handleSetup} style={btnStyle(true)}>Set up PIN</button>
            ) : (
              <>
                <button onClick={handleChange} style={btnStyle(true)}>Change PIN</button>
                <button onClick={handleRemove} style={{ ...btnStyle(), color: 'var(--red)', borderColor: 'var(--red)' }}>Remove PIN</button>
              </>
            )}
          </div>
        </div>
      )}

      {step === 'verify-current' && (
        <PinInput
          label="Current PIN"
          onSubmit={handleVerifyCurrent}
          submitLabel={actionRef.current === 'remove' ? 'Remove PIN' : 'Next'}
        />
      )}

      {step === 'enter-new' && (
        <PinInput
          label="New PIN"
          hint="4 digits"
          onSubmit={handleEnterNew}
          submitLabel="Next"
        />
      )}

      {step === 'confirm-new' && (
        <PinInput
          label="Confirm new PIN"
          onSubmit={handleConfirmNew}
          submitLabel="Save PIN"
        />
      )}
    </Modal>
  )
}

function btnStyle(primary = false) {
  return {
    padding: '8px 18px', borderRadius: 8, fontSize: 13, fontWeight: 600,
    cursor: 'pointer', border: '1px solid var(--border)',
    background: primary ? 'var(--accent)' : 'var(--surface)',
    color: primary ? '#fff' : 'var(--ink)',
  }
}
