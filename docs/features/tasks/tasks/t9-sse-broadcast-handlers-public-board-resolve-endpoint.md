---
id: T9
title: "SSE broadcast handlers + public board resolve endpoint"
layer: "ports"
deps: ["T4", "T6", "T8"]
acs: ["AC-05", "AC-06", "AC-08", "AC-12"]
files_hint:
  - "api/internal/modules/tasks/ports/sse_handler.go"
  - "api/internal/modules/tasks/ports/public_board_handler.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T9 — SSE broadcast handlers + public board resolve endpoint

## Why

Exposes the SSE push endpoints (ADR-0001) and the token-gated public board read, including the byte-identical not-found behavior AC-05 requires; derives from [contracts/openapi.yaml](../contracts/openapi.yaml) `public` tag.

## What

Implement `GET /api/v1/cards/events`, `GET /api/v1/public/boards/{token}`, `GET /api/v1/public/boards/{token}/events` in `api/internal/modules/tasks/ports/sse_handler.go` + `public_board_handler.go` — the not-found path returns the shared `common.not_found` code, never a tasks-specific one.

## Definition of Done

- [ ] integration test asserts a disabled/never-valid token returns the byte-identical common.not_found body as an unmatched route (AC-05), and an SSE subscriber receives a pushed event within the freshness NFR window
- [ ] handler responses match `contracts/openapi.yaml` exactly for every listed AC
- [ ] lint + vet clean

## Notes

Depends on T8 because the disable-triggers-broadcast wiring (AC-12) needs the disable handler's shape settled first, even though the two don't share a file.
