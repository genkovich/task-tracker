# Tracker — board

> Status of every task in the epic. `implement` updates `done` as it commits each task.
> States: `todo` · `in_progress` · `blocked` · `review` · `done`.

| # | Task | Layer | Owner | Estimate | Blocked by | Status |
|---|---|---|---|---|---|---|
| T1 | Promote staged board migrations into the live migrations/ tree | migration | genkovich | S | — | done |
| T2 | Model board domain entities and sentinel errors | domain | genkovich | M | — | done |
| T3 | Implement Postgres repository for board, columns, tasks, public_links | infra | genkovich | L | T1, T2 | done |
| T4 | Implement in-process SSE hub with per-token connection registry | infra | genkovich | M | — | done |
| T5 | Implement task use-cases: create, edit, move, delete | app | genkovich | L | T2, T3 | done |
| T6 | Implement public-link and board-state use-cases | app | genkovich | M | T2, T3 | done |
| T7 | Add HTTP handlers for task CRUD + move, with rate limiting on create | ports | genkovich | M | T5 | done |
| T8 | Add HTTP handlers for board state and public-link issue/revoke | ports | genkovich | M | T6 | done |
| T9 | Add SSE endpoints for team-editor and public-viewer live updates | ports | genkovich | M | T4, T6 | done |
| T10 | Add public-viewer HTTP handlers for read-only board access by token | ports | genkovich | S | T6 | done |
| T11 | Wire the board module into main.go and register routes | wiring | genkovich | S | T7, T8, T9, T10 | done |
| T12 | Write integration tests for concurrent move and link-revocation invariants | tests | genkovich | M | T3, T5, T9 | done |
| T13 | Build the board feature's typed API client and SSE subscription hook | ui | genkovich | M | — | done |
| T14 | Build drag-and-drop board state with optimistic updates | ui | genkovich | L | T13 | done |
| T15 | Build task card, column, and quick-add UI components | ui | genkovich | M | T14 | done |
| T16 | Build the edit-task modal (title, assignee, save, delete) | ui | genkovich | M | T14 | done |
| T17 | Build the public-link panel feature (issue/revoke) | ui | genkovich | S | — | done |
| T18 | Assemble the team-editor board page and register its route | ui | genkovich | M | T15, T16, T17 | done |
| T19 | Assemble the public read-only board page and its unavailable-link state | ui | genkovich | M | T13, T15 | done |

**Total:** 19 tasks, ~11–12 person-days (~2.5 weeks). All 19 committed on `chore/design-canon-claude-md`.

## Known deviations / follow-ups for review

- **T11 wiring**: board's ports handlers (T7–T10) register absolute paths (e.g. `/api/v1/board`)
  rather than paths relative to a mount point, so `board.New(...).Handler` is mounted directly on
  the root chi router in `main.go` instead of being passed through `server.New(...)`'s
  `RouteRegistrar` opts like other modules (nesting it there double-prefixes routes to
  `/api/v1/api/v1/...`). Functionally verified (manual boot + curl smoke test), but worth a
  `/sdd:review` look at whether to instead make the board handlers mount-relative for consistency
  with the `user`/`auth` module pattern.
- **Pre-existing, unrelated failure**: `TestMigrationsIntegration_UsersSchema` in
  `internal/platform/database` fails on `main` independent of any board change (reproduced in
  isolation) — a down-migration ordering/idempotency bug from the original scaffold, tracked here
  for visibility, not fixed as part of this feature.
