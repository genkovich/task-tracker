---
id: T16
title: "End-to-end smoke test: full board lifecycle"
layer: "tests"
deps: ["T10", "T13", "T14", "T15"]
acs: ["AC-01", "AC-02", "AC-04", "AC-05", "AC-08", "AC-09", "AC-10", "AC-12", "AC-13"]
files_hint:
  - "web/e2e/tasks-board.smoke.spec.ts"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T16 — End-to-end smoke test: full board lifecycle

## Why

Proves the whole feature works end-to-end through the real UI, per [ux-flows.md](../ux-flows.md)'s flows.

## What

Write `web/e2e/tasks-board.smoke.spec.ts` (Playwright) driving the full lifecycle across two browser contexts (editor + viewer) per `ux-flows.md`.

## Definition of Done

- [ ] a Playwright smoke test drives: add a card, drag it, edit it, generate a public link, open it in a second context and see the card, disable the link, confirm the second context auto-navigates to not-found, delete the card
- [ ] lint + vet clean

## Notes

The widest task — needs the backend wired (T10) and all three editor UI pieces (T13, T14 depend transitively via T12) plus the viewer page (T15).
