# Operating hosted Kosh

This is the runbook for running Kosh as the paid, multi-household service
(`KOSH_MODE=hosted`). **Self-hosting your own family's Kosh doesn't need any
of this.** See [SELF_HOSTING.md](SELF_HOSTING.md) instead. The design and
reasoning are in [HOSTED_PLAN.md](../HOSTED_PLAN.md).

## How it differs from self-hosting

| | Self-hosted | Hosted |
|---|---|---|
| Households | One, from `SPREADSHEET_ID` | Many, one Sheet each, in each customer's own Drive |
| Sheets access | Service-account key | Each owner's OAuth grant (`drive.file` scope only) |
| Who can sign in | `ALLOWED_EMAILS` | Any verified Google account; membership decides access |
| Extra datastore | None | Firestore "control plane": households, members, invites, billing, usage. **No financial data.** |
| Billing | None | Razorpay subscriptions; read-only when lapsed |

## Prerequisites

1. **Google OAuth client** (Web application) with:
   - Authorized JavaScript origins: the hosted URL
   - **Client secret** (hosted mode exchanges authorization codes server-side)
   - OAuth consent screen with the `drive.file` scope, a privacy policy and
     terms on your domain, and **brand verification**. Google reviews this,
     and it can take weeks, so start early. `drive.file` is a non-sensitive
     scope, so no paid security assessment is required (re-check at
     submission time).
2. **Firestore** database (Native mode) in the project:
   - Security rules that **deny all client access**. Only the server, via
     its service account, reads and writes:
     ```
     rules_version = '2';
     service cloud.firestore { match /databases/{db}/documents { match /{doc=**} { allow read, write: if false; } } }
     ```
   - A **TTL policy** on `webhookEvents.expireAt` (processed webhook IDs are
     kept 7 days).
   - The Cloud Run service account needs `roles/datastore.user`.
3. **Razorpay** account (KYC done), with:
   - Two **plans**, monthly and yearly. Their prices are what customers pay;
     keep `frontend/src/data/pricing.js` (marketing copy) in sync.
   - A **webhook** to `https://<host>/api/billing/webhook` with a secret, for
     the `subscription.*` events.

## Configuration

Everything from self-hosting except `SPREADSHEET_ID`, `CREDENTIALS_PATH` and
`ALLOWED_EMAILS`, plus:

