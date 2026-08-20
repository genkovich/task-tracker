---
status: Accepted
owner: "genkovich"
reviewers: ["Tech Lead"]
updated_at: "2026-08-20"
feature_size: "M"
ticket: "N/A — greenfield, no tracker ticket (docs/idea-brief.md, direct init)"
---

# 0001 — Build board as a backend API plus a web SPA

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

Board — перший продуктовий модуль репо (`docs/architecture-map.md` — досі лише auth/user каркас). Team member редагує board напряму через UI, viewer відкриває board через public link — обидва потребують того самого централізовано збереженого стану (spec §2 Goals). `ux-flows.md` вже описує 6 екранів (SCR-01…SCR-06) з flow-діаграмами для кожної user story — це не гіпотетичний UI, а вже спроєктований.

## Decision drivers

- Централізоване збереження стану, щоб board переживала чистку браузера власника й показувала актуальний стан кожному viewer (spec §2 Goals).
- `ux-flows.md` — повний екранний інвентар і флоу для team member та viewer — сильний сигнал реальної UI-поверхні, не лише API.
- Репо вже модульний моноліт з одним фронтендом (React SPA) і одним бекендом (Go API) — `docs/architecture-map.md` §Stack.

## Considered options

1. **Backend API + web SPA** — REST API (Go/chi) зберігає й віддає стан; React SPA — єдиний UI і для team member, і для viewer.
2. **Backend API лише (API-only)** — team member і viewer працюють напряму з REST API (curl/Postman), без браузерного UI.

## Decision outcome

**Chosen:** Option 1 — Backend API + web SPA. API-only суперечить і spec §1 («аудиторія відкриває public link зі своїх телефонів»), і всьому `ux-flows.md`, який вже спроєктував UI для кожної user story.

## Consequences

**Positive**
- Одне джерело істини (Postgres через API) — і team member, і viewer завжди бачать той самий стан.
- API можна повторно використати, якщо пізніше з'явиться мобільний застосунок.

**Negative**
- Два контейнери замість одного — SPA викликає API мережевим запитом, а не прямим викликом функції.

**Neutral**
- SPA вже архітектурно зафіксована в репо як `ssr: false` (React Router 7 SPA mode) — це не нове рішення цієї фічі, а успадкований конвент (див. sad.md §4 рішення 2 — UI-архітектура).

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4
- Related ADR: none
