---
id: T11
title: "Wire the board module into main.go and register routes"
layer: "wiring"
deps: ["T7", "T8", "T9", "T10"]
acs: []
files_hint: ["api/internal/modules/board/board.go", "api/cmd/api/main.go"]
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T11 — Wire the board module into main.go and register routes

## Why

Every module in this repo has a top-level `<domain>.go` with a single `New(...)` constructor wiring its own layers, called from `cmd/api/main.go` (`CLAUDE.md` §Architecture). ADR-0001 (Accepted) fixes board as unauthenticated — no `authMW`.

## What

- `api/internal/modules/board/board.go` — `New(repo ports.Repository, broadcaster ports.Broadcaster, ...) *ports.Handler`, wiring T5/T6 app services into the T7/T8/T9/T10 handlers.
- `api/cmd/api/main.go` — construct the Postgres repo (T3) + SSE hub (T4), call `board.New(...)`, register the returned handler as a `RouteRegistrar` (public — no `ProtectedRouteRegistrar`, per ADR-0001).

## Definition of Done

- [ ] `make -C api run` boots against the dockerized Postgres with no wiring error
- [ ] `curl localhost:<port>/api/v1/board` returns 200 with the seeded board's empty-task columns
- [ ] `go vet ./...` and `make -C api lint` clean
- [ ] no `authMW` wraps any `/api/v1/board*`, `/api/v1/tasks*`, or `/api/v1/public/*` route

## Notes

Small on purpose — this is pure composition, no new logic. If wiring reveals an interface mismatch between T5/T6 and T7/T8/T9/T10, fix the mismatch in whichever earlier task owns that interface, not here.
