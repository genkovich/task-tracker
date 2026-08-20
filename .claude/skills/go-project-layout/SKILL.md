---
name: go-project-layout
description: "Scaffolding NEW code in the api modular monolith. Use when adding a new domain module, a new platform concern, or a new cmd/ binary, or when deciding where a file belongs. A module is internal/modules/<domain>/{domain,app,ports,infra} wired into cmd/api via the RouteRegistrar / ProtectedRouteRegistrar / OrgScopedRouteRegistrar interfaces; cross-cutting code goes in internal/platform/<concern>. Ships a copy-paste module skeleton (assets/MODULE_SCAFFOLD.md) plus the repo's real Makefile/.gitignore."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents, and for the api Go backend (Go 1.25).
metadata:
  author: samber
  version: "1.2.0"
  openclaw:
    emoji: "📁"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-project-layout v1.2.0 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-project-layout/. -->
<!-- Heavily adapted for api: this repo is a modular monolith with manual DI, NOT the generic golang-standards pkg//cmd template. The upstream "ask which architecture / which DI framework" steps are pre-decided here, so they are removed; the Cobra+Viper config reference was dropped (config comes from env via internal/platform/config). The real layout, the module skeleton, and the registrar wiring are the deliverable. -->

**Persona:** You are the api architect. The architecture is already decided — your job is to make new code *fit it*, not to re-litigate it. You right-size: a new platform helper stays flat; a new domain gets the full four-layer module.

# Go Project Layout — api

## The architecture is already chosen — do not re-ask

api is a **modular monolith with manual dependency injection**. Unlike a greenfield project,
you do **not** ask the developer which architecture or which DI framework to use — those are settled:

- **Manual constructor injection**, wired by hand in `cmd/api/main.go`. Do not introduce wire / fx /
  dig / samber-do — at this service count, hand-wiring is the deliberate, correct choice (see the
  `go-design-patterns` rule).
- **Config from the environment** via `internal/platform/config` (12-factor) — not Cobra/Viper config
  files. Logs are structured JSON to stdout via `log/slog` (`internal/platform/logging`).
- **Module path:** `github.com/genkovich/task-tracker/api`, **Go 1.25**.

The only real decision per feature is **module vs platform concern**, and **which registrar** a new
HTTP surface implements. Both are answered below.

## The real top-level layout

```
api/
├── cmd/
│   ├── api/        # the API server — the composition root (manual DI lives here)
│   └── migrate/    # one-off migration runner (golang-migrate from embedded FS)
├── internal/
│   ├── modules/<domain>/   # one folder per business domain (see below)
│   ├── platform/<concern>/ # cross-cutting infrastructure (see below)
│   └── server/             # chi router + middleware stack + the registrar interfaces
├── migrations/     # *.up.sql / *.down.sql, embedded and applied via database.RunMigrations
├── docs/           # generated swagger (make swagger)
├── go.mod          # module github.com/genkovich/task-tracker/api, go 1.25
├── Makefile        # see assets/Makefile — CI runs these same targets
└── .gitignore      # see assets/.gitignore
```

> This is **not** the generic `golang-standards/project-layout` (`pkg/`, top-level `api/`, `configs/`).
> That template is background only — business code here lives in `internal/`, and there is no `pkg/`
> because nothing is published for external consumers. Don't create `pkg/`, `util/`, `helpers/`, or
> `models/` (see the `go-naming` rule on generic package names).

## Decision 1: module or platform concern?

| It is… | Put it in | Examples |
| --- | --- | --- |
| A **business domain** (its own entities, rules, HTTP surface, table) | `internal/modules/<domain>/` | courses, lessons, comments, completions, announcements, mentorship, calendar, feedback, notifications, org, preferences, team, user, auth |
| **Cross-cutting infrastructure** used by many modules | `internal/platform/<concern>/` | apperr, authmw, orgmw, httputil, database, logging, outbox, idempotency, ratelimit, redis, storage, mailer, config |
| **Routing / middleware / the registrar interfaces** | `internal/server/` | the chi stack, security headers, timeouts, rate limit |
| **A new runnable binary** | `cmd/<name>/` | a worker, a backfill tool — `package main`, minimal logic, calls into `internal/` |

If two modules need the same logic, it's a platform concern — lift it to `internal/platform/`, don't
import one module from another.

## Decision 2: the four-layer module

A module is four packages with a strict dependency direction `domain ← app ← {ports, infra}`:

| Layer | Package | Contains | Imports |
| --- | --- | --- | --- |
| **domain** | `domain/` | entities, `Err*` sentinels, pure rules | stdlib + uuid/decimal only — **no I/O, no framework** |
| **app** | `app/` | `XService` + `NewXService`, **consumer-side** ports (`XRepository`, `Clock`), `*Params` | `domain` |
| **ports** | `ports/` | chi `Handler` + `NewHandler`, `dto.go` (`*Request`/`*Response`), `errors.go` (`mapError`+`errorMap`) | `app`, `domain`, platform |
| **infra** | `infra/` | `PostgresXRepository` + `NewPostgresXRepository(db)` satisfying the app port | `domain`, `database` |

