---
id: T13
title: "SCR-01 Add/edit card dialog"
layer: "ui"
deps: ["T12"]
acs: ["AC-02", "AC-03", "AC-13", "AC-14"]
files_hint:
  - "web/src/features/card-form/"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T13 — SCR-01 Add/edit card dialog

## Why

Builds the add/edit-card in-place states per [screens.md](../screens.md) SCR-01 and [ux-flows.md](../ux-flows.md) US-02/US-07.

## What

Build `web/src/features/card-form/` — a Dialog-based add/edit form (name + assignee inputs), calling create/update, with inline validation matching the 400 `tasks.name_required` response.

## Definition of Done

- [ ] component tests cover add-card and edit-card, both the default and the empty/whitespace-name validation state, from screens.md SCR-01
- [ ] lint + vet clean

## Notes

Shares `web/src/features/board-view/` only at the integration point (button that opens the dialog); the dialog's own files are a separate directory, so this stays in the T12 lane only for `implement`'s serialization, not a file overlap.
