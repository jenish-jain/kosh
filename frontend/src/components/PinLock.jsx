import { useState, useEffect } from 'react'

const DOT_STYLE = (filled) => ({
  width: 14, height: 14, borderRadius: '50%',
  background: filled ? 'var(--accent)' : 'none',
  border: '2px solid ' + (filled ? 'var(--accent)' : 'var(--ink-3)'),
  transition: 'background .15s, border-color .15s',
})

const PAD = [
  ['1','2','3'],
  ['4','5','6'],
  ['7','8','9'],
  [null,'0','⌫'],
]

export default function PinLock({ onVerify }) {
  const [digits, setDigits] = useState([])
  const [shake, setShake] = useState(false)

  const push = digit => {
    setDigits(d => {
      const next = [...d, digit]
      if (next.length === 4) {
        onVerify(next.join('')).then(ok => {
          if (!ok) {
            setShake(true)
            setTimeout(() => { setShake(false); setDigits([]) }, 600)
          }
        })
      }
      return next.length <= 4 ? next : d
    })
  }

  const pop = () => setDigits(d => d.slice(0, -1))

  useEffect(() => {
    const onKey = e => {
      if (e.key >= '0' && e.key <= '9') push(e.key)
      else if (e.key === 'Backspace') pop()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  })

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 9999,
      background: 'var(--bg)',
      display: 'flex', flexDirection: 'column',
      alignItems: 'center', justifyContent: 'center', gap: 32,
    }}>
      <style>{`
        @keyframes pin-shake {
          0%,100% { transform: translateX(0) }
          20%      { transform: translateX(-10px) }
          40%      { transform: translateX(10px) }
          60%      { transform: translateX(-8px) }
          80%      { transform: translateX(8px) }
        }
      `}</style>

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 24, fontWeight: 700, color: 'var(--ink)' }}>Kosh</div>
        <div style={{ fontSize: 13, color: 'var(--ink-3)', marginTop: 4 }}>Enter PIN to continue</div>
      </div>

      <div style={{
        display: 'flex', gap: 16,
        animation: shake ? 'pin-shake .6s ease' : 'none',
      }}>
        {[0,1,2,3].map(i => <div key={i} style={DOT_STYLE(i < digits.length)} />)}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 72px)', gap: 12 }}>
        {PAD.flat().map((k, i) => k === null ? (
          <div key={i} />
        ) : (
          <button
            key={i}
            onClick={() => k === '⌫' ? pop() : push(k)}
            style={{
              height: 72, borderRadius: 12,
              background: 'var(--surface)', border: '1px solid var(--border)',
              fontSize: k === '⌫' ? 20 : 22, fontWeight: 500,
              color: 'var(--ink)', cursor: 'pointer',
              transition: 'background .1s',
            }}
            onMouseDown={e => e.currentTarget.style.background = 'var(--border)'}
            onMouseUp={e => e.currentTarget.style.background = 'var(--surface)'}
            onMouseLeave={e => e.currentTarget.style.background = 'var(--surface)'}
          >
            {k}
          </button>
        ))}
      </div>
    </div>
  )
}
