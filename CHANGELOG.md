# Changelog

## [Unreleased]

### board — спільна дошка з публічним лінком-переглядом (2026-08-21)

Команда веде задачі на спільній kanban-дошці (To Do / In Progress / Done) і ділиться
живим станом через публічний лінк `/b/{token}` — перегляд без акаунта, редагування ні.

- Створення / редагування / видалення задач, drag-and-drop між колонками з optimistic UI.
- Live-оновлення всім відкритим вкладкам через SSE (`event: board.state_changed`) з
  heartbeat і refetch на reconnect.
- Публічний лінк: видача / відкликання; відкликаний лінк одразу віддає 404 і закриває
  SSE-стріми глядачів; `X-Robots-Tag: noindex` на API і SPA-шляху.
- Ліміти й межі: rate limit по XFF-клієнту, title/assignee ≤200 → 422, повний Task у
  відповідях PATCH/move.

Артефакти: `docs/features/board/` (spec §5 AC-01…AC-11, sad, ADR-0001…0003, openapi,
data-model, test-plan, tasks T1–T19) · рев'ю: `_review/review-2026-08-21.md` — два
незалежні рев'ювери, дві фікс-хвилі (25 комітів), verification pass → **PASS**.
Верифікація живцем: повний смок create→move→share→revoke→SSE на локальному стеку.

Deferred (spec §8): e2e-through-UI набір + CI-джоба; секрет редакторських роутів
(accepted risk ADR-0001); кастомні board-метрики.
