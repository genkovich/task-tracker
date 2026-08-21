---
id: T1
title: "Promote staged board migrations into the live migrations/ tree"
layer: "migration"
deps: []
acs: []
files_hint: ["docs/features/board/migrations/*"]
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T1 — Promote staged board migrations into the live migrations/ tree

## Why

The six paired migrations already staged under `docs/features/board/migrations/` ([data-model.md](../data-model.md)) define the whole schema this feature needs — `boards`, seeded `columns` (ADR-0004), `tasks`, `public_links`. They need renumbering into the repo's live sequence before any code can run against them (`.claude/rules/migrations.md`, current head `000005`).

## What

Copy the 6 staged pairs into `api/migrations/` as `000006`..`000011`, renaming files to the repo's `<NNNNNN>_<verb>_<entity>.up/.down.sql` convention while keeping their SQL unchanged:

- `01_create_boards` → `000006_create_boards`
- `02_seed_board` → `000007_seed_board`
- `03_create_columns` → `000008_create_columns`
- `04_seed_columns` → `000009_seed_columns`
- `05_create_tasks` → `000010_create_tasks`
- `06_create_public_links` → `000011_create_public_links`

## Definition of Done

- [ ] `make -C api migrate-go-up` applies all six migrations against the dockerized Postgres with no error
- [ ] `make -C api migrate-go-down` reverts them cleanly in reverse order, leaving no orphaned table
- [ ] the staged files under `docs/features/board/migrations/` are left untouched (they are the design record, not the live source)
- [ ] file names/numbering follow `.claude/rules/migrations.md` exactly

## Notes

Always the first item in the DAG — `layer: migration` tasks are serialized by `implement`, and T3/T5/T12 all need a real schema to run integration tests against.
