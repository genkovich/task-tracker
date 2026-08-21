# Changelog

## [Unreleased]

### boards — кілька дощок + живий дашборд (2026-08-21)

Продукт більше не «рівно одна дошка»: дашборд показує всі дошки (назва +
кількість задач) і створює нову — з трьома фіксованими колонками, транзакційно.
Кожна дошка живе на `/board/{boardId}` зі своїм публічним лінком і своїм
SSE-каналом.

- API: `GET/POST /api/v1/boards`, `GET /api/v1/boards/{boardId}`,
  пер-дошкові `.../events` і `.../public-link`; `POST /api/v1/tasks` тепер
  вимагає `board_id` (колонку й далі обирає сервер). Старі шляхи
  `/api/v1/board*` видалені без сумісності.
- SSE hub скоуплений дошкою: події дошки A не будять глядачів дошки B;
  broadcast мутацій адресується дошці задачі.
- Міграція `000012_add_board_name`: `boards.name`, seed-борд стає
  «Дошка команди» (перша дошка).
- Web: живий дашборд (список + діалог створення → редірект на нову дошку),
  `/board` → перша дошка або дашборд, post-login → `/dashboard`.
- E2E smoke: dashboard → create board → нова дошка → задача в To Do.

Артефакти: `docs/features/boards/` (spec BRD-01…BRD-08, data-model delta,
contracts delta, staged migration); інваріант CONTEXT.md оновлено — «дошок
багато; кожна має рівно три фіксовані колонки».

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
