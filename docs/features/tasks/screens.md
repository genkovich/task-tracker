---
status: draft
feature_size: "S"
tool: "code"
updated_at: "2026-08-21"
---

# Screens — tasks

> The canonical **screen manifest** — every screen in every state — produced by `screens` (between
> `api` and `tasks`) and read by `tasks` (each `ui` task cites SCR ids + states), `implement`
> (builds the screen to the declared states) and `review` (the built screen must match this).
> Downstream stages reference **only this manifest** — never the raw Figma / `.pen` file.

## Source

- **Tool:** `code` — **degradation, named**: `docs/design-system.md` does not exist yet in this
  repo (no design canon has been established — recommended in the handoff below). Component names
  in this manifest are drawn from the **actual existing** `web/src/shared/ui/` primitives (shadcn,
  per `web/CLAUDE.md`) as the closest available reuse baseline, not an invented inventory.
- **File:** inline wireframes below (no Figma/`.pen` file — `code` mode).

## Screens

### SCR-01 — Board (editor)

| State | Trigger / condition | Components (from `web/src/shared/ui/`) | Source-ref |
|---|---|---|---|
| loading | Initial `GET /api/v1/cards` in flight, page just opened | NEW: `Skeleton` (no loading-placeholder primitive exists yet) | wireframe below |
| empty | Board loaded, zero cards in any column (spec §7 KPI: time to first card) | `EmptyState`, `Button` (Add card) | wireframe below |
| default | Board loaded, ≥1 card, cards laid out in their 3 columns | `Card` (one per task card), `Badge` (assignee, when present), `Button` (Add card, per-column) | wireframe below |
| error | `GET /api/v1/cards` fails (network/server) | `sonner` (toast) — non-blocking, board area keeps its last-known state if any | wireframe below |
| add-card · default | Team member opens the add-card action (AC-02) | `Dialog`, `Input` (name), `Input` (assignee, optional), `Button` (Save) | wireframe below |
| add-card · validation | Save attempted with empty/whitespace-only name (AC-03) | same + inline error text under the name `Input` | wireframe below |
| edit-card · default | Team member opens an existing card to edit (AC-13) | `Dialog`, `Input` (name), `Input` (assignee), `Button` (Save) | wireframe below |
| edit-card · validation | Save attempted with empty/whitespace-only name (AC-14) | same + inline error text under the name `Input` | wireframe below |
| drag · in-flight | Card being dragged, move request sent (AC-01) | `Card` (dragging visual state — CSS only, no new component) | wireframe below |
| drag · failed | Move request fails, e.g. connection lost (AC-11) | `Card` (snaps back to prior column via existing state), `sonner` (toast: save failed) | wireframe below |
| public-link · none | No active public link (AC-09 precondition) | `Button` (Get link) | wireframe below |
| public-link · active | A public link is active (AC-09) | `Badge` or `Input readOnly` (shows the link), `Button` (Copy), `Button` (Disable) | wireframe below |
| public-link · generating / disabling | `POST /public-links` or `.../disable` in flight | `Button` (disabled + built-in loading affordance, no new component) | wireframe below |

```text
+---------------------------------------------------------------+
| Task Tracker                          [ Get link ]  (no link) |
+---------------------------------------------------------------+
| To Do            | In Progress        | Done                  |
| [+ Add card]      | [+ Add card]        | [+ Add card]          |
| +---------------+ | +---------------+  |                       |
| | Card name     | | | Card name     |  |                       |
| | @assignee     | | | @assignee     |  |                       |
| +---------------+ | +---------------+  |                       |
+---------------------------------------------------------------+
  ^ default (populated)                    ^ empty column is just no cards, not a
                                              distinct screen state — the board-level
                                              `empty` state above is for ALL columns empty
```

### SCR-02 — Public board (viewer)

| State | Trigger / condition | Components | Source-ref |
|---|---|---|---|
| loading | Initial `GET /api/v1/public/boards/{token}` in flight | NEW: `Skeleton` (same as SCR-01) | wireframe below |
| empty | Link active, board has zero cards | `EmptyState` | wireframe below |
| default | Link active, board loaded, live-updating via SSE (AC-08, AC-12) | `Card` (read-only variant — no drag handle, no edit/delete affordance, per AC-06), a persistent "View only" label | wireframe below |
| error | An already-open viewer's SSE connection or a background refetch fails transiently (not the 404 case — that's SCR-03) | `sonner` (toast: connection issue, board may be stale) | wireframe below |

```text
+---------------------------------------------------------------+
| Task Tracker · View only                                      |
+---------------------------------------------------------------+
| To Do            | In Progress        | Done                  |
| +---------------+ | +---------------+  |                       |
| | Card name     | | | Card name     |  |                       |
| | @assignee     | | | @assignee     |  |                       |
| +---------------+ | +---------------+  |                       |
+---------------------------------------------------------------+
  ^ no "+Add card", no drag handles, no edit/delete controls anywhere (AC-06)
```

### SCR-03 — Not found (viewer)

| State | Trigger / condition | Components | Source-ref |
|---|---|---|---|
| default | Public link disabled, or never valid (AC-05), or an active viewer's link gets disabled mid-view (AC-12) | **Reuses the repo's existing `pages/not-found/ui/NotFoundPage.tsx`** — the same component the app's catch-all route (`route("*", ...)` in `routes.ts`) already renders for any unmatched address | `web/src/pages/not-found/ui/NotFoundPage.tsx` (existing file, no new work) |

No wireframe needed — this is a direct reuse of the existing page, which is exactly what AC-05
requires: rendering the *identical* not-found experience for a disabled/never-valid link as for any
other bad address makes the indistinguishability a structural property, not something to
separately verify. AC-12 (auto-transition while viewing) is a client-side navigation event
(SSE `link.disabled` / stream close → `navigate()` to this same route), not a new screen state.

## New components

| Component | Why no existing primitive fits | Registered in design-system |
|---|---|---|
| `Skeleton` | No loading-placeholder primitive exists anywhere in `web/src/shared/ui/` today (checked: no `skeleton`/`spinner`/`loader` file) — every other loading affordance in the repo relies on this being built once and reused | pending |

All other components on every screen above reuse the existing `web/src/shared/ui/` inventory
(`Card`, `Badge`, `Button`, `Dialog`, `Input`, `EmptyState`, `sonner`) or, for SCR-03, the existing
`NotFoundPage` route outright — no other new primitive is introduced by this feature.
