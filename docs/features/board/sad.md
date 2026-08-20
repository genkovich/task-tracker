---
status: Draft
owner: "genkovich"
reviewers: ["Tech Lead", "Security Lead"]
updated_at: "2026-08-20"
feature_size: "M"
target_surfaces: [backend-service, web-frontend]  # filled in §4 — subset of: backend-service | web-frontend | mobile-app | desktop-app | cli | worker | library-sdk. Read (never re-derived) by api/sequences/tasks/plan-tests/review → _shared/surfaces.md
---

# Software Architecture Document — board

<!-- 12 Arc42 sections. Empty section → <!-- N/A: <one-line reason> -->. -->
<!-- C4 Context (L1) lives inline in §3. C4 Container (L2) lives inline in §5. -->
<!-- Numbers in §10 come VERBATIM from spec.md §6 NFR — no inventing, no rounding. -->

## 1. Introduction and goals

**Intent.** Дати малій команді (3–7 людей) спільний, завжди актуальний канбан-стан задач без реєстрації чи облікових записів, і дати їм один лінк, яким можна показати реальний прогрес глядачам поза командою — виключно на перегляд, без права редагування. Продукт — рівно одна board, яку team member редагує напряму (створює, редагує, переміщує перетягуванням, видаляє task), і яку будь-хто може відкрити через public link (spec §2 Goals).

**Top-3 quality goals (1-liners; full scenarios in §10):**

1. **Availability з телефонів у залі воркшопу** — найближчий воркшоп є фіксованим дедлайном і живою перевіркою продукту: public link має відкриватись зі смартфонів глядачів надійно, а не лише з ноутбука команди (spec §1 Context, §7 KPIs).
2. **Узгодженість статусу task під конкурентним перетягуванням** — task завжди належить рівно одній column, навіть коли двоє team member одночасно перетягують ту саму task (AC-05b, domain invariant з spec §6).
3. **Відповідність (responsiveness) дій редагування** — щоб інструмент читався як «реально працює», а не як макет слайда: p95 запису ≤300ms, p95 завантаження board ≤500ms (spec §6 NFR).

**Stakeholders.**

| Role | Interest | Sign-off owner? |
|---|---|---|
| team member (CONTEXT glossary) | Редагує board напряму: створює/редагує/переміщує/видаляє task, керує public link | No |
| viewer (CONTEXT glossary) | Відкриває public link, бачить поточний стан board лише на перегляд | No |
| Tech Lead | SAD approval | Yes |
| Security Lead | Security review публічного неавтентифікованого доступу (spec §6.1) | Yes |

**Decision override:** real-time board delivery через Server-Sent Events, а не fetch-on-load — rationale: idea-brief §7/§14 явно розглянув і «запаркував» Approach B «жива трансляція борди» через розмір L і крихкість постійних з'єднань у залі воркшопу з нестабільним Wi-Fi. Design (§4, blast-radius gate) явно перевідкрив це рішення, і користувач обрав live push замість рекомендованого fetch-on-load, звузивши реалізацію до односпрямованого SSE (не повний WebSocket) — деталі й наслідки → ADR-0002.

## 2. Constraints

<!-- 🎯 Why: §4 strategy only works when §2 has fixed WHAT IS ALREADY FIXED — stack, versions,
     deadline, regulatory. This is an input, not an output.
     📋 Write: four blocks — Technical / Organisational / Conventions / Regulatory.
     📌 Pin versions («<datastore> 18», not «<datastore>»); «Q3 deadline — hard», not «ideally».
     Never N/A — every feature inherits at least Conventions + Technical. -->

**Technical.**
- Backend: Go 1.25.0, chi/v5 (routing), pgx/v5 + pgxpool (Postgres driver), golang-migrate/v4, google/uuid v7, prometheus/client_golang (`docs/architecture-map.md` §Stack).
- Frontend: React 19.2.4 + React Router 7.12.0 **SPA mode** (`ssr: false` — repo-wide, not per-feature), Vite 7.1.7, Tailwind 4.1.13 + shadcn/ui + CVA.
- Datastore: PostgreSQL 18 (`postgres:18-alpine`), accessed via pgx/v5 repo pattern; no other datastore in the repo today.
- Architecture convention: modular monolith, manual constructor-injection wiring (`api/cmd/api/main.go`); each module owns `domain/app/ports/infra` + a top-level `New(...)` (`CLAUDE.md` §Architecture, `.claude/rules/go-*.md`).

