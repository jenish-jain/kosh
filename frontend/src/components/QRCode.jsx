import { useEffect, useState } from 'react'
import QR from 'qrcode'

// Renders `value` as a QR code image — generated locally, so secrets (the
// enrolment otpauth:// URI) never go to a third-party QR service.
export default function QRCode({ value, size = 180, label }) {
  const [src, setSrc] = useState(null)
  useEffect(() => {
    let live = true
    QR.toDataURL(value, { width: size * 2, margin: 1, errorCorrectionLevel: 'M' })
      .then(url => { if (live) setSrc(url) })
      .catch(() => { if (live) setSrc(null) })
    return () => { live = false }
  }, [value, size])
  return (
    <div className="qr-box" style={{ width: size, height: size }}>
      {src && <img src={src} width={size} height={size} alt={label || 'QR code'} />}
    </div>
  )
}
