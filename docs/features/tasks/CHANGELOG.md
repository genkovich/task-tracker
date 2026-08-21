# Changelog — tasks

## tasks — shared kanban board with a public read-only share link

**What:** A team can now manage work on a shared 3-column kanban board (To Do / In Progress /
Done) — create, edit, move (drag-and-drop, mouse and touch), and delete cards, with changes
pushed live to every open tab via SSE. No accounts, no per-card ownership: anyone with access to
the board can edit any card. A one-click **public share link** exposes a read-only, live-updating
view of the same board to anyone with the link — no login required — and can be regenerated
(instantly revoking the old link) or disabled at any time.

**Why:** Lets a small team or an ad-hoc group coordinate a task list without spinning up an
account system, and lets that same board be shared read-only with stakeholders who shouldn't be
able to edit it. See [spec.md](../spec.md) §1/§2. Key decisions: SSE push over polling for board
sync ([ADR-0001](../adr/0001-use-sse-push-for-board-sync.md)), server-timestamp last-write-wins
with no version check — last write always wins, no conflict UI
([ADR-0002](../adr/0002-server-timestamp-last-write-wins.md)), and an opaque `crypto/rand` token
with a `disabled_at` flag for the public link rather than a UUID (which would leak a creation
timestamp) ([ADR-0003](../adr/0003-opaque-token-with-disabled-flag-for-public-link.md)).

**How to use:**
- Board: `GET/POST /api/v1/cards`, `PATCH /api/v1/cards/{id}`, `PATCH /api/v1/cards/{id}/column`,
  `DELETE /api/v1/cards/{id}`, live updates via `GET /api/v1/cards/events` (SSE).
- Public share link: `POST /api/v1/public-links` (generate/regenerate), `GET /api/v1/public-links`
  (active link), `DELETE /api/v1/public-links/{id}` (disable); the shared view resolves via
  `GET /api/v1/public/boards/{token}` + `GET /api/v1/public/boards/{token}/events` (SSE) — both
  unauthenticated and rate-limited at the higher, high-traffic tier. See
  [openapi.yaml](../contracts/openapi.yaml) for the full contract.
- Frontend: `/board` (the editable board) and `/b/:token` (the public read-only view).

**Operational notes:**
- Migrations: `000006_create_cards` + `000007_create_public_links` — applied on deploy, both
  revert cleanly (`down.sql` for each).
- Feature flag / config: none — the module is wired unconditionally in `cmd/api/main.go`.
- Rollback: `migrate down` twice (public_links, then cards) + revert the deploy. No data
  migration/backfill involved.

**Acceptance criteria delivered:** AC-01…AC-15 (full CRUD + column move, drag-and-drop incl.
touch, live SSE sync across tabs, public share-link generate/regenerate/disable with byte-identical
404 for a disabled or never-valid token, and the AC-15 concurrent delete-wins race) — see
[spec.md](../spec.md) §5 for the full list. Independent review: PASS, 13/13 findings resolved —
see [`_review/review-2026-08-21.md`](../_review/review-2026-08-21.md).