**Organisational.**
- Effort budget: ~3 person-weeks (idea-brief §11 RICE Effort signal, Approach C).
- Deadline: hard — the nearest workshop is the trigger event (spec §1 Context), but no calendar date is fixed in any upstream artifact (idea-brief, spec) — **`<TBD by PM>`**, tracked as a §11 risk row.
- Team composition: solo author driving both product and implementation (idea-brief §3 Users, §12 Feasibility «Skills» — full-cycle expertise already in hand).

**Conventions.**
- `.claude/rules/go-*.md` (naming, error handling, concurrency, context, structs/interfaces, security, observability) + `.claude/rules/migrations.md` (paired `.up.sql`/`.down.sql`, sequential 6-digit numbering, current head `000005`).
- IDs: app-generated UUIDv7 via `google/uuid`, never DB `SERIAL`/`gen_random_uuid()`.
- Error handling: domain sentinel errors → per-module `ports/errors.go` `mapError` → `apperr.Error` → `httputil.WriteError`.
- Frontend: Feature-Sliced Design (`app/ → pages/ → widgets/ → features/ → entities/ → shared/`), typed API client (`web/src/shared/api/client.ts`), UI composed from `shared/ui` shadcn primitives — never hand-edited.

**Regulatory / external.**
- No formal compliance regime applies (internal team tool, no accounts, no regulated data categories) — spec §6.1 classifies task titles/assignee names as **internal** data, not personal-data-regulated beyond the free-text assignee name.
- Security review is **required** before ship — spec §6.1 names the new unauthenticated public read access as the project's primary risk (idea-brief §10 devil's-advocate risk: a leaked/indexed link exposes team data indefinitely without revocation).

## 3. Context and scope

<!-- 🎯 Why: draws the SYSTEM BOUNDARY — who talks to it from outside, where the trust zone ends.
     Without §3, §5 and §8 (authorization) blur — unclear what's «inside» vs «outside».
     📋 Write: 2–3 sentences of business context + an external-systems table + a C4Context block.
     📌 «External: none (deliberate, no third-party in v1)» is itself a decision worth stating.
     Trust boundary — the line past which you don't trust data without checking it.
     Never N/A — greenfield still draws the planned actors + external systems. -->

Board — рівно одна спільна канбан-дошка команди. team member редагує її напряму, без входу в систему; будь-хто, хто отримав public link від team member, відкриває board виключно на перегляд, теж без входу в систему.

<!-- brownfield: `docs/architecture-map.md` (reflects cfa5e7f) — fresh full-stack scaffold, auth/user — це лише каркас автентифікації й профілю (Google OAuth + JWT), не продуктовий домен; board — перший продуктовий модуль і свідомо не використовує наявний auth (`architecture-map.md` §Constraints & known tech-debt). -->

**External systems (in / out):**

| Actor or system | Type | Interaction |
|---|---|---|
| team member | Person | Створює/редагує/переміщує/видаляє task; отримує/відкликає public link |
| viewer | Person | Відкриває public link, бачить поточний стан board лише на перегляд |
| External systems | — | **None** (deliberate) — board не викликає жодного зовнішнього сервісу (ні наявний Google OAuth, ні сповіщення, ні сторонні API); увесь стан централізований в одному Postgres |

**C4 Context (L1):**

```mermaid
C4Context
    title board — System Context

    Person(member, "Team member", "редагує board напряму, без входу в систему")
    Person(viewer, "Viewer", "відкриває public link, бачить board лише на перегляд")

    System(board, "Board", "Одна спільна канбан-дошка: колонки, task, public link")

    Rel(member, board, "Створює / редагує / переміщує / видаляє task; керує public link", "HTTPS")
    Rel(viewer, board, "Відкриває board за public link", "HTTPS")
```

Board показаний як один чорний ящик: team member взаємодіє з ним напряму (повний доступ до редагування), viewer — виключно через public link (лише перегляд). Зовнішніх систем немає — свідоме рішення: наявний Google OAuth-каркас репо (`architecture-map.md`) залишається невикористаним цією фічею, бо продукт explicitly не має акаунтів (spec §3 Non-goals).

