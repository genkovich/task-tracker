---
id: T9
title: "Extend the web API layer with the task detail and comment endpoints"
layer: "ui"
deps: ["T6", "T7", "T8"]
acs: []
files_hint: ["web/src/features/board/api/types.ts", "web/src/features/board/api/boardApi.ts"]
owner: "genkovich"
estimate: "S"
status: "done"
---

# T9 — Web API layer

## Why

[openapi.yaml](../contracts/openapi.yaml) розводить дві форми задачі: `TaskCard` у стані
дошки (без опису, з `has_description`/`comment_count`) і `Task` у деталях та відповідях
create/edit (з описом). Типи web мусять розводити їх так само, інакше картка почне
покладатись на поле, якого в її відповіді немає.

## What

- `types.ts`: `TaskPriority`, `Task` (картка на дошці), `TaskRecord` (повний рядок),
  `TaskComment`, `TaskDetail`; `TaskCreate`/`TaskUpdate` доростають нових полів.
- `boardApi`: `getTask`, `getPublicTask`, `addComment`, `deleteComment`; create/edit
  передають нові поля.

## Definition of Done

- [ ] типи дзеркалять контракт (у `Task` немає `description`, у `TaskRecord` є)
- [ ] юніт-тести пінять форму запитів нових методів і кодування токена в шляху
- [ ] `npm run typecheck` чистий
