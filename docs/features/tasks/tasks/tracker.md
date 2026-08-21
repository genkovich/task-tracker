# Tracker — tasks

> Status of every task in the epic. `implement` updates `done` as it commits each task.
> States: `todo` · `in_progress` · `blocked` · `review` · `done`.

| # | Task | Layer | Owner | Estimate | Blocked by | Status |
|---|---|---|---|---|---|---|
| T1 | Promote staged cards + public_links migrations | migration | genkovich | S | — | todo |
| T2 | Card + PublicLink domain entities and sentinel errors | domain | genkovich | S | — | todo |
| T3 | PostgresCardRepository (CRUD + move) | infra | genkovich | M | T1, T2 | todo |
| T4 | PostgresPublicLinkRepository | infra | genkovich | S | T1, T2 | todo |
| T5 | Card service | app | genkovich | M | T2 | todo |
| T6 | PublicLink service + broadcaster | app | genkovich | M | T2 | todo |
| T7 | Card HTTP handlers | ports | genkovich | M | T3, T5 | todo |
| T8 | Public-link HTTP handlers | ports | genkovich | S | T4, T6 | todo |
| T9 | SSE handlers + public board resolve | ports | genkovich | M | T4, T6, T8 | todo |
| T10 | Wire the tasks module | wiring | genkovich | M | T7, T8, T9 | todo |
| T11 | Backend integration tests: concurrency | tests | genkovich | M | T10 | todo |
| T12 | SCR-01 Board page | ui | genkovich | M | — | todo |
| T13 | SCR-01 Add/edit card dialog | ui | genkovich | S | T12 | todo |
| T14 | SCR-01 Public-link control | ui | genkovich | S | T12 | todo |
| T15 | SCR-02/03 Public viewer page | ui | genkovich | M | — | todo |
| T16 | End-to-end smoke test | tests | genkovich | M | T10, T13, T14, T15 | todo |

**Total:** 16 tasks, ~11 person-days (S≈0.5d, M≈1d).
