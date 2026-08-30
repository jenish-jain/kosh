# kosh-migrate
- **kosh-migrate** (`.claude/skills/kosh-migrate/SKILL.md`) - migrate existing financial data into Kosh Google Sheet format, tab by tab. Trigger: `/kosh-migrate`
When the user types `/kosh-migrate`, invoke the Skill tool with `skill: "kosh-migrate"` before doing anything else.

# Mobile-first rule
Every UI change must work on mobile (≤768px). The sidebar collapses to a bottom tab bar on mobile — new navigation actions must appear there too, not only in the sidebar. Modals and panels must be full-width on mobile (use `width: 100vw` via a CSS class + media query). Always check: fixed widths, desktop-only buttons (hidden via `display:none`), and overlays that assume wide viewports.

# Docs-stay-current rule
Any feature that changes setup steps, deployment, self-hosting requirements, or the overall feature set must update documentation in the same change — not as a follow-up. Check both: `README.md` (quick start / overview) and the public docs page (`frontend/src/screens/marketing/Docs.jsx`, plus `docs/SELF_HOSTING.md` if the full reference guide is affected). Keep README lean — link out to `docs/SELF_HOSTING.md` for detail rather than duplicating it, and don't let stale coverage numbers, generated reports, or other one-off writeups accumulate in it.

# Git workflow rule
Commit as you go, not in one dump at the end: each commit should be small and meaningful — one logical change, complete and working on its own, with a message explaining why. Don't bundle unrelated changes (e.g. a feature tweak and a docs cleanup) into the same commit just because they happened in the same session. Once a feature is implemented, verified, and you're confident in it, open a PR proactively — don't ask for permission first. This doesn't relax the existing rule against pushing directly to `master` or force-pushing; PRs still go through normal review.
