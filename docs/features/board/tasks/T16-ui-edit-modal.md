---
id: T16
title: "Build the edit-task modal (title, assignee, save, delete)"
layer: "ui"
deps: ["T14"]
acs: ["AC-03", "AC-06"]
files_hint: ["web/src/features/board/ui/EditTaskModal.tsx"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T16 — Build the edit-task modal (title, assignee, save, delete)

## Why

[ux-flows.md](../ux-flows.md) SCR-03: "Модальне вікно: назва, виконавець, збереження, видалення", opened from a click on a card (SCR-01 → SCR-03). Delete lives inside this same form per ux-flows.md's screen-manifest note — no separate delete screen.

## What

`EditTaskModal.tsx` — reuses `shared/ui` Dialog + Input + Button:
- pre-filled title/assignee fields; Save calls T13's `editTask`, updates the card immediately on success (AC-03); empty title blocks save with the same inline error as T15's quick-add (AC-02, reapplied here per openapi.yaml's edit-endpoint 422).
- a Delete action calling T13's `deleteTask`, removing the card from the board on success (AC-06).

## Definition of Done

- [ ] component test: changing title/assignee and saving updates the card's displayed values without a full reload (AC-03)
- [ ] component test: clicking delete removes the task from the board (AC-06)
- [ ] component test: saving with an empty title shows the inline error, does not close the modal, does not call the API
- [ ] reuses `shared/ui` Dialog/Input/Button only — no new primitive
- [ ] `npm run typecheck` clean

## Notes

Shares `features/board/ui/` directory with T15 but a distinct file — no `files_hint` overlap, can run in parallel with T15.
