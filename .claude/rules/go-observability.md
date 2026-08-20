---
paths: ["api/**/*.go"]
---

# Go observability — api

<!-- Adapted from samber/cc-skills-golang@golang-observability v1.2.1 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-observability/. -->

Understand the system's internal state from its external outputs. The always-on signal
here is **structured logging** with the stdlib `log/slog` (JSON to stdout). Metrics and
tracing (otel is an indirect dep) are not yet wired — mention them, don't fabricate them.

## MUST

- **Use `log/slog`, not `fmt.Println`/`log.Printf`.** Production code emits structured JSON, not freeform strings. Obtain the logger via `logging.New(level)`.
- **Key-value attributes, stable message.** `slog.Error("health check: database ping failed", "error", err)` — the message is a low-cardinality template; IDs/paths/counts go in attributes, never interpolated into the message (that breaks grouping).
- **No PII or secrets in logs** — no passwords, tokens, full request bodies, emails as identifiers. Log a `user_id` (UUID), not an email.
- **Log errors once, at the boundary.** Wrap-and-return down the stack; the HTTP error boundary (`httputil.WriteError`) or `main` logs. A log-and-return pair duplicates the line up the whole chain (see `go-errors`).
- **Choose the right level:** Debug (dev), Info (normal ops), Warn (degraded), Error (needs attention). Don't log expected, handled conditions at Error.

## SHOULD

- **Use the context-aware variants** (`slog.InfoContext(ctx, ...)`) on request paths, so logs can later correlate with a request/trace ID once tracing lands.
- **Carry a correlation ID.** `middleware.RequestID` (chi) is in the stack already; thread that ID through so a single request's logs are linkable.
- **Keep label/attribute cardinality bounded** if/when Prometheus metrics are added — never an unbounded value (user ID, full URL) as a metric label.
- **Treat observability as part of "done"** for a new feature: it should log its failures meaningfully (and, once metrics/tracing exist, expose latency + error signals).

## beer-lms specifics

- **`logging.New(level string) *slog.Logger`** (`internal/platform/logging`) builds the stdlib `slog` JSON handler writing to `os.Stdout`. This is the only logger constructor — do not introduce zap/logrus/zerolog or `samber/slog`.
- **House style is key-value `slog`:** `slog.Error("msg", "error", err)`, matching `cmd/api` and the platform packages.
- **`middleware.RequestID`** runs early in the server middleware stack (`internal/server`) — the per-request correlation ID is already available.
- **Known deviation to fix forward:** `internal/platform/outbox/relay.go` still uses stdlib `log.Printf` for its claim/process/dead-letter lines. New code SHOULD prefer `slog`; when touching the relay, migrate those calls.

## Enforce / see also

`sloglint` (if enabled) and `forbidigo` (ban `fmt.Print*`/`log.Print*` in non-CLI code) help — see the `go-lint` skill.
For depth (metrics, tracing, RUM, alerting), upstream `references/` in `golang-observability`.
