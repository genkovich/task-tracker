---
id: T13
title: "Build the board feature's typed API client and SSE subscription hook"
layer: "ui"
deps: []
acs: []
files_hint: ["web/src/features/board/api/"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T13 — Build the board feature's typed API client and SSE subscription hook

## Why

`CLAUDE.md` §Architecture: "API access goes through the typed client in `src/shared/api/client.ts`". [sad.md §5](../sad.md) building-block view puts `features/board/api/` in charge of both the typed calls and the SSE subscription (ADR-0002).

## What

`web/src/features/board/api/`:
- Typed calls against [openapi.yaml](../contracts/openapi.yaml): `getBoard`, `createTask`, `editTask`, `moveTask` (with a generated `Idempotency-Key`), `deleteTask` — built on the existing `shared/api/client.ts` (errors surfaced via `showApiError`).
- A hook wrapping `EventSource` against `GET /api/v1/board/events` per [events.md](../contracts/events.md): on any `board.state_changed` message, triggers a `getBoard` refetch; relies on the browser's built-in auto-reconnect, no manual retry logic.

## Definition of Done

- [ ] unit tests (mocked fetch) cover each typed call's request shape and error mapping
- [ ] unit test for the SSE hook: a simulated `board.state_changed` message triggers exactly one refetch
- [ ] no new HTTP client introduced — reuses `shared/api/client.ts`
- [ ] `npm run typecheck` clean

## Notes

Pure API-layer work against an already-fixed contract — no dependency on the backend tasks being done, though real end-to-end testing needs T11 running.
