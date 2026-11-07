import { KICK, SERIF } from '../../data/tokens.js'

// Shared frame for the hosted pre-app screens (onboarding, join, reconnect).
export default function CenterCard({ kicker, title, children, onSignOut, email }) {
  return (
    <div className="hosted-page">
      <div className="hosted-card">
        <div style={{ fontFamily: SERIF, fontSize: 56, color: 'var(--accent)', lineHeight: 1, marginBottom: 12 }}>क</div>
        {kicker && <div style={{ ...KICK, marginBottom: 8 }}>{kicker}</div>}
        <h1 className="hosted-title">{title}</h1>
        {children}
        {onSignOut && (
          <div className="hosted-foot">
            Signed in as <strong>{email}</strong> · <button className="linklike" onClick={onSignOut}>Sign out</button>
          </div>
        )}
      </div>
    </div>
  )
}

export function ErrorNote({ children }) {
  return children ? <div className="share-error" role="alert" style={{ textAlign: 'left' }}>{children}</div> : null
}
