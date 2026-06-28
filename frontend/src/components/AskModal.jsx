import { useState, useRef, useEffect } from 'react'
import { chatMessage } from '../data/api.js'

const SUGGESTIONS = [
  'How is my portfolio allocated?',
  'Am I saving enough each month?',
  'What is my net worth breakdown?',
  'Which SIPs should I increase?',
  'How much insurance coverage do I have?',
  'How many months of expenses do I have saved?',
]

export default function AskModal({ onClose }) {
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const endRef = useRef(null)
  const inputRef = useRef(null)
  const textareaRef = useRef(null)

  useEffect(() => { inputRef.current?.focus() }, [])

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  useEffect(() => {
    const h = e => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])

  const send = async (text) => {
    const q = (text !== undefined ? text : input).trim()
    if (!q || loading) return
    setInput('')
    if (textareaRef.current) textareaRef.current.style.height = 'auto'
    const userMsg = { role: 'user', content: q }
    const next = [...messages, userMsg]
    setMessages(next)
    setLoading(true)
    try {
      const reply = await chatMessage(next)
      setMessages([...next, { role: 'assistant', content: reply }])
    } catch (e) {
      setMessages([...next, { role: 'assistant', content: `Sorry, I couldn't get a response. (${e.message})` }])
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div
        style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,.22)', zIndex: 1000 }}
        onClick={onClose}
      />
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: 420, maxWidth: '100vw',
        background: 'var(--surface)', zIndex: 1001, display: 'flex', flexDirection: 'column',
        boxShadow: '-4px 0 40px rgba(0,0,0,.13)',
      }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '16px 20px', borderBottom: '1px solid var(--line)' }}>
          <div style={{ width: 32, height: 32, borderRadius: '50%', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, color: '#fff', fontWeight: 700, flexShrink: 0 }}>
            क
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 700, fontSize: 14 }}>Ask Kosh</div>
            <div style={{ fontSize: 11, color: 'var(--ink-3)', marginTop: 1 }}>AI advisor · your data, your questions</div>
          </div>
          <button
            onClick={onClose}
            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 6, borderRadius: 8, color: 'var(--ink-3)', display: 'flex' }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        {/* Messages */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          {messages.length === 0 && !loading && (
            <div>
              <div style={{ fontSize: 11, color: 'var(--ink-3)', marginBottom: 10, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '.06em' }}>
                Try asking
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {SUGGESTIONS.map(s => (
                  <button
                    key={s}
                    onClick={() => send(s)}
                    className="ask-chip"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} style={{ display: 'flex', flexDirection: m.role === 'user' ? 'row-reverse' : 'row', gap: 8, alignItems: 'flex-start' }}>
              {m.role === 'assistant' && (
                <div style={{ width: 26, height: 26, borderRadius: '50%', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, color: '#fff', flexShrink: 0, fontWeight: 700, marginTop: 2 }}>
                  क
                </div>
              )}
              <div style={{
                maxWidth: '82%',
                background: m.role === 'user' ? 'var(--accent)' : 'var(--bg)',
                color: m.role === 'user' ? '#fff' : 'var(--ink)',
                borderRadius: m.role === 'user' ? '14px 14px 4px 14px' : '4px 14px 14px 14px',
                padding: '9px 13px', fontSize: 13.5, lineHeight: 1.55,
                whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                border: m.role === 'assistant' ? '1px solid var(--line)' : 'none',
              }}>
                {m.content}
              </div>
            </div>
          ))}
          {loading && (
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
              <div style={{ width: 26, height: 26, borderRadius: '50%', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, color: '#fff', flexShrink: 0, fontWeight: 700, marginTop: 2 }}>
                क
              </div>
              <div style={{ background: 'var(--bg)', border: '1px solid var(--line)', borderRadius: '4px 14px 14px 14px', padding: '12px 16px' }}>
                <div className="ask-typing"><span /><span /><span /></div>
              </div>
            </div>
          )}
          <div ref={endRef} />
        </div>

        {/* Input */}
        <div style={{ padding: '12px 20px 20px', borderTop: '1px solid var(--line)' }}>
          <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end' }}>
            <textarea
              ref={el => { inputRef.current = el; textareaRef.current = el }}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() } }}
              onInput={e => {
                e.currentTarget.style.height = 'auto'
                e.currentTarget.style.height = Math.min(e.currentTarget.scrollHeight, 120) + 'px'
              }}
              placeholder="Ask anything about your finances…"
              rows={1}
              className="ask-input"
            />
            <button
              onClick={() => send()}
              disabled={!input.trim() || loading}
              className={'ask-send' + (!input.trim() || loading ? ' disabled' : '')}
            >
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
              </svg>
            </button>
          </div>
          <div style={{ fontSize: 11, color: 'var(--ink-3)', marginTop: 8, textAlign: 'center' }}>
            Enter to send · Shift+Enter for new line
          </div>
        </div>
      </div>
    </>
  )
}
