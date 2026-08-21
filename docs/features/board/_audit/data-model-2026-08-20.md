---
status: Draft
owner: "genkovich"
updated_at: "2026-08-20"
---

# Data-model audit — board

## Prereqs

- `docs/features/board/spec.md` — present.
- `docs/features/board/sad.md` — present, ADR-0004 (fixed seeded columns), ADR-0003 (opaque DB token), ADR-0001/0002 read for context only (no persistence impact beyond what's modeled here).
- `docs/architecture-map.md` §Conventions (`reflects_commit: cfa5e7f`) — migration tool + naming + PK/audit-column conventions derived from here, corroborated against the live `api/migrations/` tree.

## Conventions derived (and followed)

| Topic | Source | Decision |
|---|---|---|
| Migration tool / naming | `architecture-map.md` §Conventions — `golang-migrate`, `<NNNNNN>_<verb>_<entity>.up.sql`/`.down.sql`, sequential 6-digit, head `000005` | Staged files use a feature-local 2-digit ordinal (`01_..06_`); `implement` assigns real `000006`–`000011` at promotion |
| PK strategy | `api/migrations/000002_create_users.up.sql:2`, `architecture-map.md` — app-generated UUID v7, no DB `SERIAL`/`gen_random_uuid()` | Every new table: `id UUID PRIMARY KEY`, app-generated |
| Audit columns | `users` table — `created_at`/`updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` | `boards`/`columns`/`public_links`: `created_at` only (no mutation path — columns/links are create-or-delete, never updated in place); `tasks`: both `created_at` and `updated_at` (title/assignee edits, AC-03) |
| Delete strategy | `user` module `Delete(ctx, id)` — hard delete, no `deleted_at` anywhere in repo | Hard delete for task removal (AC-06) and link revoke (AC-08/AC-11, per ADR-0003 "revoke = one DELETE") |
| Constraints (`CHECK`/triggers/DB `DEFAULT`) | No `CHECK` or trigger in any existing migration | None added — `title` non-empty (AC-02) enforced at app layer only, matching repo norm |
| String types | `VARCHAR(100)`/`VARCHAR(200)`/`VARCHAR(255)`/`TEXT` mixed by field, sized to purpose | `columns.name VARCHAR(100)`, `tasks.title`/`assignee VARCHAR(200)`, `public_links.token VARCHAR(64)` |

No divergence between `architecture-map.md` and the live `api/migrations/` tree was found — both agree.

## Decision confirmed with the user

ADR-0004 fixes the column set as seeded/non-editable but no upstream artifact (spec, sad, idea-brief — idea-brief §16 lists it as an unresolved open question) names the actual columns. Asked the user directly (data-model has no house style to default to here); confirmed: **To Do / In Progress / Done**, in that left-to-right order (`position` 0/1/2). Seeded in `04_seed_columns`.

## Staged migrations

All staged under `docs/features/board/migrations/` — **not** in the live `api/migrations/` tree:

| File | Purpose |
|---|---|
| `01_create_boards.up.sql` / `.down.sql` | `boards` table (singleton aggregate root) |
| `02_seed_board.up.sql` / `.down.sql` | Bootstrap seed — the one board row (deterministic UUID) |
| `03_create_columns.up.sql` / `.down.sql` | `columns` table + `idx_columns_board_id_position` |
| `04_seed_columns.up.sql` / `.down.sql` | Lookup seed — To Do / In Progress / Done (ADR-0004, confirmed above) |
| `05_create_tasks.up.sql` / `.down.sql` | `tasks` table + `idx_tasks_column_id` |
| `06_create_public_links.up.sql` / `.down.sql` | `public_links` table (ADR-0003) |

**Promote-time hint:** repo is sequential 6-digit, current head `000005` → these six files would land as `000006`–`000011` **if no other feature promotes first**; `implement` assigns the real numbers at promotion time, not this run.

## Drift detection

No existing `board` domain code in the repo (fresh scaffold — `architecture-map.md` §Constraints: "Продуктових фіч у репо ще немає"). Drift detection is **N/A** for this run — nothing to diff the schema against yet. Re-run `data-model --drift-only board` after `implement` lands the domain layer if the schema needs re-validating against it later.

## Self-check (4 mandatory)

- **Naming matches repo convention** — snake_case, plural table names (`boards`, `columns`, `tasks`, `public_links`), matching `users`. ✅
- **Down reversibility** — every `CREATE TABLE` has a matching `DROP TABLE`; every `CREATE INDEX` has a matching `DROP INDEX`; seed inserts have matching deletes. ✅
- **FK indexes** — `columns.board_id` covered by `idx_columns_board_id_position`; `tasks.column_id` covered by `idx_tasks_column_id`; `public_links.board_id` covered by its own `UNIQUE` constraint index. ✅
- **Convention adherence** — UUID v7 app-generated PKs, sized `VARCHAR`, `TIMESTAMPTZ DEFAULT now()` audit columns, no `CHECK`/triggers — all match repo norms, no deliberate divergence. ✅

## Open items / TBD

- None carried as `<!-- TBD -->` in `data-model.md` — the one open point (column names) was resolved above during this run.
- sad.md §11 flags the workshop backup question (spec §8 OQ-2) as still open — not a schema concern, no action needed here; noted for `implement`/`ship` to track.

## Next stage

`/sdd:api board` — board has a full contract surface (task CRUD, move, link issue/revoke, SSE broadcast) per sad.md §5, so this is not an N/A-skip case.
