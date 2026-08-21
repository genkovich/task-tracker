## Summary

Ships the `tasks` feature: a shared kanban board (To Do / In Progress / Done) with drag-and-drop
cards (mouse + touch), live sync across open tabs via SSE, and a one-click public read-only share
link. No accounts — anyone with board access can edit any card. See
[spec.md](docs/features/tasks/spec.md).

## Acceptance criteria

- AC-01 — a board member sees the board with a card in a column ✓
- AC-02 — a board member views the board and its cards ✓
- AC-03 — creating a card with an empty name is rejected ✓
- AC-04 — a board member drags a card to a different column ✓
- AC-05 — a disabled public link resolves the same byte-identical 404 as a never-valid one (no board-existence leak) ✓
- AC-06 — a viewer on a public link sees a read-only board (no edit affordances) ✓
- AC-07 — two members dragging the same card near-simultaneously — last write wins, no conflict UI ✓
- AC-08 — a card moved to another column reflects live in every open tab via SSE ✓
- AC-09 — generating the first public link when none is active ✓
- AC-10 — deleting a card that's no longer relevant ✓
- AC-11 — a failed column-move request reverts the optimistic UI change and toasts ✓
- AC-12 — a public viewer's open tab reacts live to board changes via SSE ✓
- AC-13 — editing a card from the board ✓
- AC-14 — editing a card with invalid data is rejected ✓
- AC-15 — a card deleted mid-drag by another member: delete wins, the move silently no-ops (no error) ✓

## Design

- Spec: `docs/features/tasks/spec.md`
- Architecture: `docs/features/tasks/sad.md`
- Decisions: `docs/features/tasks/adr/0001-use-sse-push-for-board-sync.md`,
  `0002-server-timestamp-last-write-wins.md`,
  `0003-opaque-token-with-disabled-flag-for-public-link.md`
- Data model + migrations: `docs/features/tasks/data-model.md` (promoted as
  `api/migrations/000006_create_cards`, `000007_create_public_links`)
- API: `docs/features/tasks/contracts/openapi.yaml`
- Review record: `docs/features/tasks/_review/review-2026-08-21.md` (PASS, 13/13 findings resolved)

## Tasks (SDD-Task trailers)

T1…T16 (`597ea54`..`7d3cf5a`) — migrations, domain, repositories, services, HTTP/SSE handlers,
module wiring, board/card-form/public-link/public-board UI, and the end-to-end smoke test — plus
the post-review fix pass (`b347da2`, `SDD-Task: review-fix`) and this review record
(`2c53e5a`, `SDD-Task: review-record`). Full history: `git log --grep SDD-Task -- docs/features/tasks`.

## Verification

- Unit (Go): `go test ./...` — all green.
- Integration (Go, Docker-backed): `go test -tags integration ./...` — green at `-p 1`/`-p 2`
  (a single `-p 4` run flaked once from testcontainers resource contention, not reproduced).
- Unit (web): `npx vitest run` — 139/139 passed across 21 files.
- Typecheck (web): `npm run typecheck` — clean.
- Lint + vet (Go): `go vet ./...`, `go vet -tags integration ./...`, `gofmt -l .` — clean.
- E2E (Playwright): `--project=smoke` (12 specs, full two-context board lifecycle) and
  `--project=mobile` (real touch pointer-events drag-and-drop on a Pixel 7 viewport) — both green.
- Ran the feature live (`docker compose up`, hit the real API): created a card (AC-01/02),
  rejected an empty-name create (AC-03), moved a card via `PATCH .../column` and confirmed the
  full response body (`column_status`/`created_at`/`updated_at` — the exact field the review's
  finding #1 fixed) (AC-04), generated a public link and fetched the board through it, then
  confirmed a never-valid token and an unmatched route return byte-identical 404s (AC-05), and
  deleted a card then attempted to move it — confirmed the 404 the client's delete-wins handling
  (AC-15) relies on. Verified data cleared afterward via `docker compose down`.

## Operational notes

- Migrations: `000006_create_cards`, `000007_create_public_links` — applied on deploy, both
  revert cleanly.
- Feature flag / config: none — the module is wired unconditionally in `cmd/api/main.go`.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
