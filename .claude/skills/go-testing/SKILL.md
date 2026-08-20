---
name: go-testing
description: "How to write and run tests for api — table-driven tests with named subtests, testify assert/require (is/must), hand-written fakes against consumer-side ports (fakeCourseRepo) + an injectable Clock, fuzzing, examples, coverage, and the integration recipe: //go:build integration + dbtest.StartPostgres (testcontainers postgres:18-alpine) + database.RunMigrations(migrations.FS) + miniredis. Use when writing or reviewing Go tests, setting up an integration test, or debugging a flaky/slow test. Commands: make test, make test-integration. Conventions are enforced by the go-tests rule; this skill is the procedure."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents. Targets api (Go 1.25, testify, testcontainers, miniredis).
metadata:
  author: samber
  version: "1.2.2"
  openclaw:
    emoji: "🧪"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(gotests:*) AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-testing v1.2.2 + @golang-stretchr-testify v1.2.1 (upstream 466ea6d). SKILL form (merge). Evals: .claude/evals/golang-testing/, .claude/evals/golang-stretchr-testify/. -->

**Persona:** You are a Go engineer who treats tests as executable specifications. You write tests to constrain behavior, not to hit coverage targets.

**Modes:**

- **Write mode** — generating new tests for existing or new code. Mirror the nearest existing `_test.go` in the same package; enrich with edge cases and error paths.
- **Review mode** — reviewing a PR's test changes: coverage of new behavior, assertion quality, table-driven structure, build tags, no flakiness.
- **Debug mode** — a test is failing or flaky: reproduce reliably (`go test -run TestX/sub -count=10`), isolate the failing assertion, find the root cause.

> This skill merges two upstream topics (golang-testing + golang-stretchr-testify). The always-on **conventions** (named subtests, `t.Parallel`, build tags, fakes-over-mocks, `errors.Is` in assertions) live in the `go-tests` rule (`.claude/rules/go-tests.md`) and are injected on every `*_test.go` edit — this skill is the **how-to**; it does not restate the conventions.

## In beer-lms

- **Unit tests use hand-written fakes — no mock framework** (`mockgen`/`mockery`/`testify/mock` are not used). Define the fake against the module's consumer-side port and inject it:
  - Canonical fake: `fakeCourseRepo` in `internal/modules/courses/app/service_test.go` — a struct implementing the `app.CourseRepository` port (`ListByOrg/GetByID/CreateCourse/UpdateCourse/DeleteCourse/CreateModule/…`), built via `newFakeRepo()`.
  - **Injectable `Clock`** for time-dependent logic: the `Clock` interface (`Now() time.Time`) is in `internal/modules/courses/app/ports.go`; production uses `RealClock()` (returns `time.Now().UTC()`), tests inject a `fakeClock{now}`.
- **Integration tests** carry `//go:build integration` and use real infra via testcontainers + miniredis (see recipe below). Run with `make test-integration`.
- **Commands (Makefile):** `make test` → `go test ./...`; `make test-integration` → `go test -tags integration -p 4 ./...`; `make test-all` → `go test -tags integration ./...`; `make vet`; `make fmt`. Add `-race` when chasing a concurrency bug (the `go-concurrency` rule requires `-race` in CI).

## Best Practices Summary

1. Table-driven tests MUST use named subtests — every case has a `name` field passed to `t.Run` (lowercase, descriptive: `"missing title"`, `"course not found"`).
2. Integration tests MUST use the `//go:build integration` build tag — they never run under plain `make test`.
3. Tests MUST be order-independent — each subtest passes alone (`go test -run TestX/sub`); no shared mutable global state.
4. Independent tests SHOULD use `t.Parallel()` — but only when they share no mutable state.
5. NEVER test implementation details — test observable behavior and the public API / port contract.
6. Prefer hand-written fakes over a mock framework; inject a fake `Clock` for time.
7. Use testify as a helper on top of `*testing.T`, not a replacement for the standard library.
8. Match sentinel errors with `require.ErrorIs(t, err, domain.Err…)` — never compare error strings.
9. Keep unit tests fast (sub-millisecond) and dependency-free; anything needing Postgres/Redis is an integration test.
10. Run tests with `-race` in CI.
11. Include `Example…` functions as executable documentation.

## Test Structure and Organization

### File and package conventions

```go
// service_test.go, package <x>_test — black-box, exercises the public API / port (the house default)
package courses_test

// occasionally package <x> — white-box, only when a test genuinely needs unexported access
package courses
```

### Naming

```go
func TestGetCourse(t *testing.T) { ... }              // function/use-case test
func TestCourseService_GetCourse(t *testing.T) { ... } // method test
func BenchmarkScanCourses(b *testing.B) { ... }        // benchmark (see go-benchmark skill)
func ExampleNewCourseService() { ... }                 // example
func FuzzParseSlug(f *testing.F) { ... }               // fuzz test
```

