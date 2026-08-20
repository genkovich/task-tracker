---
paths: ["api/**/*.go"]
---

# Go error handling — api

<!-- Adapted from samber/cc-skills-golang@golang-error-handling v1.2.0 (upstream 466ea6d). RULE form (audit half -> go-reviewer / go-security-auditor). Evals: .claude/evals/golang-error-handling/. -->

Every error is an event that must be either handled or propagated with context.
Silent failures and duplicate logs are equally unacceptable. No `samber/oops` — this repo
uses the stdlib `errors` package, sentinel errors, and a per-module `apperr.Error` mapping.

## MUST

- **Check every returned error.** Never discard with `_`. The error is always the **last** return value.
- **Wrap with `%w` + context** when crossing a layer: `fmt.Errorf("create course: %w", err)`. Use `%w` internally (preserves the chain); only flatten to `%v` at a true system boundary where you intend to hide the chain.
- **Error strings are lowercase, no trailing punctuation** — they get wrapped, so mid-sentence caps read wrong. (See `go-naming`.)
- **Match sentinels with `errors.Is`, never `==`.** Inspect typed errors with `errors.As(err, &target)` (Go 1.25 here — `errors.As`, not `errors.AsType`).
- **Sentinel errors for expected conditions, typed errors to carry data.** Declare sentinels in the package that owns the concept: `var ErrCourseNotFound = errors.New("course not found")` in `domain`.
- **Log OR return, never both** (single-handling rule). Wrap-and-return down the stack; log **once** at the boundary (the HTTP handler / `main`). A log-and-return pair duplicates the line in every aggregator.
- **No `panic` for expected error conditions.** Reserve `panic` for truly unrecoverable startup/programmer errors; never in request paths or library code.

## SHOULD

- **`errors.Join`** (Go 1.20+) to combine independent errors (e.g. accumulated validation failures) rather than returning only the first.
- **Never leak technical errors to users.** Translate to a stable user-facing message + wire code; log the technical detail server-side.
- **Keep log messages low-cardinality** — stable template, IDs/paths/counts as structured attributes (see `go-observability`).
- **Fail closed.** On an error in a security-sensitive path (auth, crypto, authorization), deny — never proceed as if it succeeded.

## beer-lms specifics

- **Three-layer error flow:** `domain` returns a **sentinel** (`domain.ErrCourseNotFound`, `domain.ErrForbidden`, `domain.ErrTitleRequired`) → `ports/errors.go` maps it to a wire error → transport writes it.
- **The `errorMap` table + `mapError`** (`internal/modules/courses/ports/errors.go`): a `[]struct{ target error; appErr apperr.Error }`, matched with `errors.Is`; `mapError` falls through unknown errors unchanged (→ 500). New domain sentinels go in this table.
- **`apperr.Error{Code, Message, StatusCode}`** is the wire shape; `Code` is dotted `domain.snake_case` (`course.not_found`, `validation.invalid_body`, `auth.missing_token`) — a wire contract, not a Go name.
- **`httputil.WriteError(w, err)`** does `errors.As(err, &appErr)`; an unmapped error becomes 500 `internal_error`. This is the single log/handle boundary — do **not** also log at the service layer.
- **Repos translate pgx not-found to a domain sentinel:** `if errors.Is(err, pgx.ErrNoRows) { return nil, domain.ErrCourseNotFound }`; otherwise wrap: `fmt.Errorf("get course: %w", err)`.
- **Constraint violations → domain conflict.** Use the existing helpers in `internal/platform/database/errors.go`: `database.IsPgUniqueViolation(err)` (23505) and `database.IsPgForeignKeyViolation(err)` (23503), built on `*pgconn.PgError` + `pgerrcode`. Map their result to a domain sentinel; don't re-implement the `errors.As` + code check inline.

## Enforce / see also

`errcheck`, `errorlint`, `errname`, `nilerr`, `wrapcheck` catch most of this — see the `go-lint` skill.
For depth, upstream `references/` in `golang-error-handling` (error-creation, error-wrapping, error-handling).
For a full error-handling audit, use the `go-reviewer` / `go-security-auditor` agents.
