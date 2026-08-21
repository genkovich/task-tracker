---
status: Living
tool: pencil
figma_file: ""
pen_file: "docs/design.pen"
updated_at: "2026-08-20"
---

# Design system — Task Tracker

> The project's **design canon**, produced once per repo by `design-system` and read by
> `ux-flows` / `screens` / `implement` / `review`. Committed — the tool choice and the inventory
> are team-wide, not per-developer. `architecture-map.md` §Frontend / UI foundation stays the
> inventory of the **code**; this file is the **design-side** canon (tool, posture, tokens,
> component inventory, cross-screen conventions). Refresh via `/sdd:design-system` when the
> foundation changes.

## Character & reference

Research pass: `threads.net` (live), sampled for **character**, not literal reuse. Signals
pulled from the rendered page: near-black surface `rgb(10,10,10)`, translucent white overlays
`rgba(255,255,255,0.08)` for hover/secondary surfaces, a compact `15px` base type size on a
system-sans stack, pill-shaped buttons, generous corner radius on cards/modals, hairline
dividers/borders. Translated into an **original** palette (not Threads' literal hex values):
monochrome-first neutrals + one confident brand accent (indigo/blue, absent from Threads —
needed here for kanban "in progress" semantics), plus 5 status colors the board requires that a
single-feed app doesn't. Two themes are designed — **Light is primary/default**, Dark mirrors it
token-for-token (see `docs/design.pen` → `Foundations — Light` / `Foundations — Dark`).

## Platform posture

- **Posture:** responsive-both — task board is used on desktop (primary, wide kanban) and tablet;
  no native mobile app in scope yet.
- **Breakpoints / device classes:** follows Tailwind defaults (`sm/md/lg/xl/2xl`) — not fixed
  further by this canon.

## Design tool

- **Tool:** pencil — chosen because the repo has no Figma file and the team works directly against
  code; Pencil's `.pen` variables mirror the Tailwind CSS custom properties 1:1.
- **Library location:** `docs/design.pen`

## Token source

- **Colors:** `web/src/app/styles/global.css` (`:root` + `.dark`, oklch) — mirrored as themed
  color variables in `docs/design.pen` (hex, light/dark via the `mode` theme axis).
- **Spacing / sizing:** Tailwind's default spacing scale (4px grid: `space-1`…`space-16`) — no
  override in `global.css`; documented as swatches in `docs/design.pen` → `Foundations` →
  "Spacing".
- **Typography:** `web/src/app/styles/global.css` `@theme` (`--font-sans: "Geist Variable"`,
  `--text-display`…`--text-caption`) — mirrored in `docs/design.pen` (`font-primary`, `text-*`
  number variables).

## Component inventory

| Component | Source (`file:line` / node / URL) | States it supports | Notes |
|---|---|---|---|
| Button | `web/src/shared/ui/button.tsx:7` | default / hover / focus / active / disabled / loading | 5 variants (primary/secondary/outline/ghost/destructive); pill radius (`radius-full`) per Threads character — see `docs/design.pen` → `Controls` → "Buttons" |
| Badge | `web/src/shared/ui/badge.tsx:7` | default | Extended with 5 status variants (Todo/In Progress/Done/Blocked/At Risk) via `--color-status-*` tokens — `docs/design.pen` → `Domain Components` → "Badges" |
| Avatar | `web/src/shared/ui/avatar.tsx:101` | default / fallback (initials) | Sizes 24/32/40/48; overlapping stack pattern for assignees — `docs/design.pen` → `Domain Components` → "Avatars" |
| Card | `web/src/shared/ui/card.tsx:79` | default | Base primitive; Task Card composes it with a status color bar |
| Input | `web/src/shared/ui/input.tsx:21` | default / focused / error / disabled | `docs/design.pen` → `Controls` → "Inputs" |
| Checkbox | NEW — not yet in `web/src/shared/ui` | unchecked / checked | Designed in `docs/design.pen` → `Controls`; register the code primitive here once built |
| Tabs | NEW — not yet in `web/src/shared/ui` | active / inactive | Pill tab-bar, `docs/design.pen` → `Controls` → "Tabs" |
| Board | NEW — design-only | default / empty | Kanban layout skeleton — `docs/design.pen` → `Domain Components` → "Board" (node `N9Ill`) |
| Column | NEW — design-only | default / empty | Header (dot + name + count + add) + card stack — same frame, node `Column/*` |
| Task Card | NEW — design-only | default / hover | Status color bar + badge + title + due date + assignee avatar — node `o8sjf` |

## Interaction & writing conventions

- **Errors:** inline notice card (icon + title + message), red `status-blocked`/`destructive`
  stroke on `$card` — see `docs/design.pen` → `States` → "Feedback States" → "Notice/Error".
- **Empty states:** centered icon + short title + one-line hint on a `$muted` panel — "Notice/Empty"
  in the same frame.
- **Loading:** skeleton blocks inside the existing card shape (not a spinner-only state) for
  content; inline spinner icon + label inside buttons for actions.
- **Validation:** on-blur for individual fields, red `$destructive` stroke + helper text under the
  input.
- **Microcopy tone:** short, confident, present tense — no exclamation marks, mirrors the compact
  15px body type (Threads-adjacent restraint).
