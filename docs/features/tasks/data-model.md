---
status: Draft
owner: "genkovich"
reviewers: []
updated_at: "2026-08-21"
feature_size: "M"
---

# Data model — tasks (дельта до `docs/features/board/data-model.md`)

Дошки, колонки й публічні лінки не змінюються взагалі. Дельта — три нові колонки в
`tasks` і одна нова таблиця `task_comments`.

## Зміни

### `tasks` — три нові колонки

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `description` | TEXT | NOT NULL DEFAULT `''` | опис задачі (TSK-01); межа 4000 символів тримається в домені, не в схемі — `VARCHAR(4000)` рахує символи так само, але зміна межі стала б міграцією |
| `priority` | VARCHAR(10) | NOT NULL DEFAULT `'medium'`, CHECK IN (`'low'`,`'medium'`,`'high'`) | пріоритет (TSK-03); домен — первинний валідатор (як для `title`), CHECK — друга лінія оборони, дозволена `.claude/rules/migrations.md` |
| `due_date` | DATE | NULL | дедлайн (TSK-05); саме DATE, не TIMESTAMPTZ — дедлайн задають днем, і час доби в ньому був би вигаданою точністю з часовими зонами на додачу |

Обидва DEFAULT лишаються постійно, а не знімаються після backfill (на відміну від
`boards.name`, міграція 000012): `''` і `'medium'` структурно чесні для задачі, створеної
без деталей — «опису ще немає», «звичайний пріоритет». Це той самий випадок, що
`status DEFAULT 'draft'` у `.claude/rules/migrations.md` §Allowed, тож single-step ALTER
достатній: жодна з трьох колонок не вимагає 3-крокового патерну, бо існуючі рядки
отримують семантично правильне значення.

### `task_comments` — нова таблиця

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK | UUIDv7, генерується застосунком (`uuid.Must(uuid.NewV7())`) |
| `task_id` | UUID | NOT NULL, FK → `tasks(id)` ON DELETE CASCADE | каскад — це TSK-11: коментарі не переживають свою задачу; альтернатива (чистка в застосунку) лишала б сиріт при будь-якому іншому шляху видалення |
| `author` | VARCHAR(200) | NOT NULL | вільний текст, **без FK на `users`**: акаунтів на рівні дошки немає (ADR-0001 фічі board), форма лише предзаповнює поле іменем залогіненого; ширина симетрична `tasks.assignee` |
| `body` | VARCHAR(2000) | NOT NULL | текст коментаря (TSK-09); непорожність — у домені |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | immutable-таблиця: `updated_at` немає, бо коментар не редагується (spec §3) |

Індекс: `idx_task_comments_task_id_created_at (task_id, created_at)` — покриває і FK
(перша колонка), і єдиний реальний запит «коментарі задачі за часом» (TSK-08).

### Незмінне

- `boards`, `columns`, `public_links` — без змін.
- `tasks.column_id` лишається єдиним полем статусу (інваріант CONTEXT.md).
- Порядок карток у колонці — за `created_at`, як і був.

## Access patterns

- **Стан дошки** (SCR-01/SCR-05): до існуючого читання задач додається
  `description <> ''` як прапорець `has_description` і `COUNT` коментарів. Лічильник —
  один `LEFT JOIN ... GROUP BY` по колонці, а не запит на задачу: інакше дошка на 100
  задач дала б 100 додаткових запитів. **Тіло опису в стані дошки не їде** (spec §6) —
  тільки прапорець.
- **Деталі задачі** (SCR-03/SCR-07): рядок `tasks` за id + `SELECT ... FROM task_comments
  WHERE task_id = $1 ORDER BY created_at` — обидва по PK/новому індексу.
- **Публічні деталі** (TSK-13): токен → `board_id` (існуючий `public_links.token` UNIQUE),
  далі задача резолвиться разом зі своєю дошкою (`tasks → columns.board_id`) і звіряється
  з дошкою токена. Звірка обовʼязково в тому ж читанні, а не після — інакше це IDOR між
  дошками.
- **Видалення задачі**: без змін у коді — каскад прибирає коментарі на рівні схеми.

## Staged migrations

- `migrations/01_add_task_details.{up,down}.sql` — копія робочої
  `api/migrations/000013_add_task_details.{up,down}.sql`.
- `migrations/02_create_task_comments.{up,down}.sql` — копія робочої
  `api/migrations/000014_create_task_comments.{up,down}.sql`.

Нумерація: на момент роботи головою `api/migrations/` була `000012_add_board_name`
(план фічі казав «000013+» — збіглось). Робочі міграції лежать у `api/migrations/`, а не
в `api/db/migrations/`, як припускав план.

## Test fixtures

Без фабрик — інлайнове створення рядків у тесті, як у решті модуля. Задачі з деталями
створюються через `domain.NewTask` + репозиторій, коментарі — через `domain.NewComment`;
seed-дошка й колонки й далі приходять із міграцій 000007/000009.
