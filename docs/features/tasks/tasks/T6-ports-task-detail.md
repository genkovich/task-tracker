---
id: T6
title: "Extend the task HTTP surface: detail GET and the new fields on create/edit"
layer: "ports"
deps: ["T4"]
acs: ["TSK-01", "TSK-02", "TSK-03", "TSK-04", "TSK-05", "TSK-07"]
files_hint: ["api/internal/modules/board/ports/task_handler.go", "api/internal/modules/board/ports/dto.go", "api/internal/modules/board/ports/errors.go"]
owner: "genkovich"
estimate: "M"
status: "done"
---

# T6 — Task detail GET + нові поля на create/edit

## Why

[openapi.yaml](../contracts/openapi.yaml): `GET /api/v1/tasks/{taskId}` віддає `TaskDetail`,
а `TaskCreate`/`TaskUpdate` приймають description/priority/due_date. PATCH-семантика
навмисно не змінюється — це і далі не merge-patch, `title` обовʼязковий.

## What

- `GET /tasks/{taskId}` у `TaskHandler`.
- DTO: `Task` тепер із description/priority/due_date; `TaskCardResponse` — те, що їде в
  стані дошки; `TaskDetailResponse`, `CommentResponse`.
- `due_date` на дроті — рядок `YYYY-MM-DD` або `null`, ніколи не RFC3339: дедлайн задають
  днем, і час доби в ньому був би вигаданою точністю.
- `mapTaskError` доростає `task.description_too_long` і `task.priority_invalid`.

## Definition of Done

- [ ] `GET /tasks/{taskId}` віддає задачу + коментарі, 404 `task.not_found` на невідому
- [ ] create/edit приймають і повертають нові поля
- [ ] кривий `due_date` → 400 `validation.invalid_due_date`, не 500
- [ ] опис >4000 → 422 `task.description_too_long`; невідомий пріоритет → 422
      `task.priority_invalid`
- [ ] стан дошки на дроті не має поля `description`
