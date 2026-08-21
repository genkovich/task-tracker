---
id: T11
title: "Build the task details modal (editor) and its read-only viewer twin"
layer: "ui"
deps: ["T9", "T10"]
acs: ["TSK-01", "TSK-02", "TSK-08", "TSK-09", "TSK-10", "TSK-12"]
files_hint: ["web/src/features/board/ui/TaskDetailsModal.tsx", "web/src/features/board/ui/TaskComments.tsx", "web/src/pages/board/ui/BoardPage.tsx", "web/src/pages/board-public/ui/BoardPublicPage.tsx", "web/src/features/board/ui/Column.tsx"]
owner: "genkovich"
estimate: "L"
status: "done"
---

# T11 — Детальна модалка

## Why

[ux-flows.md](../ux-flows.md) SCR-03/SCR-07: деталі задачі — це той самий екран, що й
редагування, і той самий вміст глядачеві, лише без полів.

## What

- `TaskDetailsModal.tsx` **замість** `EditTaskModal.tsx`: модалка тепер починається з
  мережевого читання деталей і несе пʼять полів плюс коментарі — це вже інший компонент,
  а не розширений старий. Delete-поведінку перенесено один-в-один.
- `TaskComments.tsx` — стрічка + форма додавання; поле автора предзаповнене іменем
  залогіненого (вільний текст, не акаунт).
- `readOnly`-режим модалки: той самий вміст як текст, без input/textarea/select, без форми
  коментаря й без жодного видалення (TSK-12).
- `Column` починає прокидати клік і в read-only (раніше за карткою глядача нічого не було).
- `BoardPage` і `BoardPublicPage` монтують відповідний режим.

## Definition of Done

- [ ] редакторська модалка вантажить деталі, редагує пʼять полів і зберігає
- [ ] порожня назва блокує збереження, як раніше; задовгий опис показує помилку з API
- [ ] коментар додається й видаляється зі стрічки; порожній блокується у формі
- [ ] глядацька модалка не містить жодного `input`/`textarea`/кнопки видалення (TSK-12)
- [ ] `EditTaskModal.tsx` і його тест видалені, посилань не лишилось