## 4. Solution strategy

<!-- 🎯 Why: the 3–4 STRATEGIC PILLARS every ADR grows from. Without §4 each ADR looks random —
     there's no umbrella. ⭐ The densest section — the blast-radius gate fires almost always here
     (decisions are irreversible + multi-module).
     📋 Write: 3–4 choices; each a heading + 2–3 sentences of rationale.
     📌 «Store content as a table of typed blocks» is a pillar — ADR-0001 grows from it. -->

**Top strategic choices (the seeds for ADRs):**

1. **Target surface: backend API + web SPA** — board — це Go REST API (`backend-service`) плюс React SPA (`web-frontend`), єдине джерело істини — Postgres через API; і team member, і viewer завжди бачать той самий стан (ux-flows.md — 6 екранів). → **ADR-0001**.
2. **Push board-стану через Server-Sent Events (SSE), не fetch-on-load** — щойно team member змінює task, кожен відкритий клієнт (інший team member або viewer на public link) отримує оновлення миттєво, без ручного перезавантаження — «жива» демонстрація на воркшопі. Конкретний механізм — SSE (`EventSource`), не повний WebSocket: канал односпрямований (сервер → клієнт; записи й далі йдуть звичайним REST POST/PUT/DELETE), а `EventSource` має вбудований auto-reconnect — менше рухомих частин, ніж ручна reconnect-логіка WebSocket. → **ADR-0002**.
3. **Public link — непередбачуваний токен, збережений у Postgres** — жодної криптографії з підписом; відкликання = один `DELETE`/`UPDATE` рядка, миттєво і без побічних ефектів (AC-11). → **ADR-0003**.
4. **Column — фіксований, заданий заздалегідь набір (seed-міграція), без CRUD** — жодна user story spec не вимагає керування колонками; мінімальна схема даних відповідає принципу «без роздування скоупу» (idea-brief §13). → **ADR-0004**.

Each tactical decision in later sections should trace to one of these seeds. Tactical decisions that *contradict* a strategic choice are red flags — surface them in §11.

## 5. Building block view

<!-- 🎯 Why: INTERNAL DECOMPOSITION — modules, containers, datastores. The static topology: who
     may talk to whom. Without §5, §6 (the flows) has no vocabulary of participants.
     📋 Write: 1 ¶ on the style (layered / hexagonal / clean / event-driven) + a folder tree + a
     C4Container block.
     📌 Draw ONE Container per declared `target_surface` (frontmatter): a fullstack
     [backend-service, web-frontend] = a backend-API container + a web/SPA container; a
     [backend-service, mobile-app] = the API + the mobile app. The Container(web, …) line below is
     just one surface's container — swap/add per what was declared in §4. → _shared/surfaces.md
     📌 e.g. «web app, content API, media worker, datastore, object store, CDN». -->

Backend — layered модуль за наявною конвенцією репо (`domain/app/ports/infra`, ручний constructor-injection, ADR-0001): новий модуль `board` без наявних module-to-module залежностей (auth/user не зачіпаються, ADR-0001 контекст). Frontend — FSD-фіча `board` (api/model/ui) плюс окрема сторінка для публічного read-only перегляду, обидві композуються з наявних shadcn/ui-примітивів (`architecture-map.md` §Frontend), без нового styling-підходу.

**Internal decomposition:**

```
api/internal/modules/board/
├── domain/       <Board, Column (fixed set, ADR-0004), Task entities + sentinel errors>
├── app/          <BoardService: CreateTask/EditTask/MoveTask/DeleteTask/IssueLink/RevokeLink/GetState>
├── infra/        <Postgres repo (pgx) + in-process SSE hub (ADR-0002, broadcast to subscribed connections)>
├── ports/        <HTTP handlers (team-editor routes, public-viewer routes by token, SSE endpoint), DTOs, mapError>
└── board.go      <New(...) *ports.Handler, wired in api/cmd/api/main.go — no authMW (ADR-0001 context, no accounts)>

web/src/
├── pages/board/            <editor view — composes features/board, features/public-link (SCR-01, SCR-04)>
├── pages/board-public/     <public read-only view by link token (SCR-05, SCR-06)>
└── features/board/
    ├── api/                <typed client calls + SSE subscription (ADR-0002)>
    ├── model/               <drag-and-drop state, optimistic UI for create/edit/move/delete>
    └── ui/                  <task card, column, quick-add (SCR-02), edit modal (SCR-03)>
```

