---
id: T10
title: "Add public-viewer HTTP handlers for read-only board access by token"
layer: "ports"
deps: ["T6"]
acs: ["AC-09", "AC-10", "AC-11"]
files_hint: ["api/internal/modules/board/ports/public_handler.go"]
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T10 — Add public-viewer HTTP handlers for read-only board access by token

## Why

[openapi.yaml](../contracts/openapi.yaml) fixes `GET /api/v1/public/{token}/board`. AC-10 (spec §5) — a viewer cannot mutate — is satisfied structurally: this router registers only the one read-only route, no create/edit/move/delete path exists under `/api/v1/public/{token}/...`.

## What

`public_handler.go` — `getPublicBoard` calls `app.StateService.GetPublicBoardState(token)` (T6), returns the `PublicBoardState` DTO (AC-09) or 404 `board.link_invalid` for an unknown/revoked token (AC-11).

## Definition of Done

- [ ] handler test: valid token → 200 with `PublicBoardState` (columns + tasks, no `public_link` field) (AC-09)
- [ ] handler test: revoked/unknown token → 404 `board.link_invalid` (AC-11)
- [ ] confirm (by reading the router registration in T11) that no mutating route exists under this token-scoped path — the DoD assertion for AC-10 is the absence of such a route, not a runtime check
- [ ] lint + vet clean

## Notes

Smallest ports task — one handler, one route, reuses `app.StateService` entirely from T6.
