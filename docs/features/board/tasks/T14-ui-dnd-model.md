---
id: T14
title: "Build drag-and-drop board state with optimistic updates"
layer: "ui"
deps: ["T13"]
acs: ["AC-04", "AC-05"]
files_hint: ["web/src/features/board/model/"]
owner: "genkovich"
estimate: "L"
status: "todo"
---

# T14 — Build drag-and-drop board state with optimistic updates

## Why

[sad.md §5](../sad.md) building-block view: `features/board/model/` owns "drag-and-drop state, optimistic UI for create/edit/move/delete". [ux-flows.md](../ux-flows.md) US-03 flow covers both the valid-drop (AC-04) and invalid-drop (AC-05) branches, and sad.md §11 flags touch/mobile drag as a workshop-audience risk.

## What

Board state (columns + tasks) held in `features/board/model/`, exposing:
- a drop handler that, on a valid-column drop, optimistically moves the task locally and calls T13's `moveTask` (AC-04), rolling back on API failure;
- a no-op on drop outside any valid column — no API call, task stays put (AC-05);
- equivalent optimistic-then-confirm handling for create/edit/delete, reconciled against SSE-triggered refetches from T13.

## Definition of Done

- [ ] test: dropping on a valid column moves the task in local state immediately and calls `moveTask` once (AC-04)
- [ ] test: dropping outside any column leaves local state unchanged and calls no API (AC-05)
- [ ] test: a failed `moveTask` call reverts the optimistic move
- [ ] drag interactions work via both mouse and touch pointer events (manual check on a real mobile device before `ship`, per sad.md §11 risk — this task covers the code path, not the device verification)

## Notes

This is the highest-risk UI task per sad.md §11 ("Перетягування може не працювати коректно на сенсорних екранах") — budget real device testing time beyond the unit-test DoD above.