**C4 Container (L2):**

```mermaid
C4Container
    title board — Containers

    Person(member, "Team member")
    Person(viewer, "Viewer")

    Container_Boundary(app, "Board") {
        Container(web, "Web SPA", "React 19 + React Router 7 (SPA)", "editor UI (SCR-01..04) + public read-only view (SCR-05/06); SSE client")
        Container(api, "Board API", "Go + chi", "REST endpoints (create/edit/move/delete task, issue/revoke link) + SSE broadcast endpoint")
    }

    ContainerDb(pg, "PostgreSQL 18", "pgx/v5", "boards, columns (fixed seed, ADR-0004), tasks, public link tokens (ADR-0003)")

    Rel(member, web, "Редагує board напряму", "HTTPS")
    Rel(viewer, web, "Відкриває public link", "HTTPS")
    Rel(web, api, "REST виклики + підписка на оновлення", "JSON/HTTPS + SSE (ADR-0002)")
    Rel(api, pg, "Читає/записує стан board", "pgx (SQL, $1 params)")
```

## 6. Runtime view

<!-- 🎯 Why: the RUNTIME FLOW of 1–2 critical scenarios — who talks to whom, when, in what order.
     Without §6, §5 is just boxes with no life.
     📋 Write: a Mermaid sequenceDiagram. Participants are names from §5 (don't invent new ones).
     Messages are semantic («saves a draft»), NO HTTP verbs / paths / status codes — endpoint-level
     sequences arrive at the `api` stage.
     📌 e.g. «author → web: composes draft → web → content API: save». Seed the primary flow(s) here;
     the `sequences` stage then covers every §5 AC (no cap). Never N/A for M+; XS/S keeps ≥1 happy-path flow. -->

**Critical flow 1: Team member перетягує task, зміна доходить до всіх живих клієнтів (AC-04, ADR-0002)**

```mermaid
sequenceDiagram
    actor Member as Team member
    actor Other as Інший відкритий клієнт
    participant Web
    participant API as Board API
    participant DB as PostgreSQL

    Member->>Web: перетягує task в іншу column і відпускає
    Web->>API: просить перемістити task у нову column
    API->>DB: записує нову column для task
    DB-->>API: ok
    API-->>Web: підтверджує переміщення
    Web-->>Member: task показана в новій column
    API->>Other: розсилає подію "стан змінився" (SSE, ADR-0002)
    Other->>API: запитує оновлений стан board
    API->>DB: читає поточний стан
    DB-->>API: поточний стан
    API-->>Other: оновлений стан
    Other-->>Other: показує task у новій column без ручного оновлення
```

**Critical flow 2: Конкурентне перетягування тієї самої task двома team member (AC-05b, domain invariant)**

```mermaid
sequenceDiagram
    actor MemberA as Team member A
    actor MemberB as Team member B
    participant API as Board API
    participant DB as PostgreSQL

    MemberA->>API: просить перемістити task у column X
    MemberB->>API: просить перемістити ту саму task у column Y (майже одночасно)
    API->>DB: записує column X для task
    DB-->>API: ok
    API->>DB: записує column Y для task (обробляється останньою)
    DB-->>API: ok — column Y перезаписує column X
    API-->>MemberA: підтверджує запит (без знання про перегонову умову)
    API-->>MemberB: підтверджує запит
    API->>MemberA: розсилає подію "стан змінився" (SSE)
    API->>MemberB: розсилає подію "стан змінився" (SSE)
    MemberA->>API: запитує оновлений стан
    API-->>MemberA: task показана в column Y — єдиний узгоджений стан для обох
```

**Critical flow 3: Viewer відкриває public link; лінк відкликано (AC-09, AC-10, AC-11)**

```mermaid
sequenceDiagram
    actor Viewer
    participant Web
    participant API as Board API
    participant DB as PostgreSQL

    Viewer->>Web: відкриває public link (токен у адресі)
    Web->>API: запитує стан board за токеном
    API->>DB: перевіряє токен і читає стан
    alt токен чинний — AC-09
        DB-->>API: поточний стан board
        API-->>Web: стан board, лише перегляд
        Web-->>Viewer: показує board (SCR-05)
        Viewer->>Web: намагається перетягнути task — AC-10
        Web-->>Viewer: дію відхилено, "лише перегляд"
    else токен відкликаний або недійсний — AC-11
        DB-->>API: токен не знайдено / неактивний
        API-->>Web: доступ не надається
        Web-->>Viewer: показує "лінк недоступний" (SCR-06)
    end
```

## 7. Deployment view

<!-- 🎯 Why: the TOPOLOGY DevOps must know without reading the deploy charts — how many replicas,
     where the background worker lives, AT WHAT NUMBERS we scale.
     📋 Write: 2–3 sentences on topology + monitoring + concrete threshold numbers.
     📌 e.g. «500 authors → partition by quarter» (not «we'll think about scale later»).
     🎯 N/A allowed for XS/S that reuses an existing deployment unit with no change.
     Deployment-diagram scaffold → templates/deployment.md. -->

**Це закриває spec §8 Open Question 1** («де хоститься board, щоб public link був стабільно доступний з телефонів глядачів у залі воркшопу»): board не отримує нової інфраструктури — вона розгортається в тому самому одноінстансному VPS-стеку, що вже є в репо (`deploy/docker-compose.prod.yml`, `deploy/Caddyfile`, `.github/workflows/deploy.yml`): один контейнер `api` (тепер несе й board-модуль), один `web` (SPA), один `postgres`, за Caddy з автоматичним TLS на публічному домені `${DOMAIN}`. Публічний домен з HTTPS — і є відповідь на «доступність з телефонів у залі»: жодна мережа воркшопу не потрібна, глядачі йдуть у звичайний інтернет.

**Специфічна вимога ADR-0002 (SSE):** Caddy reverse-proxy для `/api/*` не повинен буферизувати SSE-потік (`encode` + типове буферизоване проксіювання зіпсують push-доставку) — для SSE-ендпоінта board потрібен `flush_interval -1` (негайний flush) у `reverse_proxy`-директиві `deploy/Caddyfile`; це конфігураційна зміна existing Caddyfile, не нова інфраструктура.

**Monitoring:**
- Метрики: наявний `/metrics` (Prometheus) розширюється board-специфічними лічильниками — кількість активних SSE-з'єднань, кількість відправлених broadcast-подій, латентність запису task (для перевірки NFR p95 ≤300ms).
- Алерти: не вводяться нові окремо для board у v1 — використовується наявний Grafana-дашборд (`deploy/grafana/dashboards/`); нова панель для SSE-з'єднань — рекомендація для `implement`, не блокер design.
- Tracing: не вводиться (репо не використовує розподілену трасировку сьогодні — поза скоупом цієї фічі).

**Scaling thresholds:**
- Один інстанс `api` комфортно тримає ціль NFR ≥20 req/s (spec §6) і очікуваний масштаб — команда 3–7 людей + ~30 глядачів воркшопу одночасно (idea-brief §11 Reach).
- Понад це (друга репліка `api`) SSE-розсилка (ADR-0002) перестає працювати коректно без спільного pub/sub — зафіксовано як прийнятний технічний борг v1 у §11, не поточна вимога.

## 8. Crosscutting concepts

<!-- 🎯 Why: CROSS-CUTTING PATTERNS spanning several modules: logging, errors, authorization, ID
     strategy, events, caching. ⭐ The second-densest section. A pattern inside one module is NOT
     here; a project-wide convention belongs in the convention file.
     📋 Write: a table — concept / convention / where defined. One row per concept.
     📌 e.g. «sortable time-based IDs generated in the app layer» as a default from the convention file. -->

| Concept | Convention | Where defined |
|---|---|---|
| Logging | Успадковано: `log/slog` JSON handler, structured key-value | `architecture-map.md` §Conventions |
| Authorization | **Не** через наявний `authMW` (Google OAuth) — board і viewer, і team member обслуговує без входу в систему. Дві можливості розрізняються шляхом/токеном, не роллю: team-editor route (`RouteRegistrar`, публічний за визначенням — знання базового URL = право редагувати) проти public-viewer route з opaque токеном у шляху (ADR-0003, перевірка в БД, без ролей і сесій) | sad.md §4 (ADR-0003), §3 (capability model, CONTEXT glossary) |
| Error handling | Успадковано: domain sentinel errors → `ports/errors.go` `mapError` → `apperr.Error` → `httputil.WriteError` | `architecture-map.md` §Conventions |
| ID strategy | Успадковано: app-generated UUIDv7 через `google/uuid`, не DB `SERIAL` | `architecture-map.md` §Conventions |
| Internationalisation | N/A — один язик (українська), продукт не має налаштувань локалі (spec не вимагає) | — |
| Observability | Успадковано `/metrics` (Prometheus) + нові board-лічильники (§7): активні SSE-з'єднання, broadcast-події, латентність запису | sad.md §7 |
| Events | In-process pub/sub у межах одного `api`-процесу для SSE-розсилки (ADR-0002) — не крос-модульна шина подій, локально для `board`-модуля | sad.md §4 (ADR-0002) |
| Rate limiting | Нове: in-process token bucket на клієнта (по IP), обмежений до ендпоінта створення task, ≤30 створень/хв (spec §6.1 abuse case). Reversible, contained в одному модулі — жодної нової інфраструктури (Redis тощо) не потрібно при одноінстансному деплойменті (§7) | sad.md §7, §8 (тут) |

## 9. Architecture decisions

<!-- 🎯 Why: the REVERSE INDEX onto the adr/ folder. `ls adr/` gives the files; §9 gives the
     semantics — why they exist, which SAD section they attach to, what status.
     📋 Write: a 4-column table, one row per ADR. Mixed status is fine.
     📌 e.g. «0001 | Store content as a table of typed blocks | Accepted | §4». -->

| # | Title | Status | Section |
|---|---|---|---|
| 0001 | Build board as a backend API plus a web SPA | Accepted | §4 |
| 0002 | Push board state changes via Server-Sent Events | Accepted | §4 |
| 0003 | Use an opaque DB-stored token for the public link | Accepted | §4 |
| 0004 | Fix the board's columns as seeded, non-editable stages | Accepted | §4 |

ADR files live under `docs/features/<slug>/adr/NNNN-<title>.md`.

## 10. Quality requirements

<!-- 🎯 Why: the QUALITY TREE — take a goal from §1 and break it into concrete leaves: tests,
     metrics, configs, drills. ⭐ Without §10, §1 is a manifesto. With §10 each declaration maps
     to something PROVABLE.
     📋 Write: per §1 goal — When / Then / How-verify. Numbers from spec §6 NFR VERBATIM (don't
     round ≤250ms to ≤300ms — that's a critic F6 hit).
     📌 e.g. «p95 ≤ 500 ms on a block update, verified by a 100 req/s load test». -->

Each top-3 goal from §1 expanded into a full scenario:

**QG-1. Availability з телефонів у залі воркшопу**
- **When:** viewer відкриває public link під час воркшопу (перша хвиля ~30 людей, idea-brief §11 Reach) або в будь-який інший момент.
- **Then:** сервіс доступний за public HTTPS-домен (§7) з Availability 99% (щомісячне SLO-вікно, spec §6 NFR).
- **How verify:** production monitoring (uptime метрика на `/metrics` + Grafana дашборд, §7); ручна перевірка з кількох мобільних пристроїв перед датою воркшопу.

**QG-2. Узгодженість статусу task під конкурентним перетягуванням**
- **When:** двоє team member одночасно перетягують ту саму task у різні column (AC-05b).
- **Then:** task лишається рівно в одній column — тій, чия зміна оброблена останньою — і кожен наступний перегляд будь-ким показує саме цей єдиний узгоджений стан (spec §6 NFR «Узгодженість статусу task»).
- **How verify:** domain invariant test — конкурентний запис двох команд move у тестовому середовищі, перевірка фінального стану в БД (integration test, `//go:build integration`).

**QG-3. Відповідність (responsiveness) дій редагування**
- **When:** team member створює, редагує, переміщує чи видаляє task; будь-хто (team member або viewer) завантажує board.
- **Then:** p95 латентність запису ≤300ms; p95 латентність завантаження board ≤500ms; throughput ≥20 req/s на інстанс (spec §6 NFR, вербатим).
- **How verify:** production monitoring (Prometheus гістограми латентності per-endpoint) + smoke load-test у CI (spec §6 «Throughput | Measurement: smoke test у CI»).

## 11. Risks and technical debt

<!-- 🎯 Why: ⭐ collects EVERYTHING that can break — not only the technical. Without §11 risks get
     discussed at standups and lost; debt lives only in the head of whoever accepted it.
     📋 Write: a risk/debt table — severity — mitigation — owner. Accepted debt in its own block.
     📌 The first risk is often a product risk, not a technical one. That's normal. -->

<!-- Severity literals: Low / Medium / High for regular risks; "Open question" for rows created by
     a Save-as-OQ resolution during the Socratic walk (see references/socratic.md). -->

| Risk / debt | Severity | Mitigation | Owner |
|---|---|---|---|
| Дедлайн воркшопу — тригер фічі — не має зафіксованої календарної дати в жодному upstream-артефакті (spec §2 Constraints, Organisational) | Medium | Підтвердити точну дату воркшопу до старту `tasks`/`implement`, щоб effort-бюджет (3 person-weeks, idea-brief §11) залишався реалістичним | genkovich (PM) |
| Base editor URL board повністю відкритий — знання базового домену дає повне право редагування, бо акаунтів немає (spec §3 Non-goals, sad.md §8 Authorization row) | Medium | Свідомий, спекою зафіксований компроміс — не таємний за замовчуванням; security review (spec §6.1) явно перевірить, чи прийнятний цей ризик для продукту без акаунтів | Security Lead |
| Наявний Google OAuth-каркас (`auth`/`user` модулі) лишається невикористаним board — напруга з product-домену, що починається без акаунтів (`architecture-map.md` §Constraints & known tech-debt, brownfield gotcha) | Low | Прийнятно для v1 — board свідомо не використовує auth (ADR-0001 context); переоцінити, якщо продукт колись додасть акаунти | Backend |
| SSE-розсилка (ADR-0002) — in-process, не переживає горизонтальне масштабування `api` понад один інстанс | Medium | Прийнятний борг v1 при одноінстансному деплойменті (§7); знадобиться спільний pub/sub (напр. Redis) лише якщо з'явиться друга репліка | Backend |
| Перетягування може не працювати коректно на сенсорних екранах — головна дія продукту недоступна більшості аудиторії воркшопу (idea-brief §10 risk) | Medium | Мобільна постава `ux-flows.md` (mobile-first, drag-and-drop через touch і мишу) — ручна перевірка на реальних мобільних пристроях перед `ship` | genkovich |
| Open architectural decision: чи потрібна резервна копія стану board перед воркшопом (spec §8 OQ-2) | Open question | Resolve до дати воркшопу; spec §8 default-now — без окремого механізму бекапу для цієї фічі, рішення відкладено навмисно | genkovich |

**Accepted debt (acceptable in v1, plan to fix later):**
- Column — фіксований, незмінний набір (ADR-0004); додавання CRUD пізніше — адитивна зміна, не backfill.
- SSE без горизонтального масштабування (рядок вище) — прийнятно, поки деплоймент лишається одноінстансним (§7).
- Жодного event-schema versioning для SSE-подій у v1 — подія лише сигналізує «стан змінився» (тригер на refetch), без власного payload, що потребував би версіювання.

## 12. Glossary

<!-- 🎯 Why: ⭐ the DOMAIN GLOSSARY that ends arguments a year later («checkpoint — weekly or
     biweekly? quarter — calendar or fiscal?»).
     📋 Write: a term / meaning table. Business + technical terms mixed.
     📌 e.g. «Lesson | a unit inside a course made of blocks (text, video)». -->

| Term | Meaning |
|---|---|
| <e.g. domain object A> | <its meaning in this domain> |
| <e.g. domain object B> | <its meaning> |
| <e.g. domain invariant name> | <the rule, in plain language> |
