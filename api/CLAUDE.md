# task-tracker/api — Go charter

Thin, always-on charter for the Go backend. It sets posture and indexes the helpers.
**It deliberately does NOT restate coding conventions** — those ride on path-rules that
auto-attach to every `*.go` edit (see `.claude/rules/go-*.md` in the monorepo root).
Don't duplicate them here.

## What this is
- Go 1.25 **modular monolith**, **manual dependency injection** (constructor injection wired in `cmd/api` — no wire/fx/dig).
- Module path: `github.com/genkovich/task-tracker/api`.
- Layout: `internal/modules/<domain>/{domain,app,ports,infra}` + `internal/platform/<concern>/` + `internal/server`.
- Stack: chi/v5 · pgx/v5 (+pgxpool) · golang-migrate · golang-jwt/v5 · testify + testcontainers · prometheus/client_golang · stdlib `log/slog` · google/uuid (v7) · AWS S3.
- Errors flow: domain sentinels → `ports/errors.go` `mapError` → `apperr.Error` → `httputil.WriteError`.
- Observability: `/livez`, `/readyz`, `/metrics` (Prometheus) served by `internal/server` outside the rate limiter.

## Working posture
- **TDD**, integration-focused tests. Commands (Makefile): `make test` · `make test-integration` (`-tags integration`, needs Docker for testcontainers) · `make lint` (golangci-lint) · `make vet` · `make check` (vet + lint + test).
- Migrations: paired `*.up.sql`/`*.down.sql` under `migrations/` — conventions in `.claude/rules/migrations.md`.
