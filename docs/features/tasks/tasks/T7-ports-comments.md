---
id: T7
title: "Add the comment routes (list, add, delete)"
layer: "ports"
deps: ["T5"]
acs: ["TSK-08", "TSK-09", "TSK-10"]
files_hint: ["api/internal/modules/board/ports/comment_handler.go"]
owner: "genkovich"
estimate: "S"
status: "done"
---

# T7 — Comment routes

## Why

[openapi.yaml](../contracts/openapi.yaml): `GET|POST /api/v1/tasks/{taskId}/comments`,
`DELETE /api/v1/tasks/{taskId}/comments/{commentId}`. Окремий хендлер, бо `task_handler.go`
уже несе CRUD + move + rate limit, і дописування туди ще трьох роутів зробило б його
найбільшим файлом модуля.

## What

`CommentHandler` із власним `RegisterRoutes`, змонтований у `board.go` поруч із рештою
редакторських роутів. `mapCommentError` мапить нові сентинели на коди контракту.

## Definition of Done

- [ ] POST → 201 зі створеним коментарем; GET → найстаріші зверху; DELETE → 204
- [ ] неіснуюча задача → 404 `task.not_found` (не 500 від FK)
- [ ] неіснуючий коментар → 404 `comment.not_found`
- [ ] порожній/задовгий автор чи текст → 422 з окремим кодом на кожен випадок (TSK-09)
- [ ] не-UUID у шляху → 400, а не паніка
