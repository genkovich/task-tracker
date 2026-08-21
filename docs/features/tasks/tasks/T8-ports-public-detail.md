---
id: T8
title: "Serve public task details in the 300/min tier and wire the new handlers"
layer: "ports"
deps: ["T4", "T6", "T7"]
acs: ["TSK-12", "TSK-13"]
files_hint: ["api/internal/modules/board/ports/public_handler.go", "api/internal/modules/board/board.go"]
owner: "genkovich"
estimate: "M"
status: "done"
---

# T8 — Public task detail + wiring

## Why

[spec.md](../spec.md) §6.1: глядацькі роути живуть у тирі 300 запитів/хв — аудиторія
воркшопу відкриває картки з телефонів за одним венью-Wi-Fi, і 60/хв вичерпалось би на
першій хвилині. Тир уже існує (`HighTrafficRouteRegistrar`, Task 1); новий роут мусить
потрапити саме туди, а не в загальну групу.

## What

- `GET /public/{token}/tasks/{taskId}` у `PublicHandler`, з тими самими заголовками, що й
  публічний стан дошки: `Cache-Control: no-store`, `X-Robots-Tag: noindex, nofollow`.
- `board.go`: `CommentHandler` у редакторську групу, публічний детальний GET — у
  `RegisterHighTrafficRoutes` разом із публічним станом дошки.

## Definition of Done

- [ ] деталі віддаються за чинним токеном і не містять ані `board_id`, ані `public_link`
- [ ] задача чужої дошки → 404 `board.link_invalid` (TSK-13)
- [ ] роут у тирі 300/хв: 61-й запит за хвилину ще проходить, 301-й — ні
- [ ] мутуючих роутів під публічним токеном не існує як маршрутів (TSK-12)
