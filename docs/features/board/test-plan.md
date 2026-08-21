---
status: Draft
owner: "genkovich"
reviewers: ["genkovich", "Tech Lead"]
updated_at: "2026-08-20"
feature_size: "M"
---

# Test plan — board

Одна board, яку team member редагує напряму (create/edit/move/delete task) без входу в систему,
і яку будь-хто відкриває через public link виключно на перегляд — цей план мапить кожен
acceptance criterion спеки на конкретний тест, перш ніж він написаний.

## Levels

| Level | Scope | Strategy (generic — no tool names) |
|---|---|---|
| Integration | The board module (task/column/public-link operations) against a real dependency it owns. | An ephemeral real Postgres, spun up per suite (testcontainers-style), torn down after. |
| E2E-through-UI *(web-frontend)* | A user-story flow driven through the real UI, not just the API — quick-add, drag-and-drop, the public-link panel, the viewer page. | The flow exercised through the rendered UI against ephemeral dependencies. |
| Load | NFR validation for the numeric §6 targets (write latency, board-load latency, throughput). | The load tool already in your repo, or e.g. k6 or Locust. |
| Unit | <!-- N/A: every AC in this feature either touches persistence (column/link state) or a full UI flow — the user confirmed integration-level coverage is sufficient; no isolated pure-logic rule was carved out as its own unit test. --> |
| Contract | <!-- N/A: web-frontend is the only consumer of the backend contract in this feature (no second service integrates against openapi.yaml yet) — integration tests already assert the real response shape end to end. --> |
| Component | <!-- N/A: not selected during AC-level confirmation — UI behaviour (incl. the viewer's disabled write affordances, AC-10) is covered by e2e-through-UI instead of isolated component tests. --> |
| Visual-regression | <!-- N/A: no numeric or spec-mandated visual invariant flagged for this feature; drag-and-drop functional correctness is covered by e2e-through-UI. --> |

## AC coverage

| AC (spec.md §5) | Test name (intent-based) | Level | Expected outcome |
|---|---|---|---|
| AC-01 (US-01) — happy path | task with a non-empty title lands in the leftmost column | integration | task created, appears in leftmost column immediately |
| AC-02 (US-01) — error | task creation is blocked when the title is empty | integration | task not created, team member told the title is required |
| AC-03 (US-02) — happy path | editing a task's title/assignee persists and is reflected | integration | new values saved, shown on the board immediately |
| AC-04 (US-03) — happy path | dropping a task on a valid column updates its status | integration | task recorded in the new column, shown there |
| AC-05 (US-03) — domain invariant | dropping a task outside any valid column leaves it unchanged | integration | task stays in its previous column, as if nothing happened |
| AC-05b (US-03) — domain invariant (concurrent) | concurrent moves of the same task to different columns converge on one column | integration | task ends up in exactly one column — the last write applied — and every subsequent read (by anyone) agrees |
| AC-06 (US-04) — happy path | deleting a task removes it from the board for everyone | integration | task gone from the board, not shown to any viewer or team member |
| AC-07 (US-05) — happy path | a public link is issued when the board has none yet | integration, e2e-through-UI | team member receives a link that always shows the current state, read-only |
| AC-08 (US-06) — happy path | revoking the public link stops it from serving board state | integration, e2e-through-UI | the link no longer returns board state to anyone |
| AC-09 (US-07) — happy path | a viewer sees the current board state through a valid public link | integration, e2e-through-UI | viewer sees all columns and tasks, read-only |
| AC-10 (US-07) — authorization | a viewer's attempt to move/edit/delete a task via the public link is rejected | integration | the action is rejected, viewer told the board is read-only |
| AC-11 (US-06, US-07) — cross-context | opening a revoked/invalid public link is denied | integration, e2e-through-UI | no board state is shown, viewer told the link no longer grants access |

## Edge cases / error paths

<!-- Each error/authorization AC already has its own row above (AC-02, AC-05, AC-05b, AC-10, AC-11). -->
<!-- These add the boundary/failure cases the spec and sad.md imply beyond those named ACs. -->

- Malicious/HTML content in task title or assignee → stored and rendered as literal text, never executed (spec §6.1 abuse case: injection through free-text fields).
- More than 30 task creations from one client within a minute → further creations within that window are rejected (spec §6.1 rate limit).
- A viewer holds a live SSE connection at the moment its link is revoked → that connection is closed synchronously by the revoke operation, not left open until it naturally times out (sad.md §6, Flow 1/3 note — "відкликання закриває вже відкриті SSE-з'єднання, не лише блокує нові").
- The database is unavailable during a write (create/edit/move/delete) → the write fails cleanly, no partial state is persisted, caller is told the write did not succeed.

## Test data

- Seed strategy: no factory library — inline struct/row construction per test, matching the repo convention already used in `api/internal/modules/user/ports/handler_integration_test.go` (data-model.md "Test fixtures"). A shared `seedBoard(t, db)` helper inserts the one `boards` row; the three `columns` rows come from the staged seed migrations (`02_seed_board`, `04_seed_columns`), not from test-time fixtures.
- Integration dependency: an ephemeral real Postgres container spun up per suite (testcontainers-style, matching `dbtest.StartPostgres` already in the repo) — never a mocked store.
- Cleanup boundary: per-test — each test starts from a freshly seeded board/columns and truncates/rolls back its own tasks and public-link rows so runs are independent and order-agnostic.

## NFR validation (load)

- p95 latency of a write (create/edit/move/delete task) ≤ 300 ms (spec §6) → scenario: sustain a steady request rate against the write endpoints for a fixed duration, assert p95 write latency ≤ 300 ms.
- p95 latency of loading the board (team member or viewer) ≤ 500 ms (spec §6) → scenario: sustain a steady request rate against the board-read endpoint for a fixed duration, assert p95 load latency ≤ 500 ms.
- Throughput ≥ 20 req/s per instance (spec §6, "Measurement: smoke test у CI") → scenario: sustain ≥ 20 req/s for a fixed duration against a mixed read/write workload, assert no error-rate regression.

## CI placement

- On every PR: integration suite (fast enough against an ephemeral container, and it is where every AC's correctness lives).
- On every PR (web-frontend surface): the e2e-through-UI suite for the public-link and cross-context flows (AC-07/08/09/11).
- On schedule / pre-release: the load scenarios (NFR validation) — heavier and noisier than a per-PR gate warrants.
