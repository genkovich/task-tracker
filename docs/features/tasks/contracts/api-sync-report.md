# API sync report — tasks — 2026-08-21

Interface kind: **HTTP/REST** (`target_surfaces: [backend-service, web-frontend]`, `web-frontend` consumes this contract, does not author its own). `data-model.md` is present — no fast-lane skip.

## Section A — field-origins

| schema_path | origin | confidence |
|---|---|---|
| Card.id | data-model.md → cards.id (UUID PK) | high |
| Card.name | data-model.md → cards.name (VARCHAR(200)) | high |
| Card.assignee | data-model.md → cards.assignee (VARCHAR(100), nullable) | high |
| Card.column_status | data-model.md → cards.column_status (VARCHAR(20)) — enum values from spec Non-goals ("To Do → In Progress → Done") | high |
| Card.created_at | data-model.md → cards.created_at | high |
| Card.updated_at | data-model.md → cards.updated_at | high |
| CardCreate.name / assignee | mirrors Card, minus server-set fields | high |
| CardUpdate.name / assignee | mirrors Card, minus server-set fields | high |
| CardMove.column_status | data-model.md → cards.column_status | high |
| CardPage.items / has_next / has_prev / next_cursor | derived (cursor wrapper convention, api skill default) | high |
| PublicLink.id | data-model.md → public_links.id (UUID PK) | high |
| PublicLink.token | data-model.md → public_links.token — ADR-0003 (crypto/rand, not v7) | high |
| PublicLink.disabled_at | data-model.md → public_links.disabled_at | high |
| PublicLink.created_at | data-model.md → public_links.created_at | high |
| Error.code / message / details | derived (api skill's unified envelope convention) | high |
| subscribeCardEvents / subscribePublicBoardEvents response | sad.md §4 item 6 + §5 ports "SSE handler" — no dedicated data-model column (it's a transport, not an entity) | medium |
| getActivePublicLink `{link: null}` shape | inferred from AC-09's precondition "no active public link currently" — no dedicated column, a query-shape convenience | medium |

No `low`-confidence rows — every request/response field traces to a `data-model.md` column, an ADR, or a stated api-skill convention (cursor wrapper, error envelope).

## Section B — drift findings

1. **Endpoint ↔ data-model** *(core)* — ✓. Every card operation reads/writes `cards`; every public-link operation reads/writes `public_links`. `getPublicBoard`/`subscribePublicBoardEvents` read both (resolve via `public_links.token`, then read `cards`).
2. **Error code ↔ repo error definition** *(core)* — no central error registry found in `api/internal/modules/` yet (each module defines its own sentinels in `domain/domain.go` + maps them in `ports/errors.go`, per `.claude/rules/go-errors.md` — there's no cross-module code list to check against). Recorded as: **no error registry found — the codes below are this contract's proposal; `implement` defines the matching Go sentinels** (`tasks.name_required`, `tasks.card_not_found`, `tasks.link_not_found`, `tasks.invalid_column`) plus one shared code, `common.not_found`, which is new — see the note below. Not a ✗, per the drift-check reference's fallback for repos with no registry yet.
3. **Validation ↔ constraint** *(core)* — ✓. `Card.name`/`CardCreate.name`/`CardUpdate.name` `maxLength: 200` matches `cards.name VARCHAR(200)`; `assignee maxLength: 100` matches `cards.assignee VARCHAR(100)`; `column_status` enum `[todo, in_progress, done]` matches the three fixed columns (spec §3 Non-goals); `PublicLink.token` has no `maxLength` in the contract because `data-model.md` sizes it `VARCHAR(255)` as a container bound, not a business rule — left unconstrained in the schema deliberately (a generated token's exact length is an implementation detail, not something a client sends).
4. **OpenAPI ↔ sequence** *(supporting)* — ✓ with one note. Every sad.md §6 `alt`/`else` branch has a matching response: move-card's "write confirmed"/"write fails" → 200/(client-side failure, no dedicated server error — see below); add-card & edit-card's "name provided"/"empty or spaces only" → 201 or 200 / 400 `tasks.name_required`; the public-board flow's "link active"/"disabled or never valid" → 200 / 404 `common.not_found`. **Gap noted, not blocking:** sad.md's "Drag a card" flow's `else` branch ("write fails, e.g. connection lost") describes a *client-side* connectivity failure, which has no corresponding *server* error response to model — the contract has nothing to add here; this is expected, not a mismatch.

## New shared error code: `common.not_found`

The public-board-by-token endpoints (`getPublicBoard`, `subscribePublicBoardEvents`) deliberately use a **shared, module-neutral** code `common.not_found` instead of a `tasks.*` code, per ADR-0003 + spec AC-05: a `tasks.link_disabled` code would itself leak the information AC-05 forbids (that a board *did* exist at this address, just disabled). This is a new code with no existing repo precedent — flagged for `implement` to define as a genuinely shared sentinel (or the platform's existing generic-404 path, if `internal/platform/httputil` already has one; the brownfield scan found none, so this is likely new). **Resolution:** Accept as is — this is a deliberate security property of the contract, not a mistake to fix.

## Conflicts

None. No field disappeared from `data-model.md`; no existing `openapi.yaml` to diff against (first pass); no orphan sequences.

## Idempotency-Key

Not required on any operation — no sad.md §6 flow shows an automated retry-with-backoff note or an async external actor; all mutations are direct, synchronous, user-triggered, and (per ADR-0002) safely repeatable — a client retry of `moveCard`/`updateCard` is just another overwrite, harmless under last-write-wins.

## `events.md`

Not produced. The feature's only "async" behavior is the SSE push (ADR-0001), which is a live one-way HTTP stream documented as two regular operations (`subscribeCardEvents`, `subscribePublicBoardEvents`) with a `text/event-stream` response — not a message-bus/webhook contract the `events.md` template is shaped for (producer/consumers/payload/retry/DLQ). No dead-letter or retry semantics apply to a browser `EventSource` reconnect.

## Next stage

`screens tasks` — `target_surfaces` includes `web-frontend`, so the screens stage runs (not skipped) to produce the per-state screen manifest this contract's operations back.
