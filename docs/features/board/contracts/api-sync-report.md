---
status: Draft
owner: "genkovich"
reviewers: []
updated_at: "2026-08-20"
feature_size: M
---

# API sync report — board

Derived from `data-model.md` (typed shape) + `sad.md` §6 sequences (error branches, async
actors) + `spec.md` §4/§5 (endpoint list, observable outcomes) → `contracts/openapi.yaml` +
`contracts/events.md`. `target_surfaces: [backend-service, web-frontend]` (sad.md frontmatter)
picks the OpenAPI/HTTP form.

## Section A — field-origins table

| schema_path | origin | confidence |
|---|---|---|
| Column.id | data-model.md → `columns.id` | high |
| Column.name | data-model.md → `columns.name` (VARCHAR(100)) | high |
| Column.position | data-model.md → `columns.position` (SMALLINT) | high |
| Column.tasks | derived (nesting convention — render board state groups tasks per column, data-model.md §Access patterns `tasks`) | high |
| Task.id | data-model.md → `tasks.id` | high |
| Task.column_id | data-model.md → `tasks.column_id` | high |
| Task.title | data-model.md → `tasks.title` (VARCHAR(200)) | high |
| Task.assignee | data-model.md → `tasks.assignee` (VARCHAR(200), NULL) | high |
| Task.created_at | data-model.md → `tasks.created_at` | high |
| Task.updated_at | data-model.md → `tasks.updated_at` | high |
| TaskCreate.title | data-model.md → `tasks.title`, non-empty enforced at app layer (AC-02 note) | high |
| TaskCreate.assignee | data-model.md → `tasks.assignee` | high |
| TaskUpdate.title / .assignee | same columns, mirrored for AC-03 edit | high |
| TaskMove.column_id | data-model.md → `tasks.column_id` (target of the move `UPDATE`) | high |
| PublicLink.token | data-model.md → `public_links.token` (VARCHAR(64), UNIQUE) | high |
| PublicLink.created_at | data-model.md → `public_links.created_at` | high |
| BoardState.public_link | derived (SCR-04 needs link presence/absence on first render — ux-flows.md SCR-04 — folded into `GET /board` instead of a second round-trip) | medium |
| PublicBoardState.columns | data-model.md → `boards` aggregate render (viewer path, no `public_link` field — AC-09/AC-10 scope) | high |
| Error.code (`task.title_required`) | inferred from AC-02 (spec.md), no existing `board` error registry yet — see checklist point 2 | low |
| Error.code (`task.not_found`) | inferred convention, mirrors `user.not_found` pattern in `api/internal/modules/user/ports/errors.go` | medium |
| Error.code (`board.column_not_found`) | inferred from AC-05 (server-side safety net; client-side drag-and-drop never sends an invalid target) | low |
| Error.code (`board.link_already_active`) | inferred from `public_links.board_id` UNIQUE constraint (data-model.md) + AC-07 precondition; no sad.md §6 branch draws this | low |
| Error.code (`board.link_not_found`) | inferred — no AC covers "revoke with no active link"; reasonable 404 for a singleton resource | low |
| Error.code (`board.link_invalid`) | data-model.md `public_links` lookup-miss ↔ AC-11 | high |
| Error.code (`task.rate_limited`) | spec.md §6.1 abuse case, exact threshold (≤30/min) quoted verbatim | medium |
| board.sse `events.md` channel | sad.md §6 Flow 1 (`API->>Other: розсилає подію "стан змінився"`) + ADR-0002 | high |

## Section B — drift findings (4-point checklist)

1. **Endpoint ↔ data-model** *(core)* — ✓. Every endpoint reads/writes ≥1 entity:
   `getBoard`/`getPublicBoard` → `columns`+`tasks`; `createTask`/`editTask`/`deleteTask` →
   `tasks`; `moveTask` → `tasks.column_id`; `issuePublicLink`/`revokePublicLink` →
   `public_links`; the two SSE endpoints carry no entity (signal-only, `events.md`).

2. **Error code ↔ repo error definition** *(core)* — no `board` Go module exists yet (this is
   a pre-implementation SDD stage; `find api/internal/modules -iname board` is empty). The
   repo's error-registry **form** is established by `user`/`auth`
   (`internal/modules/<mod>/ports/errors.go` — an `errorMap` of `{domain sentinel, apperr.Error}`
   pairs, dotted `module.error_name` codes, matched via `errors.Is`). **No error registry found
   for `board` yet — the 8 codes above are the contract's proposal; `sdd:tasks`/`sdd:implement`
   reconcile them against `board/ports/errors.go` once it's written.** Flagged, not failed, per
   the legal "no registry yet" exception.

