---
paths: ["api/**/*.go"]
---

# Go concurrency — api

<!-- Adapted from samber/cc-skills-golang@golang-concurrency v1.1.4 (upstream 466ea6d). RULE form (audit half -> go-concurrency-auditor). Evals: .claude/evals/golang-concurrency/. -->

Goroutines are cheap but not free — every one you spawn is a resource you must manage.
Aim for structured concurrency: each goroutine has a clear owner, a predictable exit,
and proper error propagation. Correctness and leak-freedom come before performance.

## MUST

- **Every goroutine has a clear exit.** Before `go`, answer *how does it stop?* — `context` cancellation, a done channel, or `WaitGroup`. A goroutine with no shutdown path is a leak.
- **Always select on `ctx.Done()`** in any loop or blocking receive that a caller can cancel; return promptly when it fires.
- **Propagate `context.Context`** as the first parameter into anything that may block (I/O, DB, channel ops). Never store a context in a struct.
- **Only the sender closes a channel.** Closing from the receiver side panics if the sender writes after close. Specify direction (`chan<-`, `<-chan`) so the compiler enforces ownership.
- **Protect shared state.** Guard mutable fields with `sync.Mutex`/`RWMutex` (or a channel); keep critical sections short and **never hold a lock across I/O**. Concurrent map read/write is a hard crash.
- **Call `wg.Add` before `go`,** never inside the goroutine — `Wait` may otherwise return early.
- **Run `-race` in CI** (`make test` / `make test-integration`). Never ignore a race finding — races can bypass authorization under load.

## SHOULD

- **Use `errgroup` (`golang.org/x/sync`)** when you need first-error propagation, sibling cancellation (`errgroup.WithContext`), or a bounded pool (`SetLimit(n)`) — prefer it over a hand-rolled worker pool. Plain `sync.WaitGroup` is for fire-and-wait with no errors.
- **Send copies, not pointers,** over channels — a pointer creates invisible shared memory.
- **Default to unbuffered channels;** a buffer masks backpressure — add one only with a measured reason.
- **Don't add concurrency without a measured need** — a synchronous version is usually simpler and correct.
- **Avoid `time.After` in a hot loop** (each call allocates a timer); reuse `time.NewTimer` + `Reset`.

## beer-lms specifics

- **The outbox relay is the reference long-running goroutine** (`internal/platform/outbox/relay.go`). `Start(ctx)` runs `for { select { case <-ctx.Done(): return nil; default: } ... }` — it checks cancellation every iteration and exits cleanly, never leaking past shutdown.
- **Safe concurrent claim across instances:** `ClaimBatch` selects rows `FOR UPDATE SKIP LOCKED`, so multiple relay instances never process the same event. Mirror this for any "claim a job" query — don't lock the whole table.
- **Idempotent retries:** `InsertDedup` uses `ON CONFLICT DO NOTHING`, so a re-processed event can't double-insert. Make any retried side effect idempotent the same way; dead-letter only after `maxAttempts` (5).
- **Backoff:** the relay does `time.Sleep(r.pollInterval)` on empty/error batches — a deliberate poll cadence, not a busy loop.

## Enforce / see also

`govet` (copylocks, loopclosure), `-race`, and `containedctx` catch common mistakes — see the `go-lint` skill.
For depth, upstream `references/` in `golang-concurrency` (channels-and-select, sync-primitives, pipelines).
For a full concurrency/race sweep, use the `go-concurrency-auditor` agent.
