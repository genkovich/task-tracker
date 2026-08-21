---
id: T11
title: "Backend integration tests: concurrency + end-to-end flows"
layer: "tests"
deps: ["T10"]
acs: ["AC-07", "AC-15"]
files_hint:
  - "api/internal/modules/tasks/ports/tasks_integration_test.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T11 — Backend integration tests: concurrency + end-to-end flows

## Why

Proves the two domain invariants (AC-07, AC-15) hold against a real database, not just a fake repository; derives from [sad §10 QG-1](../sad.md).

## What

Write `api/internal/modules/tasks/ports/tasks_integration_test.go` (testcontainers) covering the full CRUD+move+link lifecycle plus the two concurrency races from sad.md §10 QG-1.

## Definition of Done

- [ ] testcontainers-based test fires two near-simultaneous moves on the same card and asserts the last-processed one wins; a second test fires a delete racing a move and asserts the delete wins
- [ ] lint + vet clean

## Notes

None.
