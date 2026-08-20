---
paths: ["api/**/*.go"]
---

# Go code style — api

<!-- Adapted from samber/cc-skills-golang@golang-code-style v1.2.0 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-code-style/. -->
<!-- RULE form: always-on for every api Go edit. Linters handle formatting; this rule -->
<!-- carries the clarity choices that need human judgment on each edit. Deviations noted inline. -->

"Clear is better than clever." Linters fix formatting — this rule covers the clarity decisions
they cannot. Accept the formatter's output; when you break a rule here, leave an inline comment.

## MUST

- **Accept `gofmt`/`goimports` output verbatim.** Never hand-format against the formatter — run `make fmt`. Tabs for indentation, formatter-managed import grouping. Do not fight it back.
- **Handle errors and edge cases first (early return / guard clauses).** Keep the happy path at minimal indentation. This repo's settled handler shape: `if !ok { httputil.WriteValidationError(w, "auth.missing_token", "..."); return }`, then continue unindented. See `internal/modules/courses/ports/handler.go`.
- **Drop `else` after a terminating branch.** When the `if` body ends in `return`/`break`/`continue`, delete the `else` and dedent. For mutually exclusive assignments use default-then-override with `switch { case … }`, not an `if/else-if` chain.
- **Composite literals use field names** — `&http.Server{Addr: …, ReadTimeout: …}`, never positional. Positional literals break silently when a field is added or reordered.
- **Initialize slices/maps explicitly, never leave nil** when they get written or JSON-encoded. A nil map panics on write; a nil slice marshals to `null` (not `[]`), surprising API clients. `users := []User{}` / `m := map[string]int{}`.
- **Functions take `≤4` parameters**; beyond that pass a struct. `context.Context` is the first parameter, then inputs, then output destinations (see `go-context.md`).
- **Inline-suppress a linter only with a reasoned directive:** `//nolint:directive // reason`. This repo does exactly this — `argIdx++ //nolint:ineffassign,wastedassign // will be used by future filter params` and `defer tx.Rollback(ctx) //nolint:errcheck` in `internal/modules/courses/infra/postgres_course_repository.go`. No bare `//nolint`.

## SHOULD

- **`:=` for non-zero values, `var` for zero-value intent.** `var count int` (set later), `var buf bytes.Buffer` (zero value ready to use), `name := "default"` (non-zero).
- **Keep error-return signatures short, error last:** `func FetchUser(ctx context.Context, id string) (*User, error)`. Name returns only in genuinely long functions where a bare `return` would force scrolling; short funcs can return inline.
- **Break lines at semantic boundaries past ~120 cols, not at a fixed column.** A call with 4+ arguments goes one argument per line with the closing paren on its own line. If a signature is too long, the fix is usually fewer parameters (a struct), not cleverer wrapping.
- **Extract a named boolean when an `if` has 3+ operands** — `isAdmin := …; isOwner := …; if isAdmin || isOwner { … }`. A wall of `||`/`&&` hides the business rule.
- **`switch` over a repeated-variable `if`-chain;** scope check-only vars into the `if` (`if err := validate(in); err != nil { … }`).
- **Comment the *why*, not the *what*.** Code says what it does; comments justify non-obvious decisions. Delete comments that merely restate the next line.
- **Prefer modern stdlib** — `min`/`max`, the `slices`/`maps` packages, `errors.Join` — over hand-rolled equivalents. (A full sweep is the `go-modernizer` agent's job, not this rule's.)
- **Unexport aggressively** and keep one primary type per file with its constructor and methods grouped. Exporting later is cheap; unexporting is a breaking change.

## beer-lms specifics

- The early-return guard wired to `httputil` is the house transport idiom: validate auth → org → path params, each `if !ok/err { httputil.Write…; return }`, then one happy-path call ending in `httputil.WriteError(w, mapError(err))`. Mirror it in new handlers rather than nesting.
- Capacity-hinted literals are idiomatic here even though they're a style+perf call: `make([]CourseResponse, 0, len(courses))` (`ports/handler.go`). Pre-size when the length is known; don't speculatively over-allocate.

## Enforce / see also

`gofmt`/`gofumpt`/`goimports`, `revive`, `gocritic`, `wsl_v5` automate most formatting — run `make fmt` and `make lint` (config from the `go-lint` skill). Identifier naming lives in `go-naming.md`; receiver/zero-value design in `go-structs-interfaces.md`. For depth see upstream `references/` in `golang-code-style`.
