const Ic = {
  dash:    "M3 12 12 3l9 9M5 10v10h5v-6h4v6h5V10",
  invest:  "M3 17l6-6 4 4 7-8M21 7v5M21 7h-5",
  sip:     "M7 3v3M17 3v3M3.5 8.5h17M5 6h14a1.5 1.5 0 0 1 1.5 1.5V20A1.5 1.5 0 0 1 19 21.5H5A1.5 1.5 0 0 1 3.5 20V7.5A1.5 1.5 0 0 1 5 6ZM8 13h2M8 17h2M14 13h2",
  members: "M16 19c0-2.2-1.8-4-4-4s-4 1.8-4 4M12 11.5a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM19.5 18c0-1.6-1-3-2.5-3.5M17 10.8a2.4 2.4 0 0 0 0-4.6M4.5 18c0-1.6 1-3 2.5-3.5M7 10.8a2.4 2.4 0 0 1 0-4.6",
  tax:     "M7 3.5h10A1.5 1.5 0 0 1 18.5 5v15l-2.5-1.6-2 1.3-2-1.3-2 1.3-2-1.3L5.5 20V5A1.5 1.5 0 0 1 7 3.5ZM9 8.5l6 6M9.5 9.2a.7.7 0 1 0 0-.1M14.7 14.6a.7.7 0 1 0 0-.1",
  expense: "M3.5 7.5h17v11a1.5 1.5 0 0 1-1.5 1.5H5a1.5 1.5 0 0 1-1.5-1.5ZM3.5 7.5 6 4h12l2.5 3.5M9 12h6",
  plus:    "M12 5v14M5 12h14",
  edit:    "M4 20h4L19 9l-4-4L4 16v4ZM14 6l4 4",
  pause:   "M9 5v14M15 5v14",
  play:    "M7 5l12 7-12 7Z",
  chev:    "M9 6l6 6-6 6",
  chevD:   "M6 9l6 6 6-6",
  search:  "M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14ZM20 20l-3.5-3.5",
  upload:  "M12 16V4M7 9l5-5 5 5M5 20h14",
  download: "M12 4v12M7 11l5 5 5-5M5 20h14",
  trash:   "M4 6h16M9 6V4h6v2M19 6l-1 14H6L5 6",
  check:   "M5 12l4.5 4.5L19 7",
  save:    "M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2ZM17 21v-8H7v8M7 3v5h8",
  refresh: "M1 4v6h6M23 20v-6h-6M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4-4.64 4.36A9 9 0 0 1 3.51 15",
  x:       "M6 6l12 12M18 6 6 18",
  family:  "M16 19c0-2.2-1.8-4-4-4s-4 1.8-4 4M12 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6",
  link:    "M14 5h5v5M19 5l-9 9M9 5H7a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-2",
  income:  "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8ZM14 2v6h6M9 13h6M9 17h3",
}

export function Icon({ name, size = 18, sw = 1.8, className = '', style }) {
  return (
    <svg
      className={'ico ' + className}
      width={size} height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={sw}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={style}
    >
      <path d={Ic[name]} />
    </svg>
  )
}
