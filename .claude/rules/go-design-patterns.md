---
paths: ["api/**/*.go"]
---

# Go design patterns & DI — api

<!-- Adapted from samber/cc-skills-golang@golang-design-patterns v1.1.4 + @golang-dependency-injection v1.2.1 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-design-patterns/, .claude/evals/golang-dependency-injection/. -->
<!-- RULE form: always-on. Apply a pattern only when it solves a real problem here; the DI -->
<!-- section pins the house decision (manual constructor injection) so no framework creeps in. -->

Idiomatic patterns for production Go — apply the *smallest* one that solves the problem, and push
back on premature abstraction. This repo's spine is plain constructor injection wired by hand in
`cmd/api`; the patterns below all serve testability and explicit lifecycle, not sophistication.

## MUST

- **Inject dependencies through constructors — no globals, no `init()` for service setup.** `init()` runs implicitly, can't return errors, and breaks tests. Every component takes what it needs: `NewCourseService(repo CourseRepository, clock Clock) *CourseService`. The composition root is `cmd/api/main.go`, and only there.
- **Manual constructor injection — do NOT introduce wire/fx/dig/samber-do.** This repo wires every service by hand in `cmd/api` (`server.New(db, cfg.CORSAllowedOrigins, authMW, orgMiddleware, server.WithAppEnv(cfg.AppEnv))`), and at this service count that is the correct, deliberate choice. Adding a DI framework is out of scope — flag the need, don't import one.
- **Functional options for constructors with optional/growing config**, defaults applied first. The repo's pattern: `type Option func(*Server)`, `func WithAppEnv(env string) Option`, then `New(..., opts ...)` loops the options over a defaulted struct. An option that can fail must return an error. Mirror `server.Option`/`WithAppEnv` for new optional config.
- **Panic is for bugs, not expected errors.** Network/DB/validation failures return an `error` a caller can handle; panic only for violated invariants or `Must*` at startup (`uuid.Must(uuid.NewV7())`). Don't panic on a recoverable condition.
- **`defer Close()`/`Rollback()` immediately after acquiring the resource**, not 50 lines later — `tx, err := r.db.Begin(ctx); …; defer tx.Rollback(ctx) //nolint:errcheck`, `defer rows.Close()`. A later edit must not be able to skip cleanup.

## SHOULD

- **Inject a `Clock` (or similar seam) for anything time/non-deterministic**, so tests stay deterministic. This repo has `app.Clock` with `RealClock()` in production and a fake in unit tests — wire the real one in `cmd/api`, the fake in the test.
- **Use the adapter pattern to break an import cycle** instead of reaching across layers. `internal/platform/outbox` declares the `OrgMemberLister` port it needs and `NewOrgMemberAdapter` wraps the org repo at the wiring layer — the platform package never imports `org/infra`. Reach for this when two packages would otherwise import each other.
- **Graceful shutdown via signal + `Shutdown(ctx)`.** `cmd/api/main.go` does `ctx, stop := signal.NotifyContext(context.Background(), …)` and `srv.Shutdown(shutdownCtx)`; new long-running components should hang their teardown off the same signal-derived context.
- **Keep the dependency graph shallow** and the domain layer pure (no framework imports). Validate at boundaries, trust internal code, and make illegal states unrepresentable with types.
- **Compile regexps once at package level** (`var x = regexp.MustCompile(…)`), embed static assets with `//go:embed` (migrations are already an embedded FS), and use `crypto/rand` — never `math/rand` — for tokens/keys.
- **Mock at the interface boundary.** Because ports are consumer-side interfaces (see `go-structs-interfaces.md`), unit tests inject hand-written fakes — no mock framework, matching the repo's testing style.

## beer-lms specifics

- Constructor + manual-wiring + `Clock` injection is the canonical service shape here: define the port in `app/ports.go`, take it in `NewXService(...)`, supply the real impl in `cmd/api`, supply a fake in `*_test.go`.
- Functional options currently configure the server only (`WithAppEnv`); extend that mechanism for new server-level config rather than widening the `New(...)` positional signature.

## Enforce / see also

`gocritic`, `revive`, `staticcheck` flag `init()` abuse and missing-`defer` patterns — see the `go-lint` skill. Interface/return-concrete design is in `go-structs-interfaces.md`; ctx-bound timeouts/cancellation in `go-context.md`; resource-loop pitfalls in `go-safety.md`. For depth see upstream `references/` in `golang-design-patterns` (architecture, resource-management) and `golang-dependency-injection` (manual-di).
