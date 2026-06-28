import { Avatar } from '../Primitives.jsx'

export default function MemberSwitcher({ member, setMember, members }) {
  return (
    <div className="memberbar">
      <button
        className={'member-chip' + (!member ? ' active' : '')}
        onClick={() => setMember(null)}
        title="Whole family"
      >
        <span className="avatar" style={{ background: 'var(--ink)', width: 26, height: 26, flexBasis: 26, fontSize: 12 }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M16 19c0-2.2-1.8-4-4-4s-4 1.8-4 4M12 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6" />
          </svg>
        </span>
        <span className="member-chip-name">Whole family</span>
      </button>
      {(members || []).map(m => (
        <button
          key={m.id}
          className={'member-chip' + (member === m.id ? ' active' : '')}
          onClick={() => setMember(m.id)}
          title={m.name}
        >
          <Avatar member={m} />
          <span className="member-chip-name">{m.name}</span>
        </button>
      ))}
    </div>
  )
}