| Variable | Notes |
|---|---|
| `KOSH_MODE` | `hosted` |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | The OAuth client above |
| `SESSION_SECRET` | 32+ characters |
| `SHARE_ENCRYPTION_KEY` | 32+ characters (family sharing is always on in hosted mode) |
| `CONTROL_PLANE_ENCRYPTION_KEY` | 32+ characters. Encrypts each household's Google grant. **Losing or changing it disconnects every household** (owners must reconnect Drive) |
| `FIRESTORE_PROJECT` | Defaults to `GOOGLE_CLOUD_PROJECT` |
| `FIRESTORE_DATABASE` | Defaults to `(default)` |
| `RAZORPAY_KEY_ID` / `RAZORPAY_KEY_SECRET` | API keys (test keys for staging) |
| `RAZORPAY_WEBHOOK_SECRET` | The webhook's secret, 16+ characters |
| `RAZORPAY_PLAN_MONTHLY` / `RAZORPAY_PLAN_ANNUAL` | Plan IDs (`plan_...`). Loaded at startup, so a wrong ID fails boot |
| `COOKIE_SECURE` | Must be `true` (the default) |
| `OPERATOR_EMAILS` | Optional. Comma-separated Google accounts allowed to change [feature tiers](#feature-tiers). Without it the saved policy (or the defaults) can't be changed |

Hosted mode **refuses to start** if any of these (except `OPERATOR_EMAILS`) are missing or too short.

The pricing page, its nav link and the "Hosted Kosh" pointer on the docs
page only appear when the server runs in hosted mode (the frontend asks
`/api/auth/config`); a self-hosted deploy redirects `/pricing` home. If
you later show pricing from a deploy that *isn't* the hosted service, build
its frontend with `VITE_HOSTED_APP_URL=https://<hosted-host>/app` so
"Start free trial" goes to the right place.

## Deploying

Hosted Kosh is the same Docker image as a self-hosted deploy, run as a
second Cloud Run service (`kosh-hosted`; Terraform in
jenish-jain/infrastructure, behind `create_kosh_hosted`). Set the repository
variable `DEPLOY_HOSTED=true` and every push to `master` deploys the image
to both services. Env vars and secrets are owned by Terraform; CI only
changes the image.

## Plans, trials and access

- New households get a **14-day trial** (`hosted.TrialLength`). Subscribing
  during the trial defers the first charge to its end.
- Every household is on a **tier**: **trial** while the trial runs, **paid**
  while a subscription is active, being retried (`past_due`) or cancelled
  with paid time left, and **free** otherwise. What each tier may do is the
  [feature policy](#feature-tiers); by default trial and paid get
  everything and free is **read-only**. A feature the tier leaves out
  answers `402` with an explanation.
- **AI allowance** per household per month (`hosted.MonthlyQuotas`): Ask 200,
  document scans 30, tax runs 20. Failed calls are refunded.
- Households have up to **6 people** including pending invites.

## Feature tiers

Operators (`OPERATOR_EMAILS`) open **Household → Feature tiers** in the app
to switch each feature on or off per tier:

| Feature | Covers | Default: free / trial / paid |
|---|---|---|
| Adding and editing data | Every change to the household's data | off / on / on |
| Family sharing | Creating or resetting share links | off / on / on |
| Household invites | Inviting people to the household | on / on / on |
| Ask Kosh | AI questions | off / on / on |
| Document scanning | AI reading of uploads | off / on / on |
| Tax recommendations | AI tax suggestions | off / on / on |

- The defaults are the original read-only rule, so nothing changes until a
  policy is saved. Features a saved policy doesn't mention keep their
  default.
- Reading and exporting data, billing, deleting the household, accepting an
  invite and revoking or deleting share links are **never** gated. Switching
  off family sharing stops new links; existing links keep working until the
  owner revokes them.
- The policy is one Firestore document (`settings/featurePolicy`). Each
  instance caches it for 30 seconds, and keeps its last copy if Firestore
  is briefly unreachable.
- The API is `GET`/`PUT /api/operator/features`; anyone who isn't an
  operator gets a 404. Each change is logged with the operator's email.
- The pricing page and its FAQ are static text describing the defaults:
  update `frontend/src/screens/marketing/Pricing.jsx` when you change what
  a tier includes.

## Testing locally without Google or Razorpay

```bash
cd backend
go build -tags hosteddev -o kosh-hosteddev .   # never use this binary in production
OPERATOR_EMAILS=you@example.com FRONTEND_DIST=../frontend/dist ./kosh-hosteddev
```

Sign in at `/app` with any email. Drive and payments are simulated, and
`/api/dev/expire-trial` and `/api/dev/revoke` exercise the read-only and
reconnect states. To run the real Firestore store code locally, start the
emulator and set `FIRESTORE_EMULATOR_HOST`:

```bash
docker run -d --rm -p 8085:8085 gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators \
  gcloud emulators firestore start --host-port=0.0.0.0:8085
FIRESTORE_EMULATOR_HOST=localhost:8085 ./kosh-hosteddev
```

The same emulator runs the control-plane contract tests:
`FIRESTORE_EMULATOR_HOST=localhost:8085 go test ./internal/controlplane/`.

## Before launch: verify against Razorpay test mode

The billing integration was built without live Razorpay access. With test
keys, confirm the subscription status names, `start_at`/`charge_at`/`current_end`,
`cancel_at_cycle_end` behaviour, the `X-Razorpay-Event-Id` header, and that
Checkout loads under the CSP (`api.WithCSPSources` in `internal/app`). The
status mapping is `billing.Apply`.
