---
id: T10
title: "Wire the tasks module: routes, rate-limit + timeout exemptions"
layer: "wiring"
deps: ["T7", "T8", "T9"]
acs: []
files_hint:
  - "api/internal/modules/tasks/tasks.go"
  - "api/cmd/api/main.go"
  - "api/internal/server/server.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T10 — Wire the tasks module: routes, rate-limit + timeout exemptions

## Why

Registers the module and applies the two deployment-relevant exemptions sad.md §7/§8 call out; derives from [sad §7](../sad.md) and [sad §8](../sad.md).

## What

Wire `tasks.New(db)` into `server.New(...)` in `api/cmd/api/main.go`; add the SSE routes' timeout exemption and confirm the existing 60 req/min-per-IP rate limiter applies as documented in `api/internal/server/server.go`.

## Definition of Done

- [ ] tasks.New(db) is registered in cmd/api/main.go via server.New(...), all card + public-link + SSE routes reachable with security: [], the two SSE routes are exempted from the 30s request timeout (sad.md §7/§8)
- [ ] lint + vet clean

## Notes

Hard Rule: no auth middleware on any `tasks` route (spec §3 Non-goals) — do not wrap these routes in `authmw.Middleware`.
