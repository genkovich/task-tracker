---
name: go-database
description: "Go database access for api with pgx/v5 + pgxpool — the database.DB/Tx wrapper, parameterized $1 queries, manual rows.Scan, NULLable pointer fields, JSONB as []byte+json.Unmarshal, transactions and SELECT FOR UPDATE, pgx.ErrNoRows→domain sentinel, constraint mapping via pgerrcode (IsPgUniqueViolation/IsPgForeignKeyViolation), pgxpool sizing, and context propagation. Use when writing, reviewing, or debugging repository code under internal/modules/*/infra that talks to PostgreSQL through pgx. Does NOT generate database schemas or migration SQL (that is data-model / migrations)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents. Targets api (Go 1.25, pgx/v5).
metadata:
  author: samber
  version: "1.2.1"
  openclaw:
    emoji: "🗄"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-database v1.2.1 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-database/. -->

**Persona:** You are a Go backend engineer who writes safe, explicit, observable database code. You treat SQL as a first-class language — no ORMs, no magic — and you catch data-integrity issues at the boundary, not deep in the application.

**Modes:**

- **Write mode** — generating a new repository function, query helper, or transaction wrapper: grep the sibling `infra/Postgres*Repository` files first to match the existing query and scan style, then write.
- **Review/debug mode** — auditing or debugging existing database code: scan for missing `rows.Close()`, un-parameterized queries, dropped `ctx`, and absent `rows.Err()` checks.

> beer-lms is **pgx-native.** `database/sql`, `sqlx`, `go-sqlmock`, and ORMs (GORM/ent) are **NOT used here** — ignore upstream `database/sql`/`sqlx` snippets in the references unless explicitly noted. Read the pgx docs for current API signatures.

## In beer-lms

All DB access goes through `internal/platform/database.DB`, a thin wrapper over `pgxpool.Pool`. There is no global DB; the pool is created once in `database.New(ctx, dsn)` and injected (manual DI) into each `Postgres<X>Repository`.

```go
type DB struct { pool *pgxpool.Pool }        // internal/platform/database
func New(ctx, dsn) (*DB, error)              // pgxpool; AfterConnect registers pgx-shopspring-decimal
func (db *DB) Query/QueryRow/Exec(ctx, sql, args...) ...   // ctx is always first
func (db *DB) Begin(ctx) (*Tx, error)
type Tx struct { tx pgx.Tx }                 // QueryRow/Query/Exec/Commit/Rollback
```

Grounding rules for repository code (`internal/modules/<domain>/infra/postgres_<x>_repository.go`):

- **Constructor + receiver:** `func NewPostgres<X>Repository(db *database.DB) *Postgres<X>Repository`, receiver `(r *Postgres<X>Repository)`. The consumer-side port `<X>Repository` is declared in the module's `app/ports.go` — infra satisfies it, infra does not define it (see `go-structs-interfaces` rule).
- **Parameterized only.** `$1, $2, …` placeholders; never `fmt.Sprintf` user input into SQL.
- **`ctx` first, threaded through.** pgx methods already take `ctx` as the first arg — pass the inbound `r.Context()` chain down; never mint `context.Background()` mid-stack (see `go-context` rule).
- **Not-found → domain sentinel.** `if errors.Is(err, pgx.ErrNoRows) { return nil, domain.ErrCourseNotFound }`; otherwise wrap: `return nil, fmt.Errorf("get course: %w", err)`.
- **Constraint violations → domain conflict.** Use the helpers in `internal/platform/database/errors.go` — `database.IsPgUniqueViolation(err)` (23505) and `database.IsPgForeignKeyViolation(err)` (23503), both built on `*pgconn.PgError` + `pgerrcode`. Map their result to a domain sentinel; don't re-implement the `errors.As` + code check inline.
- **Manual scanning.** This repo scans by hand — `QueryRow(ctx, …).Scan(&a, &b, …)` for one row, `rows.Scan(…)` in a `for rows.Next()` loop for many. It does **not** use `pgx.RowToStructByName` / `pgx.CollectRows`. NULLable columns use pointer fields (`*string`, `*time.Time`).
- **JSONB as `[]byte` + `json.Unmarshal`.** Scan a `jsonb` column into a `[]byte`, then `json.Unmarshal` into the typed Go value (pattern in `internal/modules/completions/app/peer_blob.go`).
- **`shopspring/decimal` for money/points.** `database.New` registers `pgxdecimal.Register(conn.TypeMap())` in `AfterConnect`, so `decimal.Decimal` columns scan natively — never round-trip them through `float64` (see `go-safety` rule).
- **App-side UUID v7.** IDs are generated in Go with `uuid.Must(uuid.NewV7())` (`google/uuid`) before insert — not a DB `DEFAULT`.
- **Rows discipline.** `defer rows.Close()` immediately after a `Query`, and always check `rows.Err()` after the loop.
- **Schema/migrations are not this skill.** Migration SQL is human-authored and applied via `database.RunMigrations(migrations.FS, dsn)` (golang-migrate from an embedded FS). Designing schema + writing `*.up.sql`/`*.down.sql` is the `data-model` skill's job — never AI-generate schema here.

