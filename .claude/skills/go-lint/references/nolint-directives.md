# Nolint Directives

## Syntax

```go
//nolint:lintername // justification explaining why this suppression is needed
```

Place the directive on the same line as the flagged code, or on the line immediately above it.

## Rules

1. **MUST specify the linter name** — bare `//nolint` suppresses all linters on that line and makes it impossible to track what is being suppressed.
2. **SHOULD add a justification comment** — future readers (and your future self) need to understand why. (In the beer-lms config this is encouraged but not enforced — see rule 3.)
3. **The `nolintlint` linter enforces the name, not the reason, in beer-lms.** The shipped `.golangci.yml` sets `require-specific: true` (linter name mandatory) and `require-explanation: false`, so the repo's reason-less `defer tx.Rollback(ctx) //nolint:errcheck` passes while a bare `//nolint` is still rejected. If you flip `require-explanation` to `true`, first backfill reasons on the existing directives.
4. **MUST fix the root cause before suppressing** — only suppress after confirming the issue is a false positive or an intentional pattern.

### Real directives in this repo (keep these passing)

```go
defer tx.Rollback(ctx) //nolint:errcheck                                  // tx safety-net rollback
argIdx++ //nolint:ineffassign,wastedassign // will be used by future filter params
```

## Examples

```go
// Specific linter with reason
//nolint:errcheck // fire-and-forget logging, error not actionable
_ = logger.Sync()

// Type assertion is safe because preceding type switch guarantees the type
v := x.(MyType) //nolint:forcetypeassert // guaranteed by type switch on line 42

// Orchestration function has inherent complexity
//nolint:gocyclo // orchestration function coordinating 8 subsystems
func orchestrate() error {

// Table-driven test with many cases
//nolint:funlen // table-driven test, length is proportional to case count
func TestParser(t *testing.T) {

// Intentional parallel structure is clearer than abstracting
//nolint:dupl // intentional parallel structure for readability
```

## Multiple Linters

Suppress multiple linters on one line with comma separation:

```go
//nolint:errcheck,gosec // fire-and-forget in test helper
```

## When to Suppress vs. When to Fix

**Fix** (almost always):

- `errcheck` — check the error, even if just logging it
- `govet` — these are usually real bugs
- `staticcheck` — deprecated API usage, logic errors
- `bodyclose`, `sqlclosecheck` — resource leaks are real issues

**Suppress** (with justification):

- `funlen` — table-driven tests with many cases
- `gocyclo` — orchestration functions where splitting would obscure the flow
- `dupl` — intentional parallel structure that is clearer than an abstraction
- `exhaustive` — when a default case intentionally handles remaining values
- `goconst` — when extracting to a constant would reduce clarity (e.g., test assertions)

**Never suppress without strong justification**:

- Security linters (`bodyclose`, `sqlclosecheck`, `rowserrcheck`) — these catch real resource leaks
- `errcheck` on production code paths — unchecked errors cause silent failures
