---
id: T3
title: "Implement Postgres repository for board, columns, tasks, public_links"
layer: "infra"
deps: ["T1", "T2"]
acs: ["AC-01", "AC-04", "AC-05", "AC-06"]
files_hint: ["api/internal/modules/board/ports/repo.go", "api/internal/modules/board/infra/postgres_repo.go"]
owner: "genkovich"
estimate: "L"
status: "todo"
---

# T3 — Implement Postgres repository for board, columns, tasks, public_links

## Why

[sad.md §5](../sad.md) building-block view: `infra/` holds the pgx repo behind a `ports/` interface, following the repo's `domain/app/ports/infra` convention (`CLAUDE.md` §Architecture). [data-model.md](../data-model.md) fixes the queries and indexes this repo must serve.

## What

- `api/internal/modules/board/ports/repo.go` — the `Repository` interface `app` (T5/T6) depends on: `GetBoardState`, `LeftmostColumnID`, `InsertTask`, `UpdateTask`, `MoveTask`, `DeleteTask`, `ColumnExists`, `IssuePublicLink`, `RevokePublicLink`, `PublicLinkByToken`.
- `api/internal/modules/board/infra/postgres_repo.go` — pgx/v5 implementation using `$N` params, the app-generated UUIDv7 convention (`google/uuid`), and the indexes from data-model.md (`idx_columns_board_id_position`, `idx_tasks_column_id`, the two `public_links` UNIQUEs).

## Definition of Done

- [ ] integration test (testcontainers, `-tags integration`): inserting a task without an explicit column lands in the leftmost column by `position` (AC-01)
- [ ] integration test: `MoveTask` updates `column_id` for a valid target column (AC-04)
- [ ] integration test: moving to a non-existent `column_id` returns the domain not-found error, no row changed (AC-05)
- [ ] integration test: `DeleteTask` hard-deletes the row (AC-06) — matches the `user` module's existing hard-delete pattern (data-model.md `public_links` note)
- [ ] `go test ./internal/modules/board/... -tags integration` green
- [ ] lint + vet clean

## Notes

`MoveTask` must stay a plain single-row `UPDATE tasks SET column_id = $1 WHERE id = $2` — no version/lock column (data-model.md `tasks` access-pattern note; last-write-wins is the intended behavior, verified by T12, not prevented here).
