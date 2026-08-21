---
id: T5
title: "Implement the comment use-cases (add, delete) with board-scoped broadcast"
layer: "app"
deps: ["T2", "T3"]
acs: ["TSK-08", "TSK-09", "TSK-10", "TSK-14"]
files_hint: ["api/internal/modules/board/app/comment_service.go"]
owner: "genkovich"
estimate: "S"
status: "done"
---

# T5 — Comment use-cases

## Why

[spec.md](../spec.md) TSK-08/TSK-10: коментар змінює те, що видно на дошці (лічильник на
картці), тож він мусить будити ті самі live-зʼєднання, що й будь-яка інша мутація (TSK-14).

## What

`CommentService` поруч із `TaskService`: власний файл, ті самі два порти (`Repository`,
`Broadcaster`).

- `AddComment(taskID, author, body)` — валідує через `domain.NewComment` **до** запису,
  повертає створений коментар, транслює дошці задачі.
- `DeleteComment(commentID)` — видаляє, транслює тій самій дошці.

## Definition of Done

- [ ] валідний коментар збережений і транслюється рівно один раз у бакет своєї дошки
- [ ] порожній/задовгий автор чи текст відхиляється до будь-якого запису (TSK-09)
- [ ] видалення транслює так само (TSK-10)
- [ ] жодного прямого імпорту `infra` — лише порти
