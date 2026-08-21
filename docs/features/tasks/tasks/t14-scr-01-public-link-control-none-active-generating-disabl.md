---
id: T14
title: "SCR-01 Public-link control (none/active/generating/disabling)"
layer: "ui"
deps: ["T12"]
acs: ["AC-04", "AC-09"]
files_hint:
  - "web/src/features/public-link-control/"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T14 — SCR-01 Public-link control (none/active/generating/disabling)

## Why

Builds the public-link management states per [screens.md](../screens.md) SCR-01 and [ux-flows.md](../ux-flows.md) US-03/US-05.

## What

Build `web/src/features/public-link-control/` — renders none/active/generating/disabling, calling generate/disable, showing a copy-link affordance.

## Definition of Done

- [ ] component tests cover the none/active/generating/disabling states from screens.md SCR-01
- [ ] lint + vet clean

## Notes

None.
