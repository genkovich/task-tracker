---
id: T5
title: "Card service (create/update/move/delete/get-board)"
layer: "app"
deps: ["T2"]
acs: ["AC-01", "AC-02", "AC-07", "AC-10", "AC-11", "AC-13", "AC-14", "AC-15"]
files_hint:
  - "api/internal/modules/tasks/app/card_service.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T5 — Card service (create/update/move/delete/get-board)

## Why

The card business logic, including ADR-0002's server-timestamp last-write-wins overwrite; derives from [sad §4](../sad.md) and [ADR-0002](../adr/0002-server-timestamp-last-write-wins.md).

## What

Implement the card service in `api/internal/modules/tasks/app/` — CreateCard, UpdateCard, MoveCard, DeleteCard, GetBoard — each write unconditionally overwrites and bumps `updated_at` server-side (ADR-0002), no version check.

## Definition of Done

- [ ] unit tests (against a fake CardRepository) cover last-write-wins overwrite semantics (ADR-0002) and the delete-wins-over-concurrent-move race (AC-15)
- [ ] lint + vet clean

## Notes

Can start as soon as T2 lands, in parallel with T3/T4 — unit-tested against a fake repository; only the T11 integration test needs the real T3 implementation.