3. **Validation ↔ constraint** *(core)* — ✓. `title` `maxLength: 200` / `minLength: 1` ↔
   `tasks.title VARCHAR(200) NOT NULL` (non-empty enforced at app layer per data-model.md note,
   mirrored as `minLength: 1`); `assignee` `maxLength: 200`, nullable ↔ `tasks.assignee
   VARCHAR(200) NULL`; `Column.name` `maxLength: 100` ↔ `columns.name VARCHAR(100)`;
   `PublicLink.token` `maxLength: 64` ↔ `public_links.token VARCHAR(64)`. No conflicts found.

4. **OpenAPI ↔ sequence** *(supporting)* — partial ✓, one gap noted, not failed (per SKILL.md
   step 1: "Absent → note the gap ... still generate"):
   - Flow 1 (move + SSE broadcast) → `POST /tasks/{taskId}/move` + `Idempotency-Key` +
     `board.sse` channel. Matches.
   - Flow 2 (concurrent move, AC-05b) → same `moveTask` operation, last-write-wins — no new
     branch needed (data-model.md already documents "no version/lock column"). Matches.
   - Flow 3 (public view + revoked token, AC-09/AC-10/AC-11) →
     `getPublicBoard`/`streamPublicBoardEvents` `404 board.link_invalid` (`alt`-branch) +
     AC-10's "action rejected" is enforced by **not exposing** any mutating operation under
     `/api/v1/public/{token}/...` at all (no route exists to reject — the read-only surface
     itself is the enforcement mechanism, stronger than a runtime check). Matches.
   - **Gap:** AC-01, AC-02, AC-03, AC-06, AC-07, AC-08 (create/edit/delete task, issue/revoke
     link) have **no** corresponding `sad.md` §6 diagram — only 3 of ~9 ACs are sequenced.
     Their error responses (`task.title_required`, `task.not_found`, `board.link_already_active`,
     `board.link_not_found`) are derived from `spec.md` §5 AC text and the data-model.md
     constraints alone, not from a drawn `alt`-branch. **Save-as-OQ, owner: `sequences`, due:
     before `sdd:tasks board`** — `sdd:sequences board` should add these flows (or the team
     accepts the spec-only derivation as sufficient for an M-size feature and drops this OQ
     explicitly).

**Core points 1 and 3: ✓ clean. Core point 2: flagged (no registry yet — legal, not blocking).
Supporting point 4: 1 gap, routed as Save-as-OQ to `sequences` per the back-feed rule (step 7).**
No point failed outright; total flags = 2 (point 2's "no registry" note + point 4's sequence
gap), under the ≥3-flags pause threshold — proceeding to write.

## Deviations from defaults

- **No `BearerAuth` scheme, global `security: []` everywhere.** Board deliberately skips the
  repo's existing auth-middleware (ADR-0001 context, spec §6.1: "без входу в систему"). This
  is not an ADR-mandated override of the api skill's default per se — it's a direct
  consequence of ADR-0001 (backend API + web SPA, no accounts) and spec §6.1's explicit
  two-capability model ("може редагувати" / "може лише дивитись", enforced by network
  surface — team routes vs public-by-token routes — not by any token/session check).
- **Idempotency-Key required only on `moveTask`** — the only operation whose `sad.md` §6 flow
  (Flow 1) draws an async actor (SSE broadcast to `Other`). `createTask`/`editTask`/
  `deleteTask`/`issuePublicLink`/`revokePublicLink` also broadcast per ADR-0002, but no §6
  diagram shows it for them — not required per the skill's derivation rule (step 5), flagged
  here for visibility rather than silently applied everywhere.

## Open questions carried forward

- [ ] Sequence gap (Section B, point 4) — 6 ACs (AC-01/02/03/06/07/08) have no `sad.md` §6
      flow; their error responses are spec-derived only. Owner: `sequences`, due: before
      `sdd:tasks board`.
- [ ] `board.link_already_active` (409 on double-issue) and `board.link_not_found` (404 on
      revoke-with-nothing-active) are inferred from the data model's UNIQUE constraint and
      singleton-resource convention, not from an explicit AC. Owner: `genkovich`, due: before
      `sdd:tasks board` — confirm these are the intended behaviors, not silently accepted.