## Best Practices Summary

1. Queries MUST use parameterized placeholders — NEVER concatenate user input into SQL strings.
2. `ctx` MUST flow into every DB call — pgx's `Query/QueryRow/Exec` take it as the first argument; pass the request's ctx, don't substitute `Background()`.
3. `pgx.ErrNoRows` MUST be handled explicitly with `errors.Is` — translate "not found" to a domain sentinel; never leak it upward.
4. Rows MUST be closed after iteration — `defer rows.Close()` right after `Query`, and check `rows.Err()` when the loop ends.
5. NEVER use `Query` for a statement that returns no rows — use `Exec`. `Query` hands back rows you must close or the conn leaks back to the pool.
6. Use a transaction for multi-statement writes — `db.Begin(ctx)` → work on `*Tx` → `Commit`; on any early return the deferred `Rollback` fires.
7. Use `SELECT … FOR UPDATE` when you read a row you intend to modify in the same tx — prevents lost-update races.
8. Set a stronger isolation level (e.g. serializable) only when READ COMMITTED is genuinely insufficient; pgx exposes it via `pgx.TxOptions` on `BeginTx`.
9. Handle NULLable columns with pointer fields (`*string`, `*time.Time`) — they scan cleanly and JSON-marshal correctly.
10. Map constraint violations to a domain conflict via `database.IsPgUniqueViolation` / `IsPgForeignKeyViolation` — don't surface a raw pg error.
11. Batch large writes in reasonable chunks — not row-by-row (round-trip storm), not millions at once (locks + memory). pgx offers `Batch` / `CopyFrom` for bulk paths.
12. NEVER create or modify schema from application code — schema design needs data-volume and access-pattern context AI doesn't have. Migrations are external + human-reviewed.
13. Avoid hidden SQL features (triggers, views, materialized views, stored procedures, row-level security) in application logic — they create invisible side effects.

## Parameterized Queries

```go
// ✗ VERY BAD — SQL injection
q := fmt.Sprintf("SELECT * FROM courses WHERE org_id = '%s'", orgID)

// ✓ Good — parameterized (PostgreSQL / pgx)
var c domain.Course
err := r.db.QueryRow(ctx,
    "SELECT id, org_id, title, status FROM courses WHERE id = $1 AND org_id = $2",
    id, orgID,
).Scan(&c.ID, &c.OrgID, &c.Title, &c.Status)
```

### Dynamic IN clauses

pgx supports passing a slice for an array parameter — no manual placeholder expansion:

```go
rows, err := r.db.Query(ctx,
    "SELECT id, title FROM courses WHERE id = ANY($1)", ids, // ids []uuid.UUID
)
```

