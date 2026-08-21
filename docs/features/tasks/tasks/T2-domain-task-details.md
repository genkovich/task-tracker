---
id: T2
title: "Extend the task domain with description, priority, due date and the Comment entity"
layer: "domain"
deps: []
acs: ["TSK-01", "TSK-02", "TSK-03", "TSK-04", "TSK-05", "TSK-07", "TSK-09"]
files_hint: ["api/internal/modules/board/domain/domain.go"]
owner: "genkovich"
estimate: "M"
status: "done"
---

# T2 — Extend the task domain + Comment entity

## Why

[spec.md](../spec.md) TSK-02/TSK-04/TSK-09 ставлять межі, які мусять ловитись до будь-якого
запису в БД — інакше вони випливуть непрозорою помилкою драйвера як 500 замість 422, як це
вже було з довгим `title` (коментар у `domain.go`).

## What

- `Task` отримує `Description string`, `Priority`, `DueDate *time.Time`.
- `Priority` — власний тип-рядок із трьома константами й `ParsePriority` (порожній рядок →
  `medium`, TSK-03).
- `TaskDetails` — параметр-обʼєкт для конструктора й сетера, щоб `NewTask` не виріс до
  шести позиційних аргументів.
- `Comment` + `NewComment(taskID, author, body)`.
- Нові сентинели: `ErrDescriptionTooLong`, `ErrPriorityInvalid`, `ErrCommentAuthorRequired`,
  `ErrCommentAuthorTooLong`, `ErrCommentBodyRequired`, `ErrCommentBodyTooLong`,
  `ErrCommentNotFound`.

## Definition of Done

- [ ] опис рівно 4000 рун приймається, 4001 — відхиляється (TSK-02)
- [ ] пріоритет поза набором відхиляється; порожній = `medium` (TSK-03/TSK-04)
- [ ] `nil` дедлайн легальний і знімається сетером (TSK-05/TSK-07)
- [ ] порожній або >200 автор і порожній або >2000 текст коментаря відхиляються (TSK-09)
- [ ] жодних framework-імпортів у пакеті

## Notes

Межі рахуються рунами (`utf8.RuneCountInString`), як для `title` — 4000 кириличних
символів мусять означати 4000, а не 8000 байтів.
