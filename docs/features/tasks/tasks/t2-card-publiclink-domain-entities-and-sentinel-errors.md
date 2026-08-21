---
id: T2
title: "Card + PublicLink domain entities and sentinel errors"
layer: "domain"
deps: []
acs: ["AC-03", "AC-14"]
files_hint:
  - "api/internal/modules/tasks/domain/"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T2 — Card + PublicLink domain entities and sentinel errors

## Why

Domain sentinels are the single source of validation truth every other layer maps errors against; derives from [data-model.md](../data-model.md) and [spec §5](../spec.md).

## What

Add `Card`, `PublicLink` structs and sentinel errors `ErrCardNotFound`, `ErrNameRequired`, `ErrCardFieldTooLong`, `ErrLinkNotFound`, `ErrLinkDisabled` to `api/internal/modules/tasks/domain/`, matching the `internal/modules/user/domain/` shape.

## Definition of Done

- [ ] unit tests cover ErrNameRequired (empty/whitespace-only name) and the 200/100-char length sentinels for both create and update paths
- [ ] lint + vet clean

## Notes

None.
