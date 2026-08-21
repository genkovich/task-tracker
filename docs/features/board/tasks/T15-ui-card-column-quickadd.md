---
id: T15
title: "Build task card, column, and quick-add UI components"
layer: "ui"
deps: ["T14"]
acs: ["AC-01", "AC-02"]
files_hint: [
  "web/src/features/board/ui/TaskCard.tsx",
  "web/src/features/board/ui/Column.tsx",
  "web/src/features/board/ui/QuickAddTask.tsx"
]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T15 — Build task card, column, and quick-add UI components

## Why

[ux-flows.md](../ux-flows.md) SCR-01 (board) and SCR-02 (inline quick-add) name these three components. `architecture-map.md` §Frontend inventory (Card, Button, Input) is the reuse baseline — `CLAUDE.md`: "compose over [shadcn primitives], don't hand-edit the generated primitives".

## What

- `TaskCard.tsx` — title, assignee, draggable (wired to T14's model); reuses `shared/ui` Card.
- `Column.tsx` — name + ordered task list (`position`, AC-01 "найлівіша column" is column 0); reuses Card as the column container.
- `QuickAddTask.tsx` — inline title input in the leftmost column (SCR-02); reuses `shared/ui` Input + Button; on submit, empty title shows the inline required-name error and does not call `createTask` (AC-02); non-empty title creates and shows the task immediately (AC-01).

## Definition of Done

- [ ] component test: submitting quick-add with a non-empty title shows the new task in the leftmost column without a page reload (AC-01)
- [ ] component test: submitting with an empty title shows an inline "назва обов'язкова" error and makes no API call (AC-02)
- [ ] no new primitive introduced — verified against `web/src/shared/ui/` inventory before writing any new component
- [ ] `npm run typecheck` clean

## Notes

`Column.tsx` renders whatever fixed set `GET /api/v1/board` returns (ADR-0004) — no add/rename/delete-column affordance anywhere in this component.
