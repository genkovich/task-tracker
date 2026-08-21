---
id: T4
title: "Implement in-process SSE hub with per-token connection registry"
layer: "infra"
deps: []
acs: []
files_hint: ["api/internal/modules/board/infra/sse_hub.go", "api/internal/modules/board/ports/broadcaster.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T4 — Implement in-process SSE hub with per-token connection registry

## Why

ADR-0002 (Accepted) chose in-process SSE over a message bus. [sad.md §6](../sad.md) and [events.md](../contracts/events.md) require the hub to both broadcast `board.state_changed.v1` to every live connection and let a revoke synchronously close exactly the connections registered under one public-link token (sad.md §6 "Відкликання закриває вже відкриті SSE-з'єднання").

## What

- `api/internal/modules/board/ports/broadcaster.go` — the `Broadcaster` interface `app` (T5/T6) calls after a mutation: `Broadcast()` (team-editor + all viewers) and a way to close a token's connections on revoke.
- `api/internal/modules/board/infra/sse_hub.go` — in-process registry: team-editor connections (no token) + a `map[token][]connection` for public-viewer connections; `Register`/`Unregister`/`Broadcast`/`CloseToken(token)`.

## Definition of Done

- [ ] unit test: `Broadcast()` delivers `board.state_changed.v1` (events.md shape: `event_id`, `event_type`, `version`, `occurred_at`) to every registered connection, team and public alike
- [ ] unit test: `CloseToken(token)` closes exactly the connections registered under that token, leaving others open
- [ ] concurrent-safe: register/unregister/broadcast from multiple goroutines without a data race (`go test -race`)
- [ ] lint + vet clean

## Notes

Purely additive plumbing — no direct spec AC, but it's what makes T9's AC-11 close-on-revoke test (T12) possible. Stays a single-process registry per ADR-0002/sad.md §11 (accepted debt: doesn't survive horizontal scaling of `api`).
