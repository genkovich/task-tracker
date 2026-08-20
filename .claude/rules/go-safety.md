---
paths: ["api/**/*.go"]
---

# Go safety & data structures — api

<!-- Adapted from samber/cc-skills-golang@golang-safety v1.2.1 + @golang-data-structures v1.1.4 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-safety/, .claude/evals/golang-data-structures/. -->
<!-- RULE form: always-on. Safety = preventing our own bugs (panics, silent corruption); -->
<!-- data-structures folded in for pre-sizing + container semantics. Concurrency lives in go-concurrency.md. -->

Defensive Go: treat every untested assumption about nil, capacity, and numeric range as a latent
crash. Security handles attackers; safety handles ourselves. Pick the right structure by its
memory and copy semantics, not by familiarity.

## MUST

- **Never write to a nil map.** It panics. Initialize (`make(map[K]V)`) or lazy-init in the method: `if r.items == nil { r.items = make(map[string]int) }`. Indexing a nil map is fine (zero value); writing is not.
- **Type-assert with comma-ok:** `v, ok := x.(T)`; a bare `x.(T)` panics on mismatch. This is exactly the `claims, ok := authmw.AuthClaims(ctx)` shape used across handlers.
- **A typed nil pointer inside an interface is not `== nil`** — the type descriptor makes it non-nil. For the nil case return an untyped `nil`, never a typed nil pointer variable.
- **`append` may alias its backing array.** If capacity allows, the appended slice shares memory with the source and mutations bleed across. Force a copy with the three-index form `s[:len(s):len(s)]` (or `slices.Clone`) before handing a slice to a caller who will mutate it.
- **Bounds-check before narrowing integer conversions.** `int64(3e9)` → `int32` wraps silently to a negative. Guard against `math.MaxInt32`/`MinInt32` (or the relevant bound) first, then convert.
- **Maps must not be accessed concurrently.** Concurrent read/write is a data race that crashes. Guard shared maps with `sync.Mutex`/`sync.RWMutex` — see `go-concurrency.md`.
- **`defer` runs at function exit, not per loop iteration.** Don't `defer f.Close()` inside a `for` — resources pile up until return. Extract the body to a function so the defer fires each iteration. (The repo's per-request `defer rows.Close()` / `defer tx.Rollback(ctx)` are correct because they're function-scoped.)
- **Never compare floats with `==`** — IEEE-754 is inexact (`0.1+0.2 != 0.3`). Use `math.Abs(a-b) < epsilon`, or `shopspring/decimal` for money (already a dependency here).
- **Never copy a struct that contains a `sync.Mutex`/`sync.WaitGroup`** (or other no-copy field). Pass it by pointer; `go vet` flags copies.

## SHOULD

- **Pre-size collections when the count is known or estimable** — `make([]T, 0, n)` / `make(map[K]V, n)`. Each unplanned slice growth copies the whole backing array (O(n)); each map growth rehashes. Don't speculatively over-allocate when the common case is tiny.
- **Design useful zero values.** `var mu sync.Mutex` and `var buf bytes.Buffer` work at zero value; a struct with a nil map field does not. Either make the zero value usable or lazy-init in methods (see `go-structs-interfaces.md`).
- **Guard integer division by zero** (`if count == 0 { … }`) — it panics, unlike float division which yields `±Inf`/`NaN`.
- **Return defensive copies of internal slices/maps from exported accessors** — otherwise callers mutate your struct's internals through the shared backing array. Unexport the field, expose `slices.Clone(c.hosts)`.
- **`sync.Once` for lazy one-time init**; it's exactly-once even under concurrency.
- **Map iteration order is randomized — never depend on it.** Sort keys (`slices.Sorted(maps.Keys(m))`) when you need a stable order. Likewise never depend on *when* a slice reallocates; the growth threshold is version-specific.
- **Watch slice-of-pointers / shared backing-array gotchas:** a `[]*T` and the loop value alias the same elements. (Go 1.25 here means the per-iteration loop variable is already fixed — the classic closure-capture bug is gone.)
- **Prefer generics over `any`** when the type set is known — the compiler catches mismatches that `any` defers to a runtime panic.

## beer-lms specifics

- Capacity hints are the established pattern: `make([]CourseResponse, 0, len(courses))` and `make([]app.ReorderItem, 0, len(req.Items))` in `internal/modules/courses/ports/handler.go`. Carry it into new mapping loops.
- Money/points are `shopspring/decimal` (pgx registers `pgx-shopspring-decimal` on connect) — never reach for `float64` arithmetic on those values.

## Enforce / see also

`errcheck`, `forcetypeassert`, `nilerr`, `govet`, `staticcheck` catch much of this — see the `go-lint` skill. Concurrency-specific safety is in `go-concurrency.md`; receiver/zero-value design in `go-structs-interfaces.md`. For depth see upstream `references/` in `golang-safety` and `golang-data-structures`.
