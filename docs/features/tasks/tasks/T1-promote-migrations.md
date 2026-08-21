---
id: T1
title: "Promote the staged task-details and task_comments migrations into the live tree"
layer: "migration"
deps: []
acs: []
files_hint: ["docs/features/tasks/migrations/", "api/migrations/"]
owner: "genkovich"
estimate: "S"
status: "done"
---

# T1 — Promote the staged migrations into the live tree

## Why

[data-model.md](../data-model.md): три нові колонки в `tasks` і таблиця `task_comments`.
Робочі міграції лежать у `api/migrations/` (не `api/db/migrations/`, як припускав план
фічі), головою на момент роботи була `000012_add_board_name`.

## What

- `000013_add_task_details.{up,down}.sql` — `description TEXT NOT NULL DEFAULT ''`,
  `priority VARCHAR(10) NOT NULL DEFAULT 'medium'` з CHECK на набір значень, `due_date DATE`.
- `000014_create_task_comments.{up,down}.sql` — таблиця + індекс `(task_id, created_at)`,
  FK `ON DELETE CASCADE`.
- Staged-копії під `docs/features/tasks/migrations/` — конвенція репо (як board/boards).

## Definition of Done

- [ ] обидві пари накочуються поверх `000012` і відкочуються у зворотному порядку
- [ ] існуючі рядки `tasks` отримують `''` і `'medium'`, не NULL
- [ ] CHECK відхиляє значення пріоритету поза `low/medium/high`
- [ ] staged-копії побайтово збігаються з робочими

## Notes

DEFAULT-и лишаються постійно (на відміну від `boards.name` у 000012) — вони структурно
чесні для задачі без деталей, тож 3-крокового патерну не треба.
