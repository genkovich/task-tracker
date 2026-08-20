---
name: contracts-from-sequence
description: Згенерувати API contracts і DB schema разом з sad.md (use cases + container-level sequence), у lockstep, із вшитими invariants migration safety / rollback plan / contract reconciliation. Тригерь, коли архітектурний дизайн (SAD) готовий і час перетворити його на code-ready contracts. Не для greenfield-only — працює і коли в репо вже є деякі migration files. Тригер-фрази: "генерую API від sequence", "DB schema під use cases", "узгоджую API і DB", "reconcile contracts", "після sequence — контракти", "contracts from sad".
---

# contracts-from-sequence

Цей skill ловить **drift між API і DB на момент дизайну, а не після інциденту**. Архітектурний дизайн (sad.md) дав sequence на рівні контейнерів — тепер треба перетворити це на конкретні endpoints і таблиці у такий спосіб, щоб кожне API-поле мало DB backing, а кожна DB-колонка мала або API-expose, або internal-only justification.

Замість того, щоб писати OpenAPI і DDL по черзі (тиждень одне → потім інше → drift у дрібницях), skill генерує їх паралельно з одного джерела і одночасно ганяє self-checks на migration safety, rollback path і reconciliation gaps.

## Inputs

Skill читає з `delivery/<slug>/` (де slug — назва фічі). Якщо твій репо використовує інший layout (`docs/features/<slug>/`, `architecture/<slug>/`) — заміни шлях, протокол не змінюється.

Mandatory:

- **`sad.md`** — §5 Building Blocks (контейнери і компоненти), §6 Runtime View (container-level sequence для ≥2 use cases), §10 Quality requirements (для index decisions).
- **`adr/*.md`** — якщо є shared decisions, що впливають і на DB, і на API (наприклад polymorphic JSONB payload, UNIQUE constraint as domain invariant). Skill читає весь каталог.

Optional:

- **`spike-notes.md`** — ескізні entities і endpoint skeleton, якщо вже зробили spike.
- **`diagrams/seq-*.md`** — окремі sequence files, якщо архітектор винесе їх з sad.md §6.
- **Existing `migrations/`** — якщо у репо вже є migrations, skill читає їх, щоб не дублювати і коректно нарощувати номери.

## Protocol

5 кроків. Між ними — питання користувачу, якщо знайдено ambiguity, яку не можна resolve з context.

### Step 1. Read inputs

```
Read delivery/<slug>/sad.md (focus §5, §6, §10)
Read delivery/<slug>/adr/*.md (all ADR)
Read delivery/<slug>/spike-notes.md (if exists)
Read delivery/<slug>/diagrams/seq-*.md (if exists)
Glob existing migrations/*.up.sql (for numbering + naming)
```

Якщо `sad.md` відсутній — STOP, попроси користувача завершити SAD спершу (skill не запускається на чистому листку).

Якщо `sad.md` є, але §6 Runtime View порожній або має < 2 sequence — STOP, попроси хоча б 2 container-level sequence для типових юзкейсів.

### Step 2. Extract bindings

Для кожного use case з §6 пройти sequence message-by-message і записати у внутрішній blackboard:

- **Actor** — хто триггерить (Web, methodist, member, external system)
- **Call shape** — semantic action ("create draft course", "publish course") і entry container ("API", "Worker")
- **Fields touched** — які поля у DTO згадуються (явно або з PRD via SAD §1)
- **Persistence points** — кожне "API → DB: insert/update/select" — це DB hit, який треба mapping на конкретний column/index
- **Authz layer** — згадки middleware, OrgCtx, FK checks
- **Invariants triggered** — згадки UNIQUE, NOT NULL, idempotency, atomicity у tx

Це нейтральний intermediate representation, з якого далі будуються і OpenAPI, і DDL.

### Step 3. Generate lockstep

OpenAPI draft і DDL draft пишуться **паралельно одного блоку до другого**, не один після одного. Як працює послідовність всередині step:

