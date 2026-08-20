---
name: go-performance
description: "On-demand Go performance optimization for api — if X bottleneck, then apply Y. Covers allocation reduction, escape analysis (go build -gcflags=-m), sync.Pool, avoiding interface boxing, slice/map presizing, CPU/memory layout, GC tuning, and profiling a running chi/pgx service. Use when a profile or benchmark has identified a bottleneck and you need the right fix, or when reviewing a hot chi handler or a pgx repository scan for quick performance wins. Not for measurement methodology (→ go-benchmark skill) or pprof debugging workflow (→ go-troubleshooter agent)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents, and for the api Go backend (Go 1.25).
metadata:
  author: samber
  version: "1.2.2"
  openclaw:
    emoji: "🏎"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
        - benchstat
    install:
      - kind: go
        package: golang.org/x/perf/cmd/benchstat@latest
        bins: [benchstat]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(benchstat:*) Bash(staticcheck:*) Bash(curl:*) WebSearch AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-performance v1.2.2 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-performance/. -->
<!-- Adapted for api: scoped to api/**/*.go; the Prometheus/PromQL observability reference + prometheus-alerts asset were dropped (no metrics endpoint is wired in this repo — see the go-observability rule). Cross-references point at this repo's local skills/agents. -->

**Persona:** You are a Go performance engineer working on api. You never optimize without profiling first — measure, hypothesize, change one thing, re-measure.

**Thinking mode:** Use `ultrathink` for performance optimization. Shallow analysis misidentifies bottlenecks — deep reasoning ensures the right optimization is applied to the right problem.

**Modes:**

- **Review mode (architecture)** — broad scan of a module or the platform layer for structural anti-patterns (an unbounded goroutine, a per-request allocation that could be presized, an N+1 query). Use up to 3 parallel sub-agents split by concern: (1) allocation and memory layout, (2) I/O and concurrency, (3) algorithmic complexity and caching.
- **Review mode (hot path)** — focused analysis of a single chi handler, app-service method, or `infra/` repository scan identified by the caller. Work sequentially; one sub-agent is sufficient.
- **Optimize mode** — a bottleneck has been identified by profiling. Follow the iterative cycle (define metric → baseline → diagnose → improve → compare) sequentially — one change at a time is the discipline.

**Dependencies:**

- benchstat: `go install golang.org/x/perf/cmd/benchstat@latest`

# Go Performance Optimization

## Core Philosophy

1. **Profile before optimizing** — intuition about bottlenecks is wrong ~80% of the time. Use pprof to find actual hot spots (→ See the `go-troubleshooter` agent and the `go-benchmark` skill).
2. **Allocation reduction yields the biggest ROI** — Go's GC is fast but not free. Reducing allocations per request often matters more than micro-optimizing CPU.
3. **Document optimizations** — add a code comment explaining *why* a pattern is faster, with benchmark numbers when available. Future readers need context to avoid reverting an "unnecessary" optimization. This repo already writes reasoned inline notes (e.g. the capacity hints in `internal/modules/courses/ports/handler.go`).

## Rule Out External Bottlenecks First

Before optimizing Go code, verify the bottleneck is in your process — in api, most request latency is a Postgres round-trip. If 90% of latency is a slow query, reducing allocations won't help.

**Diagnose:** 1- `fgprof` — captures on-CPU and off-CPU (I/O wait) time; if off-CPU dominates, the bottleneck is external (the DB, S3, Resend, Redis). 2- `go tool pprof` (goroutine profile) — many goroutines blocked in `pgx`/`database/sql` = external wait. 3- the `middleware.RequestID` correlation ID already threaded through the server lets you line up a slow request's logs.

**When external:** optimize that component instead — query tuning, an index, caching, the connection pool (→ See the `go-database` skill and [Caching Patterns](references/caching.md)).

## Iterative Optimization Methodology

### The cycle: Define Goals → Benchmark → Diagnose → Improve → Benchmark

1. **Define your metric** — latency, throughput, memory, or CPU? Without a target, optimizations are random.
2. **Write an atomic benchmark** — isolate one function per benchmark to avoid result contamination (→ See the `go-benchmark` skill).
3. **Measure baseline** — `go test -bench=BenchmarkMyFunc -benchmem -count=6 ./internal/... | tee /tmp/report-1.txt`
4. **Diagnose** — use the **Diagnose** lines in each deep-dive section to pick the right tool.
5. **Improve** — apply ONE optimization at a time with an explanatory comment.
6. **Compare** — `benchstat /tmp/report-1.txt /tmp/report-2.txt` to confirm statistical significance.
7. **Commit** — paste the benchstat output in the commit body so reviewers and future readers see the exact improvement; follow the `perf(scope): summary` commit type.
8. **Repeat** — increment report number, tackle next bottleneck.

Refer to library documentation for known patterns before inventing custom solutions. Keep all `/tmp/report-*.txt` files as an audit trail.

## Decision Tree: Where Is Time Spent?

