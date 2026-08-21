---
id: T4
title: "PostgresPublicLinkRepository (generate/disable/resolve)"
layer: "infra"
deps: ["T1", "T2"]
acs: ["AC-04", "AC-09"]
files_hint:
  - "api/internal/modules/tasks/infra/public_link_repository.go"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T4 — PostgresPublicLinkRepository (generate/disable/resolve)

## Why

Public-link persistence backing ADR-0003's opaque-token + disabled_at design; derives from [data-model.md](../data-model.md) §public_links and [ADR-0003](../adr/0003-opaque-token-with-disabled-flag-for-public-link.md).

## What

Implement `PostgresPublicLinkRepository` in `api/internal/modules/tasks/infra/` — insert (auto-disabling any active row first, in one transaction), resolve-by-token (WHERE token = $1 AND disabled_at IS NULL), get-active, disable.

## Definition of Done

- [ ] integration tests cover generate (auto-disabling a prior active link), resolve-by-token, and disable
- [ ] lint + vet clean

## Notes

Shares no files with T3 — the two infra tasks parallelize once T1+T2 land.
