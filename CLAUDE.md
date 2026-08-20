# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`task-tracker` is a fresh full-stack scaffold from `base-tpl` (Go + React), not yet carrying
product features — see `docs/idea-brief.md` for the product this repo is meant to become
(a single kanban board with drag-and-drop status changes and a read-only public share link,
deliberately no accounts, sprints, or estimates). The main page is currently a bare
Google-login button; the dashboard is a bare "Hello". Feature work is expected to land via
the SDD/SDLC skill pipeline (`/sdlc:interview`, `/sdlc:write-prd`, etc.) — see the plugin
skills listed in your environment.

## Commands

Root (`make` from repo root):
- `make up` — full local stack via docker compose (api, web, postgres, migrate job, Prometheus, Grafana)
- `make down` / `make logs` — stop stack / tail logs
- `make check` — end-to-end check: `api-check` + `web-check` (what CI runs)

Backend (`api/`, or `make -C api <target>`):
- `make -C api run` — run the API against the dockerized Postgres, outside docker (fastest dev loop)
- `make -C api test` — unit tests (`go test ./...`)
- `make -C api test-integration` — integration tests (`-tags integration`, needs Docker for testcontainers); `test-all` runs unit+integration together
- `make -C api lint` — golangci-lint (no-ops with a message if not installed locally; always runs in CI)
- `make -C api vet` — `go vet ./...`
- `make -C api check` — vet + lint + test (must be green before considering a backend change done)
- `make -C api migrate-go-up` / `migrate-go-down` — run migrations via the Go binary (no external CLI needed); `migrate-create name=...` scaffolds a paired `.up.sql`/`.down.sql` (requires the `migrate` CLI)
- Single test: `go test ./internal/modules/user/... -run TestName` from `api/`; integration tests need `-tags integration` added

Frontend (`web/`):
- `npm run dev` — Vite dev server (point it at a running API — `make -C api run` or `make up`)
- `npm run typecheck` — `react-router typegen && tsc`
- `npm test` / `npm run test:watch` — vitest unit tests (Testing Library, colocated `*.test.tsx`)
- `npm run test:e2e:smoke` — Playwright smoke suite; needs the full stack up (`make up`)
- `npm run format` / `format:check` — Prettier

## Architecture

### Backend — modular monolith, manual DI

Go 1.25, module `github.com/genkovich/task-tracker/api`. No DI framework — everything is wired
by hand in `cmd/api/main.go`: config → logging → database → auth → storage → `server.New(...)`.

Each domain lives under `internal/modules/<domain>/` in four layers:
- `domain/` — entities, domain errors, no framework imports
- `app/` — use-case services, depend on `ports` interfaces (not concrete infra)
- `ports/` — HTTP handlers, DTOs, and the interfaces `app` depends on (repo, storage, etc.); also where domain errors get mapped to `apperr.Error` via a `mapError` function
- `infra/` — concrete adapters (Postgres repos, S3, etc.) implementing `ports` interfaces

A top-level `<domain>.go` file (e.g. `internal/modules/user/user.go`) wires that module's own
layers with a single `New(...)` constructor returning a `*ports.Handler`, which is then passed
into `server.New(...)` in `cmd/api/main.go`. New modules follow this same shape — see the
`go-project-layout` skill for a copy-paste scaffold.

Cross-cutting concerns live in `internal/platform/<concern>/` (config, database, logging,
authmw, apperr, httputil, storage) — not inside any one module.

`internal/server` wires the chi router: modules register routes by implementing `RouteRegistrar`
(public) and/or `ProtectedRouteRegistrar` (behind `authMW`), passed as variadic `opts` to
`server.New`. `/livez`, `/readyz`, `/metrics` are mounted outside CORS/rate-limiting. Prometheus
metrics use a per-`Server`-instance registry (not the global one) so tests can call `server.New`
repeatedly without duplicate-registration panics.

Errors flow: domain sentinel errors → each module's `ports/errors.go` `mapError` → `apperr.Error`
→ `httputil.WriteError`.

Coding conventions (naming, error handling, concurrency, context, structs/interfaces, security,
observability) are **not** restated here — they live in `.claude/rules/go-*.md` and auto-attach
as path-rules to any `*.go` edit. `api/CLAUDE.md` is the thin charter that indexes them; don't
duplicate rule content into this file.

### Migrations

Paired `<NNNNNN>_<verb>_<entity>.up.sql` / `.down.sql` under `api/migrations/`, sequential
6-digit numbering (current head: `000005`). PKs are app-generated UUIDv7 (`google/uuid`), not
DB-generated. Full conventions (zero-downtime patterns, seed rules, forbidden constructs like
triggers or `SERIAL` PKs) are in `.claude/rules/migrations.md` — read it before adding a
migration. Note: that file's own worked examples (numbering "head 000019", a `mentorship`
module) predate this scaffold and don't match the current repo state; treat the *rules* as
authoritative and the *specific numbers/module names* in it as stale.

### Frontend — FSD, React Router 7 SPA

React 19 + React Router 7 in SPA mode (`ssr: false`) + TypeScript + Vite. Feature-Sliced Design:
`app/ → pages/ → widgets/ → features/ → entities/ → shared/`. Imports only flow "down" the
layer list; no cross-imports between slices in the same layer.

- Routes are declared centrally in `web/src/routes.ts`; protected routes are nested under
  `layout("app/layouts/ProtectedLayout.tsx", [...])`.
- UI primitives are shadcn-generated into `src/shared/ui` on top of Tailwind 4 — compose over
  them, don't hand-edit the generated primitives.
- API access goes through the typed client in `src/shared/api/client.ts`, with error display via
  `showApiError`.
- Auth: Google OAuth via `features/auth-by-google` + `app/providers/auth`, JWT held in memory
  (not persisted client-side).
- A new feature is a new slice under `features/` (`api/`, `model/`, `ui/`); pages assemble
  slices and should not hold business logic themselves.

### Infra / operability

- `docker-compose.yml` (root) is the full local stack: Postgres, a one-shot `migrate` job that
  gates the `api` service's startup, `api`, `web`, Prometheus, Grafana. The day-to-day dev loop
  bypasses compose for the app itself (`make -C api run` + `npm run dev`); compose is there to
  prove the whole thing boots together.
- Deploy: `.github/workflows/deploy.yml` + `deploy/` — GHCR images → VPS → migrations → health
  gate, Caddy with automatic TLS in front.
- CI (`.github/workflows/ci.yml`) runs the same checks as `make check` locally, plus
  `govulncheck`.
- `/metrics` is always present on the API (Prometheus client), scraped in both `docker-compose.yml`
  and `deploy/docker-compose.prod.yml`; the Grafana dashboard is provisioned from
  `deploy/grafana/dashboards/`.

### Agents, skills, hooks in this repo

- `.claude/rules/go-*.md` and `migrations.md` — path-rules auto-attached to matching file edits;
  this is where Go conventions actually live, not in this file or in `api/CLAUDE.md`.
- `.claude/skills/` — Go-specific procedural skills (`go-testing`, `go-database`, `go-lint`,
  `go-dependency-management`, `go-performance`, `go-benchmark`, `go-project-layout`, `go-ci`,
  `contracts-from-sequence`).
- `.claude/agents/` — read-only auditors and a reviewer (`go-security-auditor`,
  `go-concurrency-auditor`, `go-reviewer`, `go-troubleshooter`, `go-modernizer`); the auditors
  and reviewer never edit code themselves.
- `.claude/hooks/gofmt-go.sh` runs `gofmt` after every `Edit`/`Write`/`MultiEdit` — don't hand-run
  `gofmt` separately, the hook already covers it.
