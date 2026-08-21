---
id: T6
title: "PublicLink service + in-process board-event broadcaster"
layer: "app"
deps: ["T2"]
acs: ["AC-04", "AC-09"]
files_hint:
  - "api/internal/modules/tasks/app/public_link_service.go"
  - "api/internal/modules/tasks/app/broadcaster.go"
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T6 — PublicLink service + in-process board-event broadcaster

## Why

The public-link business logic plus the in-process pub/sub broadcaster ADR-0001 requires; derives from [ADR-0001](../adr/0001-use-sse-push-for-board-sync.md) and [sad §5](../sad.md).

## What

Implement the public-link service and a small in-process `Broadcaster` (Publish/Subscribe over Go channels) in `api/internal/modules/tasks/app/` — GenerateLink, DisableLink, ResolvePublicLink, plus every card/link mutation calling `Broadcaster.Publish`.

## Definition of Done

- [ ] unit tests cover generate-replaces-active-link, disable-is-idempotent, and Broadcaster fan-out to N subscriber channels
- [ ] lint + vet clean

## Notes

Can start as soon as T2 lands, in parallel with T3/T4/T5.
