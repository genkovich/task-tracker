---
id: T2
title: "Model board domain entities and sentinel errors"
layer: "domain"
deps: []
acs: ["AC-02"]
files_hint: ["api/internal/modules/board/domain/"]
owner: "genkovich"
estimate: "M"
status: "todo"
---

# T2 — Model board domain entities and sentinel errors

## Why

`board` is the repo's first product module — [sad.md §5](../sad.md) calls for a `domain/` layer with `Board`, `Column` (fixed set, ADR-0004), `Task` entities plus sentinel errors, no framework imports, following `.claude/rules/go-*.md`.

## What

`api/internal/modules/board/domain/`:
- `Board`, `Column`, `Task`, `PublicLink` structs matching [data-model.md](../data-model.md) (UUID v7 ids, `column_id` as the task's only status field — no separate status field, per data-model.md's `tasks` note).
- A `Task` constructor/setter that enforces a non-empty title (AC-02: [spec §5](../spec.md)), returning a sentinel error (e.g. `ErrTitleRequired`).
- Sentinel errors this module needs downstream: title-required, task-not-found, column-not-found, link-not-found, link-already-active — named so `ports/errors.go` (T7/T8/T10) can map them 1:1 to `apperr.Error` codes already fixed in [openapi.yaml](../contracts/openapi.yaml) (`task.title_required`, `task.not_found`, `board.column_not_found`, `board.link_not_found`, `board.link_already_active`).

## Definition of Done

- [ ] unit tests: constructing/updating a `Task` with an empty title returns `ErrTitleRequired` (AC-02); a non-empty title succeeds
- [ ] no import of `chi`, `pgx`, or any HTTP/DB package in `domain/`
- [ ] `go vet` and `gofmt` clean (gofmt runs automatically via the repo's post-edit hook)

## Notes

Keep `Column` genuinely fixed here — no field or method that implies future CRUD (ADR-0004, sad.md §11 accepted debt row).
