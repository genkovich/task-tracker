---
status: Accepted
owner: "genkovich"
reviewers: ["Tech Lead"]
updated_at: "2026-08-20"
feature_size: "M"
ticket: "N/A — greenfield, no tracker ticket (docs/idea-brief.md, direct init)"
---

# 0002 — Push board state changes via Server-Sent Events

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

AC-09 вимагає, щоб viewer бачив поточний стан board «при кожному відкритті» public link — саме собою це не вимагає оновлення в реальному часі всередині вже відкритої сторінки. `docs/idea-brief.md` §7/§14 явно розглянув і «запаркував» Approach B «жива трансляція борди» через розмір L і крихкість постійних з'єднань саме в залі воркшопу з нестабільним Wi-Fi. Design явно перевідкрив це рішення на blast-radius gate (§4), і користувач обрав live push замість рекомендованого fetch-on-load — це рішення документується тут.

## Decision drivers

- Top-3 quality goal §1: «availability з телефонів у залі воркшопу» — жива демонстрація посилює саме цю мету під час воркшопу.
- Спекова вимога AC-09 (актуальний стан у viewer) не вимагає push, але користувач свідомо обрав ширшу гарантію (стан оновлюється без ручного оновлення сторінки).
- NFR throughput ≥20 req/s **на інстанс** (spec §6) — не вимагає горизонтального масштабування, тож push-механізм може лишатись in-process, без окремого message-broker.

## Considered options

1. **Fetch-on-load, без push** (Recommended у AskUserQuestion) — кожен клієнт запитує стан звичайним REST GET при відкритті сторінки й після кожної власної дії редагування.
2. **Live push через WebSocket/SSE** — сервер тримає відкрите з'єднання з кожним клієнтом і штовхає зміни в реальному часі, щойно хтось перетягнув task.

## Decision outcome

**Chosen:** Option 2 — Live push, конкретизовано до **Server-Sent Events (SSE)**, не повного WebSocket: канал односпрямований (сервер → клієнт), записи й далі йдуть звичайним REST POST/PUT/DELETE через API; браузерний `EventSource` має вбудований auto-reconnect, тож не потрібна ручна reconnect-логіка, яку вимагав би WebSocket. Це тримає «найменше рухомих частин» (idea-brief §8 Engineer-lens synthesis matrix) навіть при виборі push-варіанту.

## Consequences

**Positive**
- Board і у team member, і у viewer оновлюється миттєво без ручного перезавантаження — сильніша демонстрація «це реально працює», ніж fetch-on-load.
- SSE не вимагає нової інфраструктури (жодного message broker) — работає поверх звичайного HTTP-з'єднання, `chi` вже це підтримує.

**Negative**
- Довгоживучі HTTP-з'єднання на кожного відкритого клієнта — сервер і reverse-proxy (Caddy, §7) мають не буферизувати й не обривати SSE-потік по таймауту.
- Новий крос-каттинг-концерн — керування життєвим циклом з'єднань (heartbeat, множинні відкриті вкладки) — див. §8.
- Відкликання public link (AC-08/AC-11) тепер мусить не лише відхиляти нові запити за токеном, а й **синхронно закривати вже відкриті SSE-з'єднання**, автентифіковані цим токеном — без цього кроку viewer, підключений у момент відкликання, продовжував би отримувати оновлення (sad.md §6 Flow 3).

**Neutral**
- В одноінстансному деплойменті (§7) розсилка SSE-подій — просто in-process pub/sub всередині API-процесу; якщо пізніше з'явиться друга репліка API, знадобиться спільний broadcast-механізм (Redis pub/sub чи подібне) — прийнятний технічний борг v1, зафіксований у §11.

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4, §6, §8
- Related ADR: none
