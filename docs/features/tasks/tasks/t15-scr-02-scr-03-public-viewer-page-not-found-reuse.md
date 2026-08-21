---
id: T15
title: "SCR-02/SCR-03 Public viewer page + not-found reuse"
layer: "ui"
deps: []
acs: ["AC-05", "AC-06", "AC-08", "AC-12"]
files_hint:
  - "web/src/pages/public-board/"
  - "web/src/routes.ts"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T15 — SCR-02/SCR-03 Public viewer page + not-found reuse

## Why

Builds the viewer surface per [screens.md](../screens.md) SCR-02/SCR-03 and [ux-flows.md](../ux-flows.md) US-04.

## What

Build `web/src/pages/public-board/` — resolves the token from the URL, fetches `GET /api/v1/public/boards/{token}`, subscribes to its SSE stream, renders cards read-only with a "View only" label (no edit affordances), and on a 404 or a `link.disabled` SSE event, navigates to the existing `pages/not-found/ui/NotFoundPage.tsx` route via the app's own `route("*", ...)` catch-all — no new not-found page. Wire the public board route into `web/src/routes.ts`.

## Definition of Done

- [ ] component tests cover SCR-02's loading/empty/default/error states (read-only, no edit affordances per AC-06) and assert an SSE link-disabled event or a 404 navigates to the existing NotFoundPage route (SCR-03) without a manual refresh
- [ ] lint + vet clean

## Notes

No dependency on any backend task — builds directly against `contracts/openapi.yaml` (already locked) and can run fully in parallel with the whole backend chain and with T12/T13/T14.
