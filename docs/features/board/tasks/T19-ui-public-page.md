---
id: T19
title: "Assemble the public read-only board page and its unavailable-link state"
layer: "ui"
deps: ["T13", "T15"]
acs: ["AC-09", "AC-10", "AC-11"]
files_hint: ["web/src/pages/board-public/", "web/src/routes.ts"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T19 — Assemble the public read-only board page and its unavailable-link state

## Why

[ux-flows.md](../ux-flows.md) SCR-05 (public board, read-only) and SCR-06 (link unavailable) — [sad.md §5](../sad.md): `pages/board-public/` composes T13's public fetch + T15's column/card components in a read-only mode.

## What

- `web/src/pages/board-public/` — fetches `GET /api/v1/public/{token}/board` via T13's client, renders T15's `Column`/`TaskCard` in a read-only variant (no drag handlers, no click-to-edit, no quick-add — AC-10 satisfied by simply not rendering those affordances, matching the backend's structural omission in T10); subscribes to `GET /api/v1/public/{token}/events` for live updates (AC-09).
- On a 404 `board.link_invalid` response, renders the SCR-06 "лінк недоступний" state instead of the board (AC-11).
- `web/src/routes.ts` — registers this page at the public-link path (e.g. `/b/:token`, matching the `token`-scoped API paths).

## Definition of Done

- [ ] component test: a valid token renders the board read-only — no drag/edit/delete controls present in the DOM (AC-09, AC-10)
- [ ] component test: an invalid/revoked token renders the SCR-06 unavailable state, not the board (AC-11)
- [ ] manual check: revoking a link while this page is open (from another tab's editor view) makes the open public page show the unavailable state without a manual refresh, via the SSE-close behavior from T9
- [ ] `npm run typecheck` clean

## Notes

Shares `web/src/routes.ts` with T18 — same file, serialized lane. Reuses T15's presentational components in a read-only prop mode rather than forking new card/column components — keep it that way; a second copy would drift from the editor view over time.