Plus a wiring seam at the module root, `<domain>.go`, exposing `New(db) *ports.Handler` (mirror
`internal/modules/courses/courses.go`). **Interfaces are declared where they are consumed** — the
service declares `XRepository`, the handler declares `XAppService` — so the infra adapter and the app
service merely satisfy them (see the `go-structs-interfaces` rule).

**→ Copy the full skeleton from [assets/MODULE_SCAFFOLD.md](assets/MODULE_SCAFFOLD.md)** — it has every
file with real beer-lms symbols (`apperr.Error`, `httputil.WriteJSON`/`WriteError`/`WriteValidationError`,
`authmw.AuthClaims`, `orgmw.OrgCtx`, `database.DB`, `uuid.Must(uuid.NewV7())`) and an init checklist.

## Decision 3: which registrar does the HTTP surface implement?

Module handlers are passed to `server.New(...)` and picked up by whichever interface they implement
(defined in `internal/server/server.go`). Choose by route shape:

| Route shape | Interface | Method |
| --- | --- | --- |
| Public, no auth | `server.RouteRegistrar` | `RegisterRoutes(r chi.Router)` |
| Authenticated, not org-scoped | `server.ProtectedRouteRegistrar` | `RegisterProtectedRoutes(r chi.Router)` |
| Nested under `/orgs/{orgId}/`, needs org membership | `server.OrgScopedRouteRegistrar` | `RegisterOrgRoutes(r chi.Router)` |

A handler may implement more than one. Register it in `cmd/api/main.go` exactly as the existing module
handlers are threaded into `server.New(...)` — don't invent a new registry mechanism.

## Naming (enforced by the go-naming rule)

- Package names: lowercase, singular, single word, matching the directory (`courses`, `app`, `ports`,
  `infra`, `apperr`, `httputil`). No `util`/`helpers`/`common`/`models`.
- Module path matches the repo: `github.com/genkovich/task-tracker/api`. Acronyms keep one case (`OrgID`,
  `UserID`, `CoverImageURL`).
- → See the `go-naming` rule for the full convention; this skill only places files.

## Essential root files

- **Makefile** — [assets/Makefile](assets/Makefile) (the repo's real targets; CI invokes the same ones).
- **.gitignore** — [assets/.gitignore](assets/.gitignore) (the module-root file; the repo root also ignores `.claude/`).
- **.golangci.yml** — provided by the `go-lint` skill, not this one.

## Tests, benchmarks, examples

Co-locate `_test.go` with the code. Unit tests use `package <x>_test` with hand-written fakes + an
injected `Clock` (no mock framework). Integration tests start with `//go:build integration` and use
`dbtest.StartPostgres(ctx, t)` (testcontainers, auto-skip without Docker). See [testing-layout](references/testing-layout.md)
for placement details and the `go-testing` skill for the testing approach.

## Go workspaces

This repo is a **single module**, so `go.work` is not used. [workspaces](references/workspaces.md) is
kept as background only — reach for it if the React/TS `web` ever gains a sibling Go module
worth developing in lockstep; otherwise ignore it.

## In beer-lms

- **Read `internal/modules/courses/` as the reference module** — it has all four layers fully built,
  plus the `courses.go` wiring seam. Any new module should be structurally diff-able against it.
- **The composition root is `cmd/api/main.go` and only there.** Constructors take their deps; nothing
  uses `init()` or globals for service setup. New wiring goes in `main.go`, not scattered.
- **The registrar interfaces live in `internal/server/server.go`** (`RouteRegistrar`,
  `ProtectedRouteRegistrar`, `OrgScopedRouteRegistrar`) — a module's handler opts into routing purely
  by implementing the matching method.
- **Schema is not part of the module's Go files.** Migrations are staged via the SDD data-model flow or
  `make migrate-create name=...` and applied from the embedded `migrations/` FS — see the `migrations`
  rule.

## Initialization checklist (new module)

- [ ] Decide **module vs platform** (Decision 1). A domain → `internal/modules/<domain>/`; cross-cutting → `internal/platform/<concern>/`.
- [ ] Create the four layers `domain` / `app` / `ports` / `infra` from [assets/MODULE_SCAFFOLD.md](assets/MODULE_SCAFFOLD.md).
- [ ] Declare interfaces consumer-side (`app/ports.go`, the `ports` handler's `*AppService`).
- [ ] Add the `<domain>.go` wiring seam returning `*ports.Handler`.
- [ ] Pick the registrar by route shape (Decision 3); register in `cmd/api/main.go`.
- [ ] Add the schema via the data-model flow / `make migrate-create`; every `up` has a `down`.
- [ ] `make fmt && make vet && make lint && make test` green; `make test-integration` if Docker is up.

## Related skills

→ `go-design-patterns` rule (manual DI, constructor injection, functional options). → `go-naming`
rule (package/identifier conventions). → `go-lint` skill (`.golangci.yml`). → `go-ci` skill (the
GitHub Actions pipeline running these make targets). → `go-database` skill (the `infra/` repository
patterns). → the `migrations` rule (schema changes).

---

Upstream attribution: adapted from [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang) `golang-project-layout` (MIT, © samber).
