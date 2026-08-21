---
status: Accepted
owner: "genkovich"
reviewers: ["Tech Lead"]
updated_at: "2026-08-20"
feature_size: "M"
ticket: "N/A — greenfield, no tracker ticket (docs/idea-brief.md, direct init)"
---

# 0004 — Fix the board's columns as seeded, non-editable stages

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

`docs/idea-brief.md` §15 лишив явно відкритим питанням: «Які саме колонки за замовчуванням і чи можна їх перейменовувати?» (owner: genkovich, due: до write-prd). `spec.md` §4 user stories (US-01…US-07) не містять жодної історії про створення, перейменування чи видалення column — лише про task (створення/редагування/переміщення/видалення) і public link (отримати/відкликати). Це рішення закриває відкрите питання ідея-брифу на архітектурному рівні.

## Decision drivers

- Spec §4 — відсутність будь-якої user story про керування колонками є прямим сигналом, що це поза скоупом.
- Idea-brief §13 «Locked-in pointer»: «усе, що вводить... на наступних етапах трактується як роздування скоупу».
- Spec §2 Goals: «тримати весь продукт в межах однієї board без додаткових екранів чи режимів».

## Considered options

1. **Фіксований, заданий заздалегідь набір column** — board має незмінний список column (напр. «To Do / In Progress / Done»), заданий seed-міграцією; team member лише переміщує task між ними.
2. **Team member може створювати й перейменовувати column** — додається повний CRUD для column: нові ендпоінти, нова форма в UI, нова таблиця з керуванням.

## Decision outcome

**Chosen:** Option 1 — фіксований набір column. Жодна acceptance criteria в spec.md цього не вимагає; додавання CRUD зараз було б новою, неспека­фікованою функціональністю, що прямо суперечить spec §2 і idea-brief §13. Якщо потреба виникне пізніше, це технічно **не потребує міграції даних** (лише нова таблиця й ендпоінти) — рішення обране як явне, а не самостійно прийняте design, саме тому що воно було відкритим питанням ідея-брифу.

## Consequences

**Positive**
- Найпростіша можлива схема даних для column (без CRUD-шару, без валідації унікальності назв, без порядку сортування колонок).
- Закриває відкрите питання idea-brief §15 однозначно, без подальшої двозначності для `data-model`/`api`.

**Negative**
- Якщо команда захоче іншу назву чи кількість колонок, потрібна ручна зміна seed-даних/міграція, не UI-дія.

**Neutral**
- Перехід до Option 2 пізніше — адитивна зміна (нова таблиця + ендпоінти), не потребує backfill чи ламання існуючої схеми.

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4, §5
- Related ADR: none
