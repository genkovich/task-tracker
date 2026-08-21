---
id: T12
title: "Write integration tests for concurrent move and link-revocation invariants"
layer: "tests"
deps: ["T3", "T5", "T9"]
acs: ["AC-05b", "AC-11"]
files_hint: ["api/internal/modules/board/app/board_service_integration_test.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T12 — Write integration tests for concurrent move and link-revocation invariants

## Why

[sad.md §10 QG-2](../sad.md) explicitly requires this as a domain-invariant integration test against a real DB, not a unit test with a faked repo: "конкурентний запис двох команд move у тестовому середовищі, перевірка фінального стану в БД". Spec AC-05b is the concurrent-edge acceptance criterion; sad.md §11 separately flags that SSE close-on-revoke (AC-11) has no dedicated test anywhere upstream of this task.

## What

Two integration tests (`-tags integration`, real Postgres via testcontainers, matching `api/internal/modules/user/ports/handler_integration_test.go`'s existing pattern):

1. **Concurrent move (AC-05b):** fire two goroutines calling `TaskService.MoveTask` for the same task id, targeting two different valid columns, near-simultaneously; assert the task's final `column_id` in the DB matches whichever call's write landed last, and every viewer's subsequent `GetBoardState`/`GetPublicBoardState` read returns that same single column — no torn state.
2. **Revoke closes live connections (AC-11):** open a public SSE connection on an active token (via T9's handler in a test server), revoke the link (T6/T8), and assert the connection closes synchronously rather than merely rejecting new subscribe attempts.

## Definition of Done

- [ ] `go test ./internal/modules/board/... -tags integration -run TestConcurrentMove` passes reliably across repeated runs (no flake from ordering assumptions — assert "exactly one final column", not which one)
- [ ] `go test ./internal/modules/board/... -tags integration -run TestRevokeClosesConnections` passes
- [ ] both included in `make -C api test-integration`

## Notes

This task exists specifically because sad.md §11 named live-push behavior as under-tested by any upstream artifact — treat both tests as closing that gap, not as optional extras.