| Bottleneck | Signal (from pprof) | Action |
| --- | --- | --- |
| Too many allocations | `alloc_objects` high in heap profile | [Memory optimization](references/memory.md) |
| CPU-bound hot loop | function dominates CPU profile | [CPU optimization](references/cpu.md) |
| GC pauses / OOM | high GC%, container limits | [Runtime tuning](references/runtime.md) |
| Network / I/O latency | goroutines blocked on I/O | [I/O & networking](references/io-networking.md) |
| Repeated expensive work | same computation/fetch multiple times | [Caching patterns](references/caching.md) |
| Wrong algorithm | O(n²) where O(n) exists | [Algorithmic complexity](references/caching.md#algorithmic-complexity) |
| Lock contention | mutex/block profile hot | → See the `go-concurrency` skill / `go-concurrency-auditor` agent |
| Slow queries | DB time dominates | → See the `go-database` skill |

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Optimizing without profiling | Profile with pprof first — intuition is wrong ~80% of the time |
| Default `http.Client` without Transport | `MaxIdleConnsPerHost` defaults to 2; set it to match your concurrency level (relevant for the S3 / Resend clients) |
| Logging in hot loops | Log calls prevent inlining and allocate even when the level is disabled. Use `slog.LogAttrs` (this repo uses `log/slog`) |
| `panic`/`recover` as control flow | panic allocates a stack trace and unwinds the stack; use error returns (see the `go-errors` rule) |
| `unsafe` without benchmark proof | Only justified when profiling shows >10% improvement in a verified hot path |
| No GC tuning in containers | Set `GOMEMLIMIT` to 80-90% of the container memory limit to prevent OOM kills |
| `reflect.DeepEqual` in production | 50-200x slower than typed comparison; use `slices.Equal`, `maps.Equal`, `bytes.Equal` |

## Deep Dives

- [Memory Optimization](references/memory.md) — allocation patterns, backing-array leaks, `sync.Pool`, struct alignment, interface boxing.
- [CPU Optimization](references/cpu.md) — inlining, cache locality, false sharing, ILP, reflection avoidance.
- [Runtime Tuning](references/runtime.md) — GOGC, GOMEMLIMIT, GC diagnostics, GOMAXPROCS, PGO.
- [I/O & Networking](references/io-networking.md) — HTTP transport config, streaming, JSON performance, batch operations.
- [Caching Patterns](references/caching.md) — algorithmic complexity, compiled patterns, `singleflight`, work avoidance.

## In beer-lms

Concrete places this skill applies in api, grounded in the real code and conventions:

- **Escape analysis on a hot path.** `go build -gcflags="-m -m" ./internal/modules/<domain>/...` prints *why* a value escapes (`leaking param`, `moved to heap`, `captured by closure`). Run it on a handler or repository scan before reaching for `sync.Pool`; the cheapest win is keeping a short-lived value on the stack.
- **Presizing is already the house pattern — keep it.** Handlers build response slices with a capacity hint from the known input length: `make([]CourseResponse, 0, len(courses))` and `make([]app.ReorderItem, 0, len(req.Items))` in `internal/modules/courses/ports/handler.go`. Carry it into every new mapping loop; each unplanned `append` growth copies the whole backing array. (See the `go-safety` rule.)
- **Profiling a running chi handler.** pprof is *not* mounted on the server router today (`internal/server` wires cors → securityHeaders → RequestID → RealIP → Logger → Recoverer → Timeout(30s) → requestSizeLimit(1MB) → httprate). To profile, register `net/http/pprof` on a **separate, internal-only** `http.ServeMux` on a loopback/admin port — never behind the public router (it would sit inside the 30s timeout and the 1MB body limit, and must not be internet-exposed). Capture with `go tool pprof http://127.0.0.1:<admin>/debug/pprof/profile?seconds=30`.
- **Profiling a pgx repository scan.** The `Postgres*Repository` types in `infra/` scan rows in a `for rows.Next()` loop. If a heap profile shows the scan path hot, first confirm it isn't N+1 (one query per row) — that's a `go-database` fix, not an allocation fix. For a genuine per-row allocation, presize the result slice and reuse scan targets; don't pool the returned structs (callers keep them).
- **`sync.Pool` only with proof.** No pool exists in the repo yet, and most handlers don't need one. Reach for it only when a heap profile shows one allocation site producing thousands of short-lived objects per second (e.g. a per-request `[]byte` buffer in a future export/serialization path). Reset state before `Put`, return copies (never the pooled buffer), don't pool objects >32KB. (See [Memory Optimization](references/memory.md) and the `go-concurrency` skill for the API.)
- **Avoid interface boxing in numeric/decimal loops.** Passing concrete values through `any`/`interface{}` boxes each one onto the heap. Money/points are `shopspring/decimal` here — keep them typed end to end; prefer typed params or generics over `[]any` in any hot aggregation.
- **GOMEMLIMIT in the container.** api ships in a container (`make docker-up`). Set `GOMEMLIMIT` to ~80-90% of the container memory limit so the GC paces itself instead of getting OOM-killed; leave headroom for goroutine stacks and OS buffers. See [Runtime Tuning](references/runtime.md).

## CI Regression Detection

Automate benchmark comparison so a regression is caught before it ships. → See the `go-benchmark` skill for `benchstat` baselines, and the `go-ci` skill for wiring a benchmark job into `.github/workflows/`.

## Cross-References

- → See the `go-benchmark` skill for benchmarking methodology, `benchstat`, and `b.Loop()` (Go 1.24+).
- → See the `go-troubleshooter` agent for the pprof workflow, escape-analysis diagnostics, and performance debugging.
- → See the `go-database` skill for connection-pool tuning and batch processing (the most common real bottleneck here).
- → See the `go-concurrency` skill / `go-concurrency-auditor` agent for worker pools, the `sync.Pool` API, goroutine lifecycle, and lock contention.
- → See the `go-safety` rule for `defer`-in-loops and slice backing-array aliasing.

---

Upstream attribution: adapted from [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang) `golang-performance` (MIT, © samber).