### Dynamic column names

Never interpolate a column or sort key from user input. Use an allowlist:

```go
allowed := map[string]bool{"title": true, "created_at": true}
if !allowed[sortCol] {
    return nil, fmt.Errorf("invalid sort column: %s", sortCol)
}
q := fmt.Sprintf("SELECT id, title FROM courses ORDER BY %s", sortCol) // sortCol is now safe
```

For injection-prevention patterns see the `go-security` rule.

## Struct Scanning and NULLable Columns

Scan manually into struct fields; use pointer fields for NULLable columns; scan `jsonb` into `[]byte` then `json.Unmarshal`. See [Scanning Reference](./references/scanning.md) — its sqlx / `pgx.CollectRows` sections are marked not-used-here; the pointer-field and JSON-marshaling guidance applies directly.

## Error Handling

```go
func (r *PostgresCourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
    var c domain.Course
    err := r.db.QueryRow(ctx,
        "SELECT id, org_id, title, status FROM courses WHERE id = $1",
        id,
    ).Scan(&c.ID, &c.OrgID, &c.Title, &c.Status)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrCourseNotFound // translate to domain sentinel
        }
        return nil, fmt.Errorf("get course %s: %w", id, err)
    }
    return &c, nil
}
```

### Always close rows

```go
rows, err := r.db.Query(ctx, "SELECT id, title FROM courses WHERE org_id = $1", orgID)
if err != nil {
    return nil, fmt.Errorf("list courses: %w", err)
}
defer rows.Close() // prevents connection leaks

courses := make([]domain.Course, 0) // never leave nil if it gets JSON-encoded
for rows.Next() {
    var c domain.Course
    if err := rows.Scan(&c.ID, &c.Title); err != nil {
        return nil, fmt.Errorf("scan course: %w", err)
    }
    courses = append(courses, c)
}
if err := rows.Err(); err != nil { // always check after iteration
    return nil, fmt.Errorf("iterate courses: %w", err)
}
```

### Common database error patterns

| Error | How to detect | Action |
| --- | --- | --- |
| Row not found | `errors.Is(err, pgx.ErrNoRows)` | Return domain sentinel (`domain.Err…NotFound`) |
| Unique constraint (23505) | `database.IsPgUniqueViolation(err)` | Return domain conflict sentinel |
| Foreign-key violation (23503) | `database.IsPgForeignKeyViolation(err)` | Return domain validation/conflict sentinel |
| Connection refused | `err != nil` on the first call after `New` | Fail fast, log once at the boundary, retry with backoff |
| Serialization failure (40001) | `*pgconn.PgError` code `40001` | Retry the whole transaction |
| Context canceled | `errors.Is(err, context.Canceled)` | Stop, propagate |

See the `go-errors` rule for the full domain-sentinel → `apperr.Error` → wire flow.

## Context Propagation

pgx requires a context on every call — there is no context-free variant to misuse:

```go
// ✓ respects cancellation and the request deadline
r.db.Query(ctx, "SELECT …")
```

Thread the same `ctx` from the chi handler through the service into the repository; never swap in a fresh `Background()`. See the `go-context` rule.

## Transactions, Isolation, and Locking

The house transaction pattern (cite `internal/modules/comments/infra/postgres_comment_repo.go:100`–`148`):

```go
tx, err := r.db.Begin(ctx)
if err != nil {
    return fmt.Errorf("begin tx: %w", err)
}
defer tx.Rollback(ctx) //nolint:errcheck // no-op once Commit succeeds; ignore rollback error on the happy path

// ... multiple tx.Exec / tx.QueryRow / tx.Query calls ...

if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit tx: %w", err)
}
return nil
```