1. Для кожного use case — endpoint в OpenAPI + table(s) у DDL.
2. Поле з'являється в одному → негайно перевіряємо, чи є у другому (з justification інакше).
3. Validation rule в OpenAPI (`maxLength`, `pattern`, `enum`) → відповідна constraint у DDL (`VARCHAR(N)` для bounded, або **app-layer validation** і коментар у DDL, якщо це business rule).
4. Index decisions — з §10 Quality (latency p95, throughput) і §6 sequence (які поля у WHERE / ORDER BY).
5. Migration files номеруються по існуючим: `Glob migrations/*.up.sql` → `max(NNNN) + 1`.

**Reconciliation invariant перевіряється continuous:** для кожного API field має існувати DB backing (column / view / computed). Для кожної DB колонки — або API expose, або явний коментар `internal-only: <reason>`.

### Step 4. Self-check invariants

Перед записом файлів — прогнати чотири блоки чек-листа. Кожен пункт або **✓** (pass), або **⚠️ flag for human** (записати у reconciliation-report.md).

**Block A. Migration safety**

- [ ] NOT NULL колонка на existing table має 3-step pattern (ADD nullable → backfill → SET NOT NULL у трьох окремих migrations). Якщо table нова — пропускаємо.
- [ ] CREATE INDEX на existing large table — `CONCURRENTLY` і окремий migration file з одним statement (golang-migrate за замовчуванням обгортає у tx, CONCURRENTLY у tx падає).
- [ ] DROP COLUMN — тільки у deprecate-flow (deprecate app-side спершу, окремою migration через тиждень).
- [ ] ALTER TYPE — заборонено на existing з даними. Pattern: new column → backfill → swap → drop old.
- [ ] UNIQUE на existing column — clean duplicates окремою migration спершу, потім UNIQUE.

**Block B. Schema-as-context**

- [ ] CLAUDE.md проекту згадує schema файли через `@migrations/NNNN_*.up.sql` (last 3-5) або `@delivery/<slug>/contracts/data-model.md`. Якщо немає — додати або emit pending action у report.
- [ ] Naming consistency — table names plural snake_case, FK як `fk_<table>_<ref>`, indexes як `idx_<table>_<cols>` (узгоджуємо з existing migrations якщо є).

**Block C. Rollback plan**

- [ ] Pre-deploy backup checklist (Postgres snapshot, queue drain якщо є outbox, feature flag).
- [ ] Ordered rollback steps (feature flag off → app revert → migration down → verify).
- [ ] Кожна `*.up.sql` має парну `*.down.sql`. Для нових tables — `DROP TABLE` у зворотному порядку (FK chain).
- [ ] Risk matrix — мінімум 3 failure scenarios з severity + action.

**Block D. Contract reconciliation**

- [ ] Кожне поле в OpenAPI request/response body → є column / computed / view у DDL.
- [ ] Кожна колонка у DDL → expose у OpenAPI або коментар `internal-only`.
- [ ] Кожний sequence call "API → DB" з §6 → є endpoint + query у lockstep.
- [ ] Error codes у OpenAPI responses → є відповідний domain sentinel (наприклад `lesson.sequence_conflict` ↔ unique_violation на `UNIQUE(course_id, sequence)`).
- [ ] Authz checks з §8 Crosscutting (org_id filter, IsMethodist gate) → відбито в OpenAPI security schemes і репо-шарі.

Якщо знайдено ≥3 ⚠️ flags у одному блоці — STOP, покажи юзеру summary і запитай decision (зафіксувати у ADR / changed sad.md / прийняти і записати у report).

### Step 5. Output

Записати у `delivery/<slug>/contracts/`:

| File | Purpose |
|---|---|
| `openapi.yaml` | Drafted OpenAPI 3.x з усіма endpoints, schemas, error responses |
| `data-model.md` | Mermaid `erDiagram` (overhead-view) → DDL для всіх таблиць → indexes → constraint rationale |
| `migrations/NNNN_*.up.sql` + `*.down.sql` | По одному file per table (або per logical change), парні |
| `rollback-plan.md` | Pre-deploy backup → ordered steps → risk matrix |
| `reconciliation-report.md` | Список ⚠️ flags зі Step 4 + decisions made + open questions для human |

Після запису — короткий summary у chat: скільки tables, скільки endpoints, скільки flags, що відкрите.

