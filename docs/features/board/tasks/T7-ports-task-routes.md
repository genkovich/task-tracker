---
id: T7
title: "Add HTTP handlers for task CRUD + move, with rate limiting on create"
layer: "ports"
deps: ["T5"]
acs: ["AC-01", "AC-02", "AC-03", "AC-04", "AC-05", "AC-06"]
files_hint: ["api/internal/modules/board/ports/task_handler.go", "api/internal/modules/board/ports/dto.go", "api/internal/modules/board/ports/errors.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T7 — Add HTTP handlers for task CRUD + move, with rate limiting on create

## Why

[openapi.yaml](../contracts/openapi.yaml) fixes `POST /api/v1/tasks`, `PATCH/DELETE /api/v1/tasks/{taskId}`, `POST /api/v1/tasks/{taskId}/move` — all `security: []` (ADR-0001, no accounts). Spec §6.1 abuse case caps task creation at ≤30/min per client; sad.md §8 places this rate limiter as an in-process, module-local concern (no new infra).

## What

- `task_handler.go` — chi handlers for the four routes, calling `app.TaskService` (T5); `moveTask` reads the required `Idempotency-Key` header per the contract.
- `dto.go` — `TaskCreate`/`TaskUpdate`/`TaskMove`/`Task` request/response structs matching openapi.yaml's schemas exactly (`additionalProperties: false` semantics enforced by strict decoding).
- `errors.go` — `mapError` translating T2's sentinel errors to the exact `apperr.Error` codes in openapi.yaml (`task.title_required`, `task.not_found`, `board.column_not_found`, `task.rate_limited`).
- An in-process token-bucket limiter (per client IP) wrapping only the create-task handler, ≤30 requests/minute, returning 429 `task.rate_limited` past the limit.

## Definition of Done

- [ ] handler tests: `POST /tasks` → 201 (AC-01), 422 `task.title_required` on empty title (AC-02); `PATCH /tasks/{id}` → 200 (AC-03), 404 on unknown id, 422 on empty title; `DELETE /tasks/{id}` → 204 (AC-06), 404 on unknown id
- [ ] handler test: `POST /tasks/{id}/move` → 200 on a valid column (AC-04), 422 `board.column_not_found` on an invalid one (AC-05), 404 on unknown task
- [ ] handler test: the 31st create request within a minute from the same client returns 429 `task.rate_limited`
- [ ] every response body matches its openapi.yaml schema (status code + shape)
- [ ] lint + vet clean

## Notes

No `authMW` on this router (ADR-0001) — these routes are registered as public in T11, deliberately.
