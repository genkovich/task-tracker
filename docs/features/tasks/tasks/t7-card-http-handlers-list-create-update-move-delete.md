---
id: T7
title: "Card HTTP handlers (list/create/update/move/delete)"
layer: "ports"
deps: ["T3", "T5"]
acs: ["AC-01", "AC-02", "AC-03", "AC-07", "AC-10", "AC-11", "AC-13", "AC-14"]
files_hint:
  - "api/internal/modules/tasks/ports/card_handler.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T7 — Card HTTP handlers (list/create/update/move/delete)

## Why

Exposes card operations per [contracts/openapi.yaml](../contracts/openapi.yaml) `cards` tag, unauthenticated per spec §3 Non-goals.

## What

Implement `GET/POST /api/v1/cards`, `PATCH/DELETE /api/v1/cards/{id}`, `PATCH /api/v1/cards/{id}/column` in `api/internal/modules/tasks/ports/card_handler.go`, registered with no auth middleware.

## Definition of Done

- [ ] handler tests assert the exact status/body contract in contracts/openapi.yaml for every listed AC, no auth middleware attached to any route (spec §3 Non-goals)
- [ ] handler responses match `contracts/openapi.yaml` exactly for every listed AC
- [ ] lint + vet clean

## Notes

None.
