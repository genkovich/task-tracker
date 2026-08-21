---
id: T4
title: "Task use-cases accept details; state reads serve task detail (editor + token-scoped)"
layer: "app"
deps: ["T2", "T3"]
acs: ["TSK-01", "TSK-03", "TSK-05", "TSK-07", "TSK-12", "TSK-13", "TSK-14"]
files_hint: ["api/internal/modules/board/app/task_service.go", "api/internal/modules/board/app/state_service.go"]
owner: "genkovich"
estimate: "M"
status: "done"
---

# T4 — Task use-cases + task-detail reads

## Why

[spec.md](../spec.md) TSK-13 — головний ризик фічі: публічний детальний GET без перевірки
дошки токена перетворює один публічний лінк на читалку всіх задач продукту.

## What

- `TaskService.CreateTask/EditTask` приймають `domain.TaskDetails` замість голих
  title/assignee; broadcast лишається рівно один на мутацію.
- `StateService.GetTaskDetail(taskID)` — редакторські деталі.
- `StateService.GetPublicTaskDetail(token, taskID)` — резолвить токен у дошку, резолвить
  задачу разом із її дошкою і **зіставляє їх**; розбіжність віддається як
  `domain.ErrLinkNotFound`, тобто рівно те саме, що й невідомий токен.

## Definition of Done

- [ ] create/edit несуть нові поля й транслюють `board.state_changed` дошці задачі (TSK-14)
- [ ] `GetTaskDetail` повертає задачу + коментарі
- [ ] задача чужої дошки за токеном → та сама помилка, що й невідомий токен (TSK-13)
- [ ] відкликаний токен → та сама помилка, навіть якщо задача існує

## Notes

Помилка навмисно неінформативна: різні коди для «немає токена» і «є токен, але чужа
задача» самі по собі підтверджували б існування задачі.
