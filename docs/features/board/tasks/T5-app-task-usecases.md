---
id: T5
title: "Implement task use-cases: create, edit, move, delete"
layer: "app"
deps: ["T2", "T3"]
acs: ["AC-01", "AC-02", "AC-03", "AC-04", "AC-05", "AC-06"]
files_hint: ["api/internal/modules/board/app/task_service.go"]
owner: "genkovich"
estimate: "L"
status: "todo"
---

# T5 — Implement task use-cases: create, edit, move, delete

## Why

[sad.md §5](../sad.md) building-block view names `BoardService: CreateTask/EditTask/MoveTask/DeleteTask/...` as the `app` layer's responsibility, depending on the `ports.Repository` interface (T3), not a concrete adapter (`CLAUDE.md` §Architecture).

## What

`api/internal/modules/board/app/task_service.go` — a service depending on `ports.Repository` (T3) and `ports.Broadcaster` (T4):
- `CreateTask(title, assignee)` — inserts into the leftmost column (AC-01), rejects empty title via the T2 domain constructor (AC-02).
- `EditTask(id, title, assignee)` — updates title/assignee (AC-03), same empty-title rejection.
- `MoveTask(id, columnID)` — validates the target column exists before writing (AC-05); a single unconditional `UPDATE` otherwise — no lock/version check (AC-04, last-write-wins by design).
- `DeleteTask(id)` — hard delete (AC-06).
- Every successful mutation calls `Broadcaster.Broadcast()` after the write commits (ADR-0002).

## Definition of Done

- [ ] unit tests (repo faked via the `ports.Repository` interface) cover: create lands in leftmost column (AC-01), empty title rejected (AC-02), edit updates title/assignee (AC-03), move to a valid column succeeds (AC-04), move to an invalid column is rejected with no write (AC-05), delete removes the task (AC-06)
- [ ] each success path calls the fake broadcaster exactly once
- [ ] lint + vet clean

## Notes

Do not add a version/lock field or a compare-and-swap on move — that would contradict the last-write-wins invariant spec §6 NFR and AC-05b require; T12 tests that this task's `MoveTask` behaves correctly under concurrency exactly because it stays a plain unconditional write.
