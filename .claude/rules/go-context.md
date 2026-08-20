---
paths: ["api/**/*.go"]
---

# Go context.Context — api

<!-- Adapted from samber/cc-skills-golang@golang-context v1.2.1 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-context/. -->
<!-- RULE form: always-on. ctx propagation is a per-edit discipline — a single broken link -->
<!-- (a stray context.Background() mid-request) silently defeats every downstream cancellation. -->

`context.Context` carries cancellation, deadlines, and request-scoped values across API
boundaries — it's the "session" of one unit of work. The whole point is propagation: thread the
*same* ctx end to end, and cancelling the top cancels everything below it for free.

## MUST

- **`ctx` is the first parameter, named `ctx context.Context`** — every function that does or calls I/O. No exceptions for "small" helpers.
- **Propagate the same ctx through the whole chain:** HTTP handler → app service → repository → pgx. Never swap in a fresh `context.Background()` in the middle — that severs cancellation and the request's deadline. This repo already flows `r.Context()` (from chi) down through the service to `db.Query/Exec(ctx, …)`.
- **Never store a context in a struct** — pass it explicitly per call. (Long-running components like the outbox relay receive ctx in `Start(ctx)` and pass it on; they don't stash it as a field.)
- **`context.Background()` only at the top level** — `main`, `init`, tests. `cmd/api/main.go` does this once via `signal.NotifyContext(context.Background(), …)`; nothing deeper should call `Background()`.
- **Call `cancel()` on every path for `WithCancel`/`WithTimeout`/`WithDeadline`** — `ctx, cancel := context.WithTimeout(…); defer cancel()`. Skipping it leaks the timer/goroutine. The health check does this: `ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second); defer cancel()` in `internal/server/server.go`.
- **Context value keys are unexported types, never bare strings** — bare keys collide across packages. This repo uses `type contextKey struct{}` with `ctx.Value(contextKey{})` in `internal/platform/authmw/authmw.go`; mirror that for any new request-scoped value.
- **Context values carry request-scoped metadata only** (auth claims, org context, request ID) — never function parameters that should be explicit arguments.

## SHOULD

- **Respect cancellation in any loop or wait** — `select { case <-ctx.Done(): return ctx.Err(); … }`, and check `ctx.Err()` between retry attempts. The outbox relay's `for { select { case <-ctx.Done(): return nil; default: } … }` is the model.
- **Use `context.TODO()`** (not `nil`, never `nil`) as a deliberate placeholder when a ctx is needed but not yet threaded through; it's greppable debt.
- **Inside an HTTP handler use `r.Context()`** as the root; carry it into every service and DB call rather than minting a new one.
- **Use `context.WithoutCancel` (Go 1.21+) for background work that must outlive the request** — e.g. fire-and-forget audit/outbox writes — so the response returning doesn't cancel the side effect mid-flight.
- **Timeout every external call** — pgx queries, S3 (`aws-sdk-go-v2`), Resend, Redis — with a `WithTimeout` derived from the inbound ctx, so a slow upstream can't pin a goroutine forever.

## beer-lms specifics

- Read auth/org context via the platform accessors, not raw `ctx.Value`: `authmw.AuthClaims(ctx) (*Claims, bool)` and `orgmw.OrgCtx(ctx)`. `WithClaims` is tests-only — don't use it in production paths.
- The server already wraps requests with `middleware.Timeout(30s)`; a per-call `WithTimeout` is for the specific downstream (DB ping, external API), layered under that envelope — not a replacement for it.

## Enforce / see also

`govet` (lostcancel) and `staticcheck` catch missing `cancel()` and misuse — see the `go-lint` skill. Database ctx specifics are in the `go-database` skill; goroutine cancellation in `go-concurrency.md`. For depth see upstream `references/` in `golang-context` (cancellation, values-tracing, http-services).
