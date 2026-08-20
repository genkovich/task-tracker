---
paths: ["api/**/*.go"]
---

# Go naming conventions — api

<!-- Adapted from samber/cc-skills-golang@golang-naming v1.1.1 (upstream 466ea6d, 2026-06-23). -->
<!-- RULE form: always-on for every api Go edit. The upstream SKILL is on-demand; -->
<!-- naming needs reminding on EVERY edit (+24pp uplift in upstream EVALUATIONS), so it is a rule. -->
<!-- Deviations from upstream defaults are documented inline. Evals: .claude/evals/golang-naming/. -->

Go favors short, readable names. Capitalization controls visibility (uppercase = exported).
All identifiers use **MixedCaps**, never underscores. To override a rule, add an inline comment.

## MUST

- **MixedCaps / mixedCaps only.** No `snake_case`, `ALL_CAPS`, or Hungarian (`kBuf`). Exceptions: test subcases (`TestX_InvalidInput`), generated code, cgo.
- **Acronyms keep one case.** `URL`, `ID`, `HTTP`, `JWT`, `org` — never `Url`, `Id`, `Http`. So `FetchByID`, `OrgID`, `UserID`, not `FetchById`/`OrgId`. (This repo: `CoverImageURL`, `AuthorUserID`, `oc.OrgID`.)
- **Error strings are fully lowercase, no trailing punctuation** — including acronyms: `"invalid course id"`, not `"invalid course ID"`. They get wrapped (`fmt.Errorf("create course: %w", err)`), so mid-sentence caps read wrong.
- **Sentinel errors use the `Err` prefix**, declared in the package that owns the concept: `domain.ErrCourseNotFound`, `domain.ErrForbidden`. Custom error *types* use the `Error` suffix (`apperr.Error`), never an `Err`-prefixed type.
- **Constructors:** `New` for a package's single primary type (`server.New`, `logging.New`, `apperr` has no constructor); `NewTypeName` only when a package builds several types (`NewCourseService`, `NewHandler`, `NewPostgresCourseRepository`, `NewRelay` all coexist in their packages — correct).
- **No stuttering.** The package name is always at the call site. `apperr.Error` not `apperr.AppError`; `httputil.WriteJSON` not `httputil.WriteJSONResponse`. New package `foo` → type `foo.Client`, not `foo.FooClient`.
- **Receivers: one consistent 1–2 letter name per type.** This repo's settled choices: `(s *Server)`, `(h *Handler)`, `(r *PostgresCourseRepository)`, `(t *Tx)`, `(c *Claims)`, `(db *DB)`. Never `this`/`self`, never mixed across methods of one type.
- **Booleans use `is`/`has`/`can`.** Fields: `isConnected`, `hasPermission`; methods keep the prefix: `IsAdmin()`, `IsMentor`. Not bare `connected`/`admin`.

## SHOULD

- **Enum zero value = `Unknown`/`Invalid` sentinel** at iota 0, or start real values at `iota + 1`, so an uninitialized `var s Status` is detectable rather than silently a valid state. Enum members are type-prefixed: `StatusDraft`, `StatusPublished` (this repo uses `domain.Status` string consts).
- **Getters omit `Get`:** `claims.UserID` (field) / `Status()` (method), never `GetStatus()`. `Is`/`Has` stays only for `bool` returns.
- **Name length ∝ scope.** `i`, `e`, `ev`, `c`, `m` in tight loops; descriptive names at package scope. `ctx` for `context.Context`, `err` for `error` (never `e`).
- **Don't encode the type in the name:** `courses` not `courseSlice`, `count`/`n` not `countInt`, `timeout` not `timeoutDuration`.
- **Variant suffixes/prefixes:** `WithContext` for ctx variants, `In` for in-place, `Must` for panic-on-error, `f` for format funcs (`Errorf`, `Wrapf`), `With…` for functional options (`server.WithAppEnv`).
- **Multi-method interfaces are nouns** (`CourseRepository`, `TokenValidator`, `RouteRegistrar`); single-method interfaces take the `-er` form (`OrgMemberLister`, `Reader`). Use canonical method names: `String()` not `Stringify()`, `Close()` not `Release()`.
- **No generic packages** (`util`, `helpers`, `common`, `base`, `models`). Name by concept: `apperr`, `httputil`, `authmw`, `idempotency`. Packages are singular, lowercase, single-word.
- **Import aliases only on real collision** (`mrand "math/rand/v2"`); this repo aliases `httpSwagger`, `pgxdecimal` for genuine clarity/collision — not for taste.

## beer-lms specifics

- **Error codes** (the `apperr.Error.Code` string, not Go identifiers) use dotted `domain.snake_case`: `course.not_found`, `validation.invalid_course_id`, `auth.missing_token`. See `internal/modules/courses/ports/errors.go`. This is a wire contract, distinct from Go naming.
- **Module layout drives names:** `domain` (entities + `Err*` sentinels), `app` (`*Service`, `*Params`, consumer-side `*Repository`/`Clock` ports), `ports` (HTTP `Handler`, `*Request`/`*Response` DTOs), `infra` (`Postgres*Repository`). Mirror these suffixes in new modules.

## Enforce with linters

`revive`, `predeclared`, `misspell`, `errname` catch most of this — see the `go-lint` skill (provides this repo's `.golangci.yml`). For full rules/examples see upstream `references/` in `golang-naming` (packages-files, identifiers, functions-methods, types-errors, testing).