## Invariants — детально

Розгорнуті rules, які skill enforce-ить незалежно від конкретного use case.

### DB як просте сховище

Дозволено у DB: `PRIMARY KEY`, `UNIQUE`, `NOT NULL`, `FOREIGN KEY`, B-tree / partial / GIN indexes, `DEFAULT now()` для timestamp колонок.

Заборонено у DB:

- `DEFAULT` для business values (`status = 'draft'`, `role = 'member'`) — два sources of truth, app має задавати.
- `CHECK` constraints для business rules (status у whitelist, score > 0) — блокує evolution. Validation у app.
- Triggers для business logic.
- Stored procedures.
- UUID v4 (`gen_random_uuid()`) для primary key — генеруємо v7 у app для cursor pagination.

Allowed винятки (skill пропустить без flag):

- CHECK constraint для **integrity invariants**, які не є business rules (наприклад `CHECK (size_bytes >= 0)`).
- DEFAULT для **immutable system values** (`created_at` за `now()`).

### UUID v7 у app

PK колонки — `UUID` тип у DDL, але без `DEFAULT gen_random_uuid()`. App layer створює UUID v7 (`uuid.NewV7()` у Go, `uuidv7()` у TS). Цей вибір дає sortable timestamps для cursor pagination.

### Column types

- `VARCHAR(N)` — для bounded strings з maxLength у API (title VARCHAR(200), code VARCHAR(20))
- `TEXT` — тільки для URLs, рендер-готового markdown, descriptions без maxLength
- `TIMESTAMPTZ` — завжди (без `TIMESTAMP` без TZ)
- `JSONB` — не `JSON` (JSON не індексується GIN)
- `BIGINT` для size / count полів, що можуть рости (`SMALLINT` тільки коли точно ≤ 32767)

### ER diagram перед DDL

`data-model.md` починається з Mermaid `erDiagram` блоку (4-8 entities + relationships + key columns). Reader за 30 секунд бачить shape. DDL — деталізація того ж ER.

### Migration file naming

`NNNN_create_<table>.up.sql` / `NNNN_create_<table>.down.sql`. NNNN — 4-digit, наступний після max у каталозі. Якщо migrations не існує — починаємо з `0001`.

## Conflicts — human in the loop

Skill ловить, але не приймає рішення. Типи конфліктів і протокол:

| Conflict | Skill action |
|---|---|
| API повертає поле, якого немає у DDL | ⚠️ flag → запитати: додати column / view / computed? або зняти з API? |
| DDL має колонку, яку API не повертає | ⚠️ flag → запитати: expose у API? або підтвердити internal-only + додати коментар? |
| Sequence call → DB не мапиться на endpoint | ⚠️ flag → запитати: пропущений endpoint? або internal background job (не у скоупі API)? |
| Two writes у sequence без atomicity guarantee | ⚠️ flag → запропонувати tx-wrapping або outbox pattern, чекати rозhwarding |
| Index decision без supporting query у §6 / §10 | ⚠️ flag → запитати: яка query це обслуговує? premature optimization? |

Усі flags пишуться у `reconciliation-report.md`. Якщо їх ≥3 у одному блоці — chat-pause, користувач resolve, потім continue.

## Що skill не робить

- Не пише business code (handlers, services, repo methods) — це окрема дисципліна (impl pack).
- Не запускає migrations — тільки генерує файли. `make migrate-up` робить людина після review.
- Не виправляє sad.md — якщо §6 Runtime View має gaps, skill flag-ує, не редагує SAD.
- Не приймає рішень з ⚠️ flags — людина вирішує.

## Запуск

З root репо у Claude Code:

```
@delivery/course-lesson-mvp/sad.md @delivery/course-lesson-mvp/spike-notes.md
Згенеруй contracts (API + DB) з sad.md + spike. Slug = course-lesson-mvp.
```

Skill підтягнеться через trigger phrase, прочитає inputs, прожене 5 steps, виведе `contracts/` папку і summary з відкритими питаннями.

Якщо існуючі `migrations/` файли вже є — skill автоматично нумерує далі (не overwrite).
