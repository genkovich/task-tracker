---
id: T9
title: "Add SSE endpoints for team-editor and public-viewer live updates"
layer: "ports"
deps: ["T4", "T6"]
acs: ["AC-11"]
files_hint: ["api/internal/modules/board/ports/sse_handler.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T9 — Add SSE endpoints for team-editor and public-viewer live updates

## Why

[openapi.yaml](../contracts/openapi.yaml) fixes `GET /api/v1/board/events` and `GET /api/v1/public/{token}/events` as `text/event-stream`. [sad.md §6 Flow 3](../sad.md) and [events.md](../contracts/events.md) require a revoked token's connections to close synchronously with the revoke, not just reject new subscribe attempts.

## What

- `sse_handler.go` — `streamBoardEvents` registers the connection with the T4 hub under the team-editor bucket (no token); `streamPublicBoardEvents` first validates the token via `app.StateService.GetPublicBoardState` (404 `board.link_invalid` per openapi.yaml if invalid, AC-11), then registers under that token.
- Both write `board.state_changed` per [events.md](../contracts/events.md) on every hub broadcast, using `http.Flusher` to push immediately (no response buffering — matches the Caddy `flush_interval -1` requirement noted in sad.md §7).

## Definition of Done

- [ ] handler test: a public SSE request with an invalid/revoked token returns 404 `board.link_invalid` before registering any connection
- [ ] handler test: a valid public SSE connection receives a `board.state_changed` event when the hub broadcasts
- [ ] handler test: calling `DELETE /api/v1/board/public-link` (T8) on an active token closes every SSE connection registered under that token synchronously (AC-11) — this is the acceptance test sad.md §11 flags as currently unmeasured by any formal AC; it lives here and in T12
- [ ] lint + vet clean

## Notes

Depends on both T4 (the hub mechanics) and T6 (token validation) — do not duplicate token-validity logic here; delegate to `app.StateService`.
