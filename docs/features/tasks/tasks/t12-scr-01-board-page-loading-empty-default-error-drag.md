---
id: T12
title: "SCR-01 Board page — loading/empty/default/error + drag"
layer: "ui"
deps: []
acs: ["AC-01", "AC-10", "AC-11"]
files_hint:
  - "web/src/features/board-view/"
  - "web/src/shared/ui/skeleton.tsx"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T12 — SCR-01 Board page — loading/empty/default/error + drag

## Why

Builds the primary editor surface per [screens.md](../screens.md) SCR-01, the states not covered by T13/T14.

## What

Build the board page in `web/src/features/board-view/` (api/model/ui slice) — fetches `GET /api/v1/cards`, subscribes to `GET /api/v1/cards/events`, renders the 3-column layout, handles drag via the move endpoint with optimistic revert on failure. Add `web/src/shared/ui/skeleton.tsx` (the one NEW component screens.md flags).

## Definition of Done

- [ ] component tests cover the loading/empty/default/error states from screens.md SCR-01, plus a drag-and-drop test asserting a failed move visually reverts (AC-11)
- [ ] lint + vet clean

## Notes

Owns the one NEW component (`Skeleton`) screens.md flags — T13/T14 depend on this task existing first because they attach dialogs/controls to the board page shell it builds.
