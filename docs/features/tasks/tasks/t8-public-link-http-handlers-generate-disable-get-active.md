---
id: T8
title: "Public-link HTTP handlers (generate/disable/get-active)"
layer: "ports"
deps: ["T4", "T6"]
acs: ["AC-04", "AC-09"]
files_hint:
  - "api/internal/modules/tasks/ports/public_link_handler.go"
owner: "genkovich"
estimate: "S"
status: "todo"
---

# T8 — Public-link HTTP handlers (generate/disable/get-active)

## Why

Exposes public-link operations per [contracts/openapi.yaml](../contracts/openapi.yaml) `public-links` tag.

## What

Implement `POST /api/v1/public-links`, `GET /api/v1/public-links/active`, `POST /api/v1/public-links/{id}/disable` in `api/internal/modules/tasks/ports/public_link_handler.go`.

## Definition of Done

- [ ] handler tests assert the exact status/body contract for generatePublicLink / disablePublicLink / getActivePublicLink
- [ ] handler responses match `contracts/openapi.yaml` exactly for every listed AC
- [ ] lint + vet clean

## Notes

None.
