// One-time Google authorization for hosted Kosh: the household owner grants
// the narrow drive.file scope (Kosh only ever sees the spreadsheet it
// creates), and the backend exchanges the returned code for long-lived
// access so the household's data is reachable for every member.
//
// Like requestDriveAccessToken in driveAuth.js, this opens a popup and must
// be called synchronously from a click handler — no await beforehand.

const SCOPE = 'openid email https://www.googleapis.com/auth/drive.file'

export function requestDriveCode({ clientId, email, devLogin }) {
  // hosteddev builds simulate Google: the backend's fake accepts this form.
  if (devLogin) return Promise.resolve(`code:${email}`)

  const oauth2 = window.google?.accounts?.oauth2
  if (!oauth2) return Promise.reject(new Error('Google sign-in is still loading — try again in a moment.'))
  return new Promise((resolve, reject) => {
    const client = oauth2.initCodeClient({
      client_id: clientId,
      scope: SCOPE,
      ux_mode: 'popup',
      login_hint: email,
      callback: resp => {
        if (resp.error) reject(new Error(resp.error_description || resp.error))
        else resolve(resp.code)
      },
      error_callback: err => reject(new Error(err?.type === 'popup_closed' ? 'The Google window was closed before finishing.' : (err?.message || 'Google sign-in failed.'))),
    })
    client.requestCode()
  })
}
