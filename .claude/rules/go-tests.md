---
paths: ["api/**/*_test.go"]
---

# Go testing conventions — api

<!-- Adapted from samber/cc-skills-golang@golang-testing v1.2.2 (upstream 466ea6d). RULE form (conventions only; procedures -> go-testing skill). Evals: .claude/evals/golang-testing/. -->

Tests are executable specifications: write them to constrain behavior, not to hit a coverage
number. This rule carries the CONVENTIONS; step-by-step test-writing procedures (scaffolding,
fuzzing setup, coverage workflows) live in the `go-testing` skill.

## MUST

- **Table-driven tests use named subtests.** Every case has a `name` field passed to `t.Run`. Subtest names are lowercase, descriptive (`"missing title"`, `"course not found"`).
- **Test observable behavior, not implementation details** — the public API / port contract, not unexported internals or call sequences.
- **Tests are order-independent.** Each test (and subtest) must pass when run alone (`go test -run TestX/sub`); no shared mutable global state between them.
- **Integration tests carry the build tag.** First line `//go:build integration`, package `<x>_test`. They never run under plain `make test`.
- **Assertions via testify, errors via `errors.Is`.** Use `require.ErrorIs(t, err, domain.ErrCourseNotFound)` for sentinels — never compare error strings.

## SHOULD

- **`require` to stop, `assert` to continue.** `require.NoError` / `require.Len` when later lines depend on the result; `assert.*` for independent field checks within one case.
- **`t.Helper()` in every helper** so failures report the caller's line (e.g. DB setup helpers).
- **`t.Parallel()` for independent tests** — but only when they share no mutable state; be deliberate, not reflexive.
- **Prefer hand-written fakes over a mock framework.** Define the fake against the consumer-side port; inject a fake `Clock` for time-dependent logic.
- **Keep unit tests fast (sub-millisecond)** and dependency-free; anything needing Postgres/Redis is an integration test behind the build tag.

## beer-lms specifics

- **Unit tests:** `package <x>_test`, testify `assert`/`require`, **hand-written fakes** — e.g. `fakeCourseRepo` (implements the `app` `CourseRepository` port) and `fakeClock{now}` (implements `Clock`) in `internal/modules/courses/app/service_test.go`. No mockgen/mockery.
- **Integration tests:** `//go:build integration`, then `dbtest.StartPostgres(ctx, t)` (`internal/platform/database/dbtest`) — testcontainers `postgres:18-alpine`, `t.Skip` when docker is absent, auto-terminate via `t.Cleanup`. Follow with `database.RunMigrations(migrations.FS, c.DSN)`; use `c.DB`. See `internal/modules/comments/infra/postgres_comment_repo_test.go` for the `setupDB` helper pattern.
- **Seed admin UUID** `019a0000-0000-7000-8000-000000000001` via `uuid.MustParse(...)` as the author/actor in repo integration tests. Redis-dependent tests use `miniredis`.
- **Commands (Makefile):** `make test` (`go test ./...`), `make test-integration` (`go test -tags integration -p 4 ./...`), plus `make lint` / `make vet`.

## Enforce / see also

`thelper`, `paralleltest`, `testifylint`, `tparallel` enforce much of this — see the `go-lint` skill.
For test-writing procedures and depth, the `go-testing` skill and upstream `references/` in `golang-testing`.
