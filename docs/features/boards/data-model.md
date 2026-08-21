---
status: Draft
owner: "genkovich"
reviewers: []
updated_at: "2026-08-21"
feature_size: "M"
---

# Data model — boards (дельта до `docs/features/board/data-model.md`)

Схема фічі board уже мульти-дошкова по конструкції: `columns.board_id` FK,
`tasks.column_id` → колонка визначає дошку, `public_links.board_id` UNIQUE — одна
на дошку. Дельта цієї фічі — одне поле.

## Зміни

### `boards` — нова колонка `name`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `name` | VARCHAR(200) | NOT NULL | назва дошки з дашборда (BRD-02); непорожня, ≤200 символів — валідація в app-шарі, симетрично `tasks.title` |

Міграція: single-step `ADD COLUMN ... DEFAULT 'Дошка команди'` + `DROP DEFAULT`.
Default тут — backfill для єдиного існуючого seed-рядка (він стає «першою дошкою»,
BRD-07), а не постійний бізнес-default: після backfill default знімається, name
завжди задає застосунок.

### Незмінне

- `columns`, `tasks`, `public_links` — без змін; три колонки кожної нової дошки
  вставляє застосунок транзакційно разом із рядком `boards` (BRD-02), як
  seed-міграція робила для першої дошки.
- Seed-рядки `000007_seed_board` / `000009_seed_columns` лишаються — це «перша
  дошка».

## Access patterns

- Дашборд (BRD-01): `SELECT b.id, b.name, b.created_at, COUNT(t.id) FROM boards b
  LEFT JOIN columns c ... LEFT JOIN tasks t ... GROUP BY b.id ORDER BY b.created_at` —
  таблиця дощок мала (десятки), нових індексів не треба; JOIN йде по існуючих
  `idx_columns_board_id_position` / `idx_tasks_column_id`.
- Стан дошки (BRD-04): існуючий `GetBoardState(boardID)` + попередній lookup рядка
  `boards` (дає 404 для неіснуючої дошки).

## Staged migrations

`migrations/01_add_board_name.{up,down}.sql` — копія робочої
`api/migrations/000012_add_board_name.{up,down}.sql`.
