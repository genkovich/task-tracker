---
id: T1
title: "Promote staged cards + public_links migrations"
layer: "migration"
deps: []
acs: []
files_hint:
  - "docs/features/tasks/migrations/01_create_cards.up.sql"
  - "docs/features/tasks/migrations/01_create_cards.down.sql"
  - "docs/features/tasks/migrations/02_create_public_links.up.sql"
  - "docs/features/tasks/migrations/02_create_public_links.down.sql"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T1 — Promote staged cards + public_links migrations

## Why

Provides the `cards` and `public_links` tables every other task reads/writes; derives from [data-model.md](../data-model.md).

## What

Promote `docs/features/tasks/migrations/01_create_cards.{up,down}.sql` and `02_create_public_links.{up,down}.sql` into the live `api/migrations/` tree, assigning the real next sequence number (`000006`, `000007` at promotion time — re-check in case another feature landed first).

## Definition of Done

- [ ] the promoted migration pair applies and reverts cleanly against a local Postgres
- [ ] the promoted migration applies and reverts cleanly (`make -C api migrate-up` / `migrate-down` or the repo's equivalent)
- [ ] lint + vet clean

## Notes

Serialized by `implement` with any other migration task (none here — this feature has exactly one).
