# Plan: Kosh as a hosted service (alongside self-hosting)

Goal: offer Kosh as a paid hosted service (monthly + annual plans) for
households who can't or don't want to self-host, **without** weakening the
self-host path or the core promise: *your financial data lives in your own
Google Sheet, not our database.*

Approved by user 2026-11-07.

---

## Starting point (single-tenant by construction)

- `backend/main.go` builds **one** Sheets client from a service-account file
  and **one** `SPREADSHEET_ID`; every repository/handler shares it.
- `ALLOWED_EMAILS` is a global env var — no notion of "whose household".
- Family-share links (`/s/<token>`) look recipients up in *the* Sheet — with
  many tenants there's no way to know which Sheet to look in.
- Already the right shape: document uploads use a **per-user OAuth token with
  the `drive.file` scope** (`frontend/src/data/driveAuth.js`).

## Core decision: tenant data stays in the customer's own Drive

Kosh creates the Sheet in the customer's Drive (or they pick an existing one
via Google Picker) using `drive.file`; the hosted service stores only an
**encrypted OAuth refresh token** per tenant.

Rejected alternatives:
- *Customer shares a Sheet with our service account* — clunky onboarding, and
  one leaked key reads every customer's finances.
- *Our own database* — discards the differentiator and makes us custodian of
  financial data (DPDP liability).

Consequences: a breach exposes no financial database; cancelling leaves the
data with the customer (they can even self-host against the same Sheet);
`drive.file` is a non-sensitive scope → lighter Google verification (no paid
security assessment). Re-check the scope classification at submission time.

## Architecture: one codebase, two modes

```
                    ┌────────── same Docker image ──────────┐
KOSH_MODE=selfhost  │  Single-tenant resolver (today's      │  env vars, service account,
  (default)         │  behaviour, unchanged)                │  ALLOWED_EMAILS
                    │                                       │
KOSH_MODE=hosted    │  Multi-tenant resolver ───────────────┼─▶ Control-plane DB (Firestore)
                    │   session → tenant → their OAuth token│    tenants, members, subscriptions,
                    │   → Sheets client (cached per tenant) │    encrypted refresh tokens,
                    │  + billing, entitlements, AI quotas   │    share-token → tenant index
                    └───────────────────────────────────────┘
                                   │
                                   ▼
                     Customer's own Google Sheet (their Drive)
```

- **Control plane** — the only new datastore, and it holds *no* financial
  data. Firestore: scale-to-zero, ~₹0 at this size, no VPC needed from Cloud
  Run. Refresh tokens envelope-encrypted with Cloud KMS.
- **`TenantResolver`** — per-request repositories instead of one global set.
  Self-host's implementation resolves everything to the env-configured Sheet.
  The same seam later carries entitlements, AI quotas, and share-token lookup.
- **Batched reads** — read all tabs in one `values.batchGet` instead of ~15
  `ReadSheet` calls. All tenants share one GCP project's Sheets quota
  (300 reads/min/project by default), so this is a prerequisite for hosting.

## Pricing & packaging (placeholders — validate before launch)

Per-tenant infra cost is ~₹0 (data lives in their Drive); the real variable
costs are AI calls and payment fees (~2%).

| Plan | Price | Includes |
|---|---|---|
| **Self-host** | Free (AGPL-3.0) | Everything; bring your own GCP project + API keys |
| **Trial** | 14 days, no card | Full hosted plan |
| **Kosh Family — monthly** | ₹149/mo | Hosted, whole household + co-owner logins, family sharing, monthly AI allowance (e.g. 30 document parses, 200 questions) |
| **Kosh Family — annual** | ₹1,499/yr (~2 months free) | Same |

- One paid plan × two billing cycles at launch; add a "Plus" (more AI) tier
  only once real usage data exists. No free hosted tier — self-host + the demo
  are the free options.
- **Lapsed subscription** → app goes read-only; the Sheet remains theirs.
  Market this: "cancel anytime, your data never leaves your Drive."
- **Payments:** Razorpay Subscriptions for INR (UPI Autopay, cards,
  e-mandate). International later via a merchant of record (Paddle / Lemon
  Squeezy) so GST/VAT is handled for us.

## Licensing

- **Relicensed MIT → AGPL-3.0-only** (2026-11-07), before any hosted launch.
  AGPL §13 requires anyone running a *modified* Kosh as a network service to
  offer its source to users — so a third party can't take the code, close
  it, and compete with the hosted service. Self-hosting for your own family
  is unaffected.
- Versions already published under MIT stay MIT for whoever received them;
  AGPL applies from this point on.
- As sole copyright holder the author isn't bound by AGPL for their own hosted
  service — but keep the hosted build open anyway and link "Source" in the
  app footer (it's the trust story).
- **Before accepting outside contributions:** add a CLA (or DCO plus a
  relicensing clause), otherwise contributors' code would bind the hosted
  service under AGPL like anyone else.

## Phases

| # | Phase | Scope |
|---|---|---|
| **0** | Long-lead admin — start now | Business entity + Razorpay KYC; Google OAuth brand verification on the domain (takes weeks); Privacy Policy, Terms, Refund Policy, DPDP grievance contact (Razorpay and Google both require them live); talk to a CA about GST |
| **1** | Tenant-aware core | `TenantResolver` with self-host as the default implementation; batched reads; **no behaviour change** — ships to the current deploy as a pure refactor |
| **2** | Hosted onboarding | Any Google account signs up → `drive.file` consent → "create my Kosh sheet" or import an existing one via Picker; co-owner invites; share-token → tenant index in the control plane |
| **3** | Billing | Razorpay plans + checkout, webhook-driven subscription state, entitlement middleware, trial, read-only-on-lapse |
| **4** | Hosted extras | Account page (plan, invoices, cancel, delete account), per-tenant AI quotas, transactional email |
| **5** | Launch | Pricing page on the marketing site; docs split Hosted vs Self-host; a **staging environment** in infra (only production exists today — real payment webhooks shouldn't be tested there); monitoring with no financial data in logs |

The author's own family stays on the current self-hosted instance until the
hosted service is proven, then migrates as tenant #1.

## Risks

1. **Sheets quota at scale** — batched reads (Phase 1), short-lived caching,
   and a quota-increase request.
2. **Google verification timeline** — gates launch; hence Phase 0.
3. **Trust** — asking people to connect financial data. Lead with "it's your
   Sheet," open source, and the self-host option.

## Open questions

1. **Market:** India/INR-only at launch, or international from day one?
   (Razorpay vs merchant of record.)
2. **Entity:** registered business / GST number in place yet? (Sets the
   payments timeline.)
3. **AI:** included with an allowance (as above) or a separate add-on?

## Status
- Plan approved 2026-11-07. Licence moved to AGPL-3.0-only.
- Phase 1 (tenant-aware core) in progress.
