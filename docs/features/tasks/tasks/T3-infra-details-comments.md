---
id: T3
title: "Persist task details and comments; carry the card fields in board state"
layer: "infra"
deps: ["T1", "T2"]
acs: ["TSK-01", "TSK-05", "TSK-08", "TSK-10", "TSK-11"]
files_hint: ["api/internal/modules/board/ports/repo.go", "api/internal/modules/board/infra/postgres_repo.go"]
owner: "genkovich"
estimate: "L"
status: "done"
---

# T3 — Persist details and comments; card fields in board state

## Why

[data-model.md](../data-model.md) §Access patterns: стан дошки не тягне описи, а лічильник
коментарів мусить лишатись одним агрегатом на колонку. Публічні деталі (TSK-13) вимагають,
щоб пошук задачі одразу повертав її дошку.

## What

- `ports.TaskListItem` — задача в стані дошки: без `Description`, з `HasDescription` і
  `CommentCount`. `ColumnState.Tasks` міняє тип на нього.
- `ports.TaskDetail` — повна задача + її коментарі.
- Нові методи `ports.Repository`: `TaskByID` (повертає задачу + `boardID`),
  `ListComments`, `InsertComment`, `DeleteComment`.
- `tasksForColumn` — `LEFT JOIN task_comments ... GROUP BY` замість запиту на задачу.
- Insert/Update задачі переносять нові поля.

## Definition of Done

- [ ] задача round-trip-иться з описом/пріоритетом/дедлайном
- [ ] стан дошки несе `has_description` і `comment_count` і НЕ несе тіла опису
- [ ] коментарі вставляються, читаються найстарішими зверху й видаляються
- [ ] видалення задачі прибирає її коментарі (TSK-11, каскад схеми)
- [ ] `TaskByID` віддає `boardID`, щоб виклик міг зіставити з дошкою токена
- [ ] неіснуюча задача при вставці коментаря → `domain.ErrTaskNotFound`, не сира FK-помилка
