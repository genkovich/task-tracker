---
id: T10
title: "Show the card markers: priority, due date, description, comment count"
layer: "ui"
deps: ["T9"]
acs: ["TSK-03", "TSK-05", "TSK-06", "TSK-08"]
files_hint: ["web/src/features/board/ui/TaskCard.tsx"]
owner: "genkovich"
estimate: "M"
status: "done"
---

# T10 — Маркери на картці

## Why

[ux-flows.md](../ux-flows.md): колонка мусить лишитись скануванням за пів секунди, тож
картка показує маркери, а не вміст. TSK-06 вимагає, щоб протермінований дедлайн було видно
не читаючи дати.

## What

Маркери йдуть у `TaskCardVisual` — спільну візуальну частину картки та її drag-двійника
(Task 1), тож привид під курсором показує те саме.

- пріоритет — кольорова крапка: `status-blocked` (high) / `status-warning` (medium) /
  `status-todo` (low); нових кольорових токенів фіча не вводить
- дедлайн — бейдж із датою; протермінований несе `destructive`
- «є опис» — іконка, лише коли `has_description`
- коментарі — іконка з числом, лише коли `comment_count > 0`

## Definition of Done

- [ ] маркер пріоритету відповідає пріоритету задачі (TSK-03)
- [ ] майбутній дедлайн — звичайний бейдж, минулий — destructive (TSK-05/TSK-06)
- [ ] іконка опису й лічильник зʼявляються лише за наявності (TSK-01/TSK-08)
- [ ] ті самі маркери в drag-привиді
- [ ] тільки токени, обидві теми притомні
