---
id: T17
title: "Build the public-link panel feature (issue/revoke)"
layer: "ui"
deps: []
acs: ["AC-07", "AC-08"]
files_hint: ["web/src/features/public-link/"]
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T17 — Build the public-link panel feature (issue/revoke)

## Why

[sad.md §5](../sad.md) building-block view calls out `features/public-link/` as its own slice (api/model/ui) — SCR-04 in [ux-flows.md](../ux-flows.md): "Показує стан лінка (є / немає), дії «отримати» і «відкликати»".

## What

`web/src/features/public-link/`:
- `api/` — `issuePublicLink`/`revokePublicLink` calls against `POST`/`DELETE /api/v1/board/public-link`, on the same `shared/api/client.ts` pattern as T13.
- `model/` — link state: none / active(token).
- `ui/` — panel component reusing `shared/ui` Popover, Button, Badge: no-link state shows an "отримати лінк" action that displays the returned URL (AC-07); active-link state shows the URL plus a "відкликати" action that clears back to no-link on success (AC-08).

## Definition of Done

- [ ] component test: issuing a link from the no-link state renders the returned URL (AC-07)
- [ ] component test: issuing while a link already exists surfaces the 409 as a visible error, panel stays in the active-link state
- [ ] component test: revoking from the active-link state clears the panel back to no-link (AC-08)
- [ ] reuses `shared/ui` Popover/Button/Badge only — no new primitive
- [ ] `npm run typecheck` clean

## Notes

Independent of T13 — writes its own small api layer since it calls a disjoint pair of endpoints; can be built in parallel with the entire `features/board/` line.
