---
status: Draft
owner: "genkovich"
reviewers: []
updated_at: "2026-08-20"
feature_size: M
---

# Events — board

Async contract for `sad.md` §6 Flow 1 (team member перетягує task → усі live-клієнти
оновлюються) та Flow 3's SSE leg (ADR-0002). Це **не** message-bus контракт — ADR-0002
свідомо обрала in-process SSE замість будь-якого брокера («NFR throughput ≥20 req/s на
інстанс не вимагає горизонтального масштабування»); канал живе всередині одного процесу
API, без черги, без persist-шару. Секція нижче адаптує стандартну форму цього файлу до
того, що реально описано в §6 — не вигадує retry/DLQ-числа, яких там нема.

## Channel: `board.sse` (in-process, per API instance)

- **Producer:** `Board API` — та сама служба, що приймає мутуючі REST-запити (create/edit/
  move/delete task, issue/revoke public link); мовний брокер не задіяний.
- **Consumers:** кожен live-клієнт (team-editor SPA через `GET /api/v1/board/events`, viewer
  через `GET /api/v1/public/{token}/events`) з відкритим SSE-з'єднанням до цього ж
  API-інстанса.
- **Delivery:** best-effort, at-most-once на з'єднання. Подія — лише сигнал «стан змінився»,
  без даних; клієнт після отримання одразу робить звичайний REST `GET` за актуальним
  станом (sad.md §6 Flow 1: `Other->>API: запитує оновлений стан`), тож пропущена подія під
  час розриву з'єднання не губить дані — наступний `GET` завжди повертає узгоджений стан
  (AC-05b).
- **Ordering:** none — подія не несе версії чи порядкового номера; єдине джерело істини
  лишається REST `GET`, подія тільки каже "запитай знову".

## Event: `board.state_changed.v1`

```json
{
  "event_id": "<uuid>",
  "event_type": "board.state_changed",
  "version": 1,
  "occurred_at": "<iso8601>"
}
```

- **Required fields:** `event_id`, `event_type`, `version`, `occurred_at`. Немає поля `data`
  — sad.md §6 Flow 1 не показує жодного payload у самій події, лише подальший `GET`.
- **Origin:** `sad.md` §6 Flow 1 — `API->>Other: розсилає подію "стан змінився" (SSE, ADR-0002)`,
  і Flow 3 (виток закриття з'єднання при відкликанні лінка).
- **Backwards-compat policy:** additive-only — новий payload-field у майбутньому (наприклад
  ідентифікатор зміненої task, щоб клієнт міг оновити точково) — це `v2`, не мовчазна зміна
  `v1`. Клієнти ігнорують невідомі поля.

## Idempotency & retry

- **Idempotency:** подія — сигнал без стану, дублювання нешкідливе (клієнт просто зробить
  зайвий `GET`).
- **Retry / reconnect:** керується браузерним `EventSource` (auto-reconnect, вбудований —
  ADR-0002 «Positive»); сервер не веде окремого лічильника спроб і не гарантує доставку
  пропущених під час розриву подій — REST `GET` після реконекту відновлює consistency.
- **Dead-letter:** N/A — немає черги, немає збереження недоставлених подій.

## Connection lifecycle (revocation — AC-08, AC-11)

- API веде реєстр «токен public link → активні SSE-з'єднання, автентифіковані цим
  токеном» (sad.md §6, ADR-0002 negative consequence).
- `DELETE /api/v1/board/public-link` синхронно закриває кожне зареєстроване з'єднання для
  відкликаного токена, а не лише блокує нові запити на `/api/v1/public/{token}/events` —
  інакше viewer, підключений у момент відкликання, продовжував би отримувати `board.state_changed`
  всупереч AC-11.

## Schema registry

- Registry: немає окремого реєстру — єдине джерело схеми цього файлу, `contracts/openapi.yaml`
  (SSE-ендпоінти задокументовані там `content: text/event-stream`).
- Validator: немає — репо не має JSON Schema/Avro-тулінгу для подій (`docs/architecture-map.md`
  не згадує жодного); подія достатньо мала (4 фіксованих поля), щоб перевірятись інлайн у
  тестах SSE-хендлера.
