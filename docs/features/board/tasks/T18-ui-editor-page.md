---
id: T18
title: "Assemble the team-editor board page and register its route"
layer: "ui"
deps: ["T15", "T16", "T17"]
acs: []
files_hint: ["web/src/pages/board/", "web/src/routes.ts"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T18 — Assemble the team-editor board page and register its route

## Why

[sad.md §5](../sad.md): `pages/board/` "composes `features/board`, `features/public-link`" (SCR-01, SCR-04). `CLAUDE.md` §Architecture: "pages assemble slices and should not hold business logic themselves." [ux-flows.md](../ux-flows.md) SCR-01 is the product's default page (spec §2 — no accounts, board is the whole product).

## What

- `web/src/pages/board/` — lays out T15's columns/cards/quick-add, wires T16's edit modal to a card click, and mounts T17's public-link panel behind a "поділитись" action (SCR-01 → SCR-04).
- `web/src/routes.ts` — registers this page at the app's default route (currently a bare "Hello" dashboard placeholder, per `CLAUDE.md` — this task replaces it as the real product surface).

## Definition of Done

- [ ] page composes T15/T16/T17 with no fetch/mutation logic of its own — all data access goes through those features' `api`/`model` layers
- [ ] registered in `routes.ts`; `npm run dev` against a running API shows the board at the default, unauthenticated product route (ADR-0001: board has no login gate)
- [ ] manual check: creating, editing, moving, and deleting a task all work end-to-end against the real API (T1–T12)
- [ ] `npm run typecheck` clean

## Notes

Shares `web/src/routes.ts` with T19 — both edit the same file, so `implement` serializes this pair into one lane even though they're otherwise independent.
