---
id: T3
title: "PostgresCardRepository (CRUD + move)"
layer: "infra"
deps: ["T1", "T2"]
acs: ["AC-01", "AC-02", "AC-10", "AC-13"]
files_hint:
  - "api/internal/modules/tasks/infra/card_repository.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T3 — PostgresCardRepository (CRUD + move)

## Why

Card persistence for all card mutations and reads; derives from [data-model.md](../data-model.md) §cards and [sad §5](../sad.md).

## What

Implement `PostgresCardRepository` in `api/internal/modules/tasks/infra/` satisfying a `domain.CardRepository` interface — create, list-all (ordered by column then created_at), update, move (column_status + updated_at), delete.

## Definition of Done

- [ ] integration tests (testcontainers) cover create/list/update/move/delete against a real Postgres instance
- [ ] lint + vet clean

## Notes

Shares no files with T4 — the two infra tasks parallelize once T1+T2 land.
