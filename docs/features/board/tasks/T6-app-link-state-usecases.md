---
id: T6
title: "Implement public-link and board-state use-cases"
layer: "app"
deps: ["T2", "T3"]
acs: ["AC-07", "AC-08", "AC-09", "AC-11"]
files_hint: ["api/internal/modules/board/app/link_service.go", "api/internal/modules/board/app/state_service.go"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T6 — Implement public-link and board-state use-cases

## Why

[sad.md §5](../sad.md) building-block view: `BoardService` also owns `IssueLink/RevokeLink/GetState`. [ADR-0003](../adr/0003-opaque-db-stored-public-link-token.md) fixes the token as opaque and DB-stored; [data-model.md](../data-model.md) enforces at most one active link per board via a UNIQUE constraint.

## What

`api/internal/modules/board/app/link_service.go`:
- `IssuePublicLink()` — rejects if a link is already active (AC-07 precondition; maps to the `board.link_already_active` 409 in [openapi.yaml](../contracts/openapi.yaml)); generates an opaque token (ADR-0003) and persists it.
- `RevokePublicLink()` — deletes the active link (AC-08) and calls `Broadcaster.CloseToken(token)` (T4) so already-open SSE connections stop immediately (sad.md §6 Flow 3, AC-11).

`api/internal/modules/board/app/state_service.go`:
- `GetBoardState()` — team-editor view: columns + tasks + current public link (or none).
- `GetPublicBoardState(token)` — viewer view by token (AC-09); returns the not-found sentinel for an unknown/revoked token (AC-11).

## Definition of Done

- [ ] unit tests: issuing a link when none is active succeeds (AC-07); issuing when one is already active returns the already-active sentinel, no second row written
- [ ] unit test: revoking the active link deletes it and calls the fake broadcaster's `CloseToken` with the correct token (AC-08, AC-11)
- [ ] unit test: `GetPublicBoardState` returns the board for a valid token (AC-09) and the not-found sentinel for an unknown one (AC-11)
- [ ] lint + vet clean

## Notes

`GetPublicBoardState` must not expose team-editor-only fields (e.g. no second link) — matches `PublicBoardState` in openapi.yaml, which deliberately omits `public_link`.
