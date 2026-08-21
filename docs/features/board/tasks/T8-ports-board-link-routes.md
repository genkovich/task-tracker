---
id: T8
title: "Add HTTP handlers for board state and public-link issue/revoke"
layer: "ports"
deps: ["T6"]
acs: ["AC-07", "AC-08"]
files_hint: ["api/internal/modules/board/ports/board_handler.go", "api/internal/modules/board/ports/link_handler.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T8 — Add HTTP handlers for board state and public-link issue/revoke

## Why

[openapi.yaml](../contracts/openapi.yaml) fixes `GET /api/v1/board` (`BoardState` incl. `public_link`), `POST/DELETE /api/v1/board/public-link` — the team-editor's view of link state (AC-07, AC-08).

## What

- `board_handler.go` — `GET /api/v1/board` returning `app.StateService.GetBoardState()` as the `BoardState` DTO.
- `link_handler.go` — `POST /api/v1/board/public-link` → 201 `PublicLink` or 409 `board.link_already_active` (AC-07); `DELETE /api/v1/board/public-link` → 204 or 404 `board.link_not_found` (AC-08).

## Definition of Done

- [ ] handler test: `GET /board` returns columns + tasks + `public_link: null` when none active, and the link object when one exists
- [ ] handler test: `POST /board/public-link` → 201 with a `token` when none active (AC-07); 409 `board.link_already_active` when one already exists
- [ ] handler test: `DELETE /board/public-link` → 204 when a link is active (AC-08); 404 `board.link_not_found` otherwise
- [ ] response bodies match openapi.yaml's `BoardState`/`PublicLink`/`Error` schemas
- [ ] lint + vet clean

## Notes

Shares no files with T7 (different handler files) — can be implemented in parallel once T6 lands.