## Table-Driven Tests

The idiomatic Go shape — always name each case:

```go
func TestCalculatePoints(t *testing.T) {
    tests := []struct {
        name     string
        lessons  int
        expected int
    }{
        {name: "no lessons", lessons: 0, expected: 0},
        {name: "single lesson", lessons: 1, expected: 10},
        {name: "full module", lessons: 5, expected: 50},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got := CalculatePoints(tt.lessons)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

## testify: assert vs require

Both packages offer identical assertions; the difference is failure behavior:

- **`require`** calls `t.FailNow()` — use for preconditions where continuing would panic or mislead (error checks, non-nil setup).
- **`assert`** records the failure and continues — use for independent field checks so you see all failures at once.

Use `assert.New(t)` / `require.New(t)` for readability, named `is` and `must`:

```go
func TestParseConfig(t *testing.T) {
    is := assert.New(t)
    must := require.New(t)

    cfg, err := ParseConfig("testdata/valid.yaml")
    must.NoError(err)   // stop here if parsing failed — cfg would be nil
    must.NotNil(cfg)

    is.Equal("production", cfg.Environment)
    is.Equal(8080, cfg.Port)
}
```

**Rule:** `require` for preconditions, `assert` for verifications. Never mix randomly. Argument order is always `(expected, actual)`.

### Core assertions

```go
is := assert.New(t)

is.Equal(expected, actual)            // DeepEqual + exact type
is.NotEqual(unexpected, actual)
is.Nil(obj)      is.NotNil(obj)
is.True(cond)    is.False(cond)
is.Empty(coll)   is.NotEmpty(coll)    is.Len(coll, n)
is.Contains("hello world", "world")   // strings, slices, map keys
is.Greater(a, b) is.Less(a, b)        is.Zero(v)

// Errors — walk the chain, never compare strings
is.NoError(err)  is.Error(err)
is.ErrorIs(err, domain.ErrCourseNotFound)
is.ErrorAs(err, &appErr)
is.ErrorContains(err, "not found")

// Type
is.IsType(&domain.Course{}, obj)
```

### Advanced assertions

```go
is.ElementsMatch([]string{"b", "a"}, got)         // unordered comparison
is.InDelta(3.14, computed, 0.01)                  // float tolerance
is.JSONEq(`{"a":1}`, body)                        // ignores whitespace/key order — great for handler tests
is.WithinDuration(expected, actual, time.Second)
is.Regexp(`^course-[a-f0-9-]+$`, id)

// Async polling (e.g. waiting on an outbox relay to drain)
is.Eventually(func() bool {
    return store.Processed(eventID)
}, 5*time.Second, 50*time.Millisecond)
```

**Common testify mistakes:** comparing a wrapped error with `is.Equal(ErrX, err)` (use `is.ErrorIs`); swapped `(expected, actual)`; using `assert` for a guard that the next line dereferences (use `require`).

## Hand-Written Fakes (the house style)

beer-lms isolates the service from infra with a fake that implements the consumer-side port — no mock framework. Shape (after `fakeCourseRepo`):

```go
type fakeCourseRepo struct {
    courses map[uuid.UUID]*domain.Course
    err     error // inject to exercise error paths
}

func newFakeRepo() *fakeCourseRepo {
    return &fakeCourseRepo{courses: map[uuid.UUID]*domain.Course{}}
}

func (f *fakeCourseRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Course, error) {
    if f.err != nil {
        return nil, f.err
    }
    c, ok := f.courses[id]
    if !ok {
        return nil, domain.ErrCourseNotFound
    }
    return c, nil
}
// ... implement the remaining port methods ...
```

Inject time the same way — a `fakeClock` implementing the `Clock` port:

```go
type fakeClock struct{ now time.Time }
func (c fakeClock) Now() time.Time { return c.now }

svc := app.NewCourseService(newFakeRepo(), fakeClock{now: fixed})
```

Why fakes over `testify/mock` here: fakes are real Go code the compiler checks, they read like the production type, and they don't couple the test to a call sequence. For the `testify/mock` API (argument matchers, call modifiers, `AssertExpectations`) — available but **not the house style** — see [Mocking](./references/mocking.md) and [testify/mock reference](./references/testify-mock.md).

## Testing HTTP Handlers

Use `net/http/httptest` with table-driven cases; assert status + body with `is.JSONEq`. The repo's handlers read auth/org from context, so set it in the test via `authmw.WithClaims` (tests-only) before serving. See [HTTP Testing](./references/http-testing.md).

## Integration Tests (the beer-lms recipe)

First line is the build tag; spin up real Postgres via the `dbtest` helper, apply migrations, then exercise the real `infra` adapter:

```go
//go:build integration

