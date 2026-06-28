# kosh-migrate
- **kosh-migrate** (`.claude/skills/kosh-migrate/SKILL.md`) - migrate existing financial data into Kosh Google Sheet format, tab by tab. Trigger: `/kosh-migrate`
When the user types `/kosh-migrate`, invoke the Skill tool with `skill: "kosh-migrate"` before doing anything else.

# Mobile-first rule
Every UI change must work on mobile (≤768px). The sidebar collapses to a bottom tab bar on mobile — new navigation actions must appear there too, not only in the sidebar. Modals and panels must be full-width on mobile (use `width: 100vw` via a CSS class + media query). Always check: fixed widths, desktop-only buttons (hidden via `display:none`), and overlays that assume wide viewports.