- `defer tx.Rollback(ctx)` directly after a successful `Begin` is the safety net: any early `return` rolls back; after `Commit` the rollback is a no-op. The `//nolint:errcheck` is the repo's settled directive for exactly this line.
- Use `FOR UPDATE` when you read-then-write the same row inside the tx. For `FOR UPDATE SKIP LOCKED` (concurrent claim across instances) see the outbox relay (`internal/platform/outbox`) and the `go-concurrency` rule.
- For custom isolation, pass `pgx.TxOptions{IsoLevel: pgx.Serializable}` to `BeginTx` (the platform wrapper exposes `Begin` with defaults; reach for raw `pool.BeginTx` only when you genuinely need a non-default level).

For locking variants and isolation trade-offs see [Transactions](./references/transactions.md) (its `BeginTxx`/`sql.TxOptions` snippets are `database/sql`; translate to `pgx.TxOptions`).

## Connection Pool

The pool is configured in `database.New` via the pgxpool config / DSN — e.g. `pool_max_conns` in the connection string — not via `database/sql` setters like `SetMaxOpenConns` (those don't exist on a `*pgxpool.Pool`). Size it to your Postgres `max_connections` budget across all instances. For sizing guidance see [Database Performance](./references/performance.md).

## Migrations

Migration SQL is **external and human-reviewed**, applied through `database.RunMigrations(migrations.FS, dsn)` (golang-migrate/v4 from the embedded `migrations` FS) and `make migrate-go-up` / the `cmd/migrate` binary. Designing the schema and writing the paired `*.up.sql` / `*.down.sql` files is the `data-model` skill — this skill never AI-generates schema.

## Testing Database Code

- **Unit tests** of the service layer inject a **hand-written fake** repository (the repo does not use `go-sqlmock` or `testify/mock`) — see the `go-testing` skill and the `fakeCourseRepo` example.
- **Integration tests** run against a real PostgreSQL via testcontainers: `dbtest.StartPostgres(ctx, t)` (image `postgres:18-alpine`) + `database.RunMigrations(migrations.FS, c.DSN)`, gated by `//go:build integration` and run with `make test-integration`. See [Testing Database Code](./references/testing.md) (its sqlx/sqlmock/suite snippets are not the house style — the build-tag + testcontainers shape is) and the `go-testing` skill for the canonical recipe.

## Avoid Hidden SQL Features

Do not rely on triggers, views, materialized views, stored procedures, or row-level security in application logic — they create invisible side effects and make debugging impossible. Keep SQL explicit and visible in Go where it can be tested and version-controlled.

## Schema Creation

**This skill does NOT cover schema creation.** AI-generated schemas are often subtly wrong — missing indexes, wrong column types, bad normalization. Schema design requires understanding data volumes, access patterns, and constraints. Use the `data-model` skill + human review.

## Deep Dives

- **[Transactions](./references/transactions.md)** — boundaries, isolation levels, deadlock prevention, `SELECT FOR UPDATE` (translate `database/sql` snippets to `pgx.TxOptions`).
- **[Testing Database Code](./references/testing.md)** — integration tests with containers, fixtures, schema setup/teardown (use the beer-lms `dbtest` recipe, not sqlx/suite).
- **[Database Performance](./references/performance.md)** — pool sizing, batch processing, indexing strategy, query optimization.
- **[Struct Scanning](./references/scanning.md)** — pointer fields for NULLs, JSON marshaling (skip the sqlx / `CollectRows` sections).

## Cross-References

- → `go-errors` rule — domain sentinel → `apperr.Error` mapping and `%w` wrapping for DB errors.
- → `go-context` rule — propagating `ctx` into every pgx call.
- → `go-safety` rule — `shopspring/decimal` over `float64`, nil-map and slice-aliasing hazards.
- → `go-testing` skill — hand-written fakes + the testcontainers integration recipe.
- → `go-security` rule — SQL injection prevention.

## References

- [pgx](https://github.com/jackc/pgx)
- [pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
- [database/sql tutorial](https://go.dev/doc/database/) (background only — not used here)
- [golang-migrate](https://github.com/golang-migrate/migrate)