package infra_test

func TestCommentRepo_Integration(t *testing.T) {
    ctx := context.Background()

    c := dbtest.StartPostgres(ctx, t)                 // testcontainers postgres:18-alpine; t.Skip if no docker; auto-terminate
    if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
        t.Fatalf("RunMigrations: %v", err)
    }

    repo := infra.NewPostgresCommentRepository(c.DB)   // c.DB is *database.DB
    author := uuid.MustParse("019a0000-0000-7000-8000-000000000001") // seed admin UUID

    // ... write + read back, assert with require/assert ...
}
```

- `dbtest.StartPostgres(ctx, t)` lives in `internal/platform/database/dbtest/container.go` and returns `*Container{ DB *database.DB; DSN string }`.
- **Redis** in integration tests uses **miniredis** (no container): `mr := miniredis.RunT(t)` then point your store/limiter at `mr.Addr()` — see `internal/platform/idempotency/redis_store_test.go`, `internal/platform/ratelimit/redis_limiter_test.go`, and `test/integration/course_lesson/e1_lifecycle_test.go`.
- Run them: `make test-integration` (`go test -tags integration -p 4 ./...`). They are excluded from `make test`.
- For container/fixture detail see [Integration Testing](./references/integration-testing.md) (its docker-compose + `sql.Open` + `suite.Suite` shape is the upstream spelling — beer-lms uses `dbtest.StartPostgres` + `database.RunMigrations(migrations.FS)` instead).

## Deterministic Goroutine Tests — testing/synctest (Go 1.25+)

For concurrent code with timers/deadlines/`context` cancellation, `testing/synctest` makes ordering deterministic — synthetic time advances only when all goroutines are blocked:

```go
func TestContextTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {                 // Go 1.25+ stable API — NOT the old experimental synctest.Run
        ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
        defer cancel()

        time.Sleep(5 * time.Second)
        synctest.Wait()
        if err := ctx.Err(); err != context.DeadlineExceeded {
            t.Fatalf("got %v, want DeadlineExceeded", err)
        }
    })
}
```

> `go.uber.org/goleak` is **not a dependency** here. To assert goroutines don't leak, give every goroutine a clear exit (ctx cancellation / done channel) per the `go-concurrency` rule and verify cleanup via `t.Cleanup` + the relay's `Start(ctx)` shutdown contract — don't add goleak.

## Fuzzing

```go
func FuzzReverse(f *testing.F) {
    f.Add("hello"); f.Add(""); f.Add("a")
    f.Fuzz(func(t *testing.T, in string) {
        if Reverse(Reverse(in)) != in {
            t.Errorf("round-trip failed for %q", in)
        }
    })
}
```

Run: `go test -fuzz=FuzzReverse ./...`.

## Examples as Documentation

```go
func ExampleCalculatePoints() {
    fmt.Println(CalculatePoints(5))
    // Output: 50
}
```

Verified by `go test`.

## Code Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total   # total %
go tool cover -html=coverage.out                # browse uncovered lines
```

(`make coverage` writes `coverage.out` + `coverage.html`.)

## Scaffolding (optional)

`gotests` can generate a table-driven skeleton you then enrich:

```bash
go install github.com/cweill/gotests/gotests@latest
gotests -all -w ./internal/modules/courses/app/service.go
```

Optional only — hand-writing the table to match the nearest existing test is equally fine.

## Quick Reference

```bash
go test ./...                            # all unit tests (make test)
go test -run TestGetCourse ./...         # one test by name
go test -run TestGetCourse/missing_title ./...   # one subtest
go test -race ./...                      # race detection
go test -cover ./...                     # coverage summary
go test -bench=. -benchmem ./...         # benchmarks (see go-benchmark)
go test -tags integration -p 4 ./...     # integration tests (make test-integration)
```

## Enforce with Linters

`thelper`, `paralleltest`, `tparallel`, `testifylint` catch most test-convention mistakes — see the `go-lint` skill.

## Cross-References

- → `go-tests` rule — the always-on conventions (this skill is the procedure).
- → `go-database` skill — repository integration tests against the `dbtest` Postgres container.
- → `go-concurrency` rule — `-race`, goroutine exit contracts (replaces goleak here).
- → `go-lint` skill — `testifylint` / `paralleltest` configuration.
- → [testify/mock reference](./references/testify-mock.md) — available but not the house style.

## References

- [Go testing package](https://pkg.go.dev/testing)
- [testify](https://github.com/stretchr/testify)
- [testing/synctest](https://pkg.go.dev/testing/synctest)
