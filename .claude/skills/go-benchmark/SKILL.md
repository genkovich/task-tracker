---
name: go-benchmark
description: "Writing, running, and comparing Go benchmarks for api — func BenchmarkX(b *testing.B) with b.Loop() (Go 1.25), b.ReportAllocs/b.ReportMetric, sub-benchmarks, go test -bench=. -benchmem, statistical comparison with benchstat, and profiling (CPU/mem/trace) straight from a benchmark. Use when you need to measure a hot path — e.g. a pgx repository row→DTO mapping loop or a chi handler — before optimizing. Measurement methodology only; for the fix to apply once a bottleneck is found, see the go-performance skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents. Targets api (Go 1.25).
metadata:
  author: samber
  version: "1.2.4"
  openclaw:
    emoji: "📊"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
        - benchstat
    install:
      - kind: go
        package: golang.org/x/perf/cmd/benchstat@latest
        bins: [benchstat]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(benchstat:*) Bash(golangci-lint:*) Bash(git:*) Agent mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-benchmark v1.2.4 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-benchmark/. -->

**Persona:** You are a Go performance-measurement engineer. You never draw conclusions from a single run — statistical rigor and controlled conditions come before any optimization decision.

**Thinking mode:** Use deep reasoning for benchmark and profile interpretation — shallow analysis misreads profiling data and produces unsound conclusions.

**Dependencies:** `benchstat` — `go install golang.org/x/perf/cmd/benchstat@latest`.

Performance can't be improved if it isn't measured. This skill is the measurement workflow: write a benchmark, run it, profile it, compare before/after with statistical rigor. For the optimization patterns to apply **after** you've measured ("if X bottleneck, apply Y"), see the `go-performance` skill.

## In beer-lms

- **Go 1.25 ⇒ use `b.Loop()`** for new benchmarks (it times only the loop body and keeps args/results alive, avoiding dead-code-elimination mistakes).
- **No Makefile bench target** — run benchmarks ad hoc, scoped to the package under test:
  ```bash
  go test -bench=. -benchmem ./internal/modules/courses/...
  ```
- **Sensible targets here:**
  - A **pgx row→DTO mapping loop** — e.g. the `scanCourse` / `scanLessonNoBlocks` helpers in `internal/modules/courses/infra/postgres_course_repository.go`, or the capacity-hinted DTO loops in `internal/modules/courses/ports/handler.go` (`make([]CourseResponse, 0, len(courses))`). A benchmark is how you *justify* a capacity hint (see the `go-safety` rule) instead of guessing.
  - A **hot chi handler path** in `internal/modules/courses/ports/handler.go`.
- **Don't micro-optimize money/points** — they're `shopspring/decimal` (a correctness requirement, `go-safety` rule), not a place to swap in `float64` for speed.
- Benchmarks live next to the code as `Benchmark…` functions in the package's `_test.go` (same files as the unit tests). They don't run under `make test` unless you pass `-bench`.

## Writing Benchmarks

### `b.Loop()` (Go 1.24+) — preferred

```go
func BenchmarkScanCourses(b *testing.B) {
    rows := fixtureCourseRows(1000) // setup — excluded from timing
    for b.Loop() {
        _ = mapRowsToDTOs(rows)     // the compiler cannot eliminate this
    }
}
```

Legacy `b.N` loops still compile and are fine when preserving old benchmarks, but are easier to get wrong (setup may need `b.ResetTimer()`, and the result may need a sink). Prefer `b.Loop()` for anything new.

### Memory tracking

```go
func BenchmarkMapDTO(b *testing.B) {
    b.ReportAllocs() // or pass -benchmem
    var sink []CourseResponse
    for b.Loop() {
        sink = mapRowsToDTOs(rows)
    }
    _ = sink
}
```

`b.ReportMetric` adds a custom metric (e.g. throughput):

```go
b.ReportMetric(float64(rowsProcessed)/b.Elapsed().Seconds(), "rows/s") // b.Elapsed() valid inside b.Loop()
```

### Sub-benchmarks and table-driven

```go
func BenchmarkEncode(b *testing.B) {
    for _, size := range []int{64, 256, 4096} {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := make([]byte, size)
            for b.Loop() {
                Encode(data)
            }
        })
    }
}
```

## Running Benchmarks

```bash
go test -bench=BenchmarkScanCourses -benchmem -count=10 ./internal/modules/courses/... | tee bench.txt
```

| Flag | Purpose |
| --- | --- |
| `-bench=.` | run all benchmarks (regexp filter) |
| `-benchmem` | report allocations (B/op, allocs/op) |
| `-count=10` | run 10 times for statistical significance |
| `-benchtime=3s` | minimum time per benchmark (default 1s) |
| `-cpu=1,2,4` | run with different GOMAXPROCS |
| `-cpuprofile=cpu.prof` | write a CPU profile |
| `-memprofile=mem.prof` | write a memory profile |
| `-trace=trace.out` | write an execution trace |

**Output:** `BenchmarkScanCourses/size=64-8  5000000  230.5 ns/op  128 B/op  2 allocs/op` — `-8` is GOMAXPROCS, `ns/op` time per op, `B/op` bytes/op, `allocs/op` heap allocations per op.

## Comparing With benchstat

Never claim an improvement from a single run. Capture before/after with `-count=10`, then compare:

```bash
go test -bench=BenchmarkScanCourses -benchmem -count=10 ./internal/modules/courses/... > old.txt
# ... make the change ...
go test -bench=BenchmarkScanCourses -benchmem -count=10 ./internal/modules/courses/... > new.txt
benchstat old.txt new.txt
```

benchstat reports the delta with a p-value and confidence interval. A row marked `~` means **no statistically significant difference** — you cannot claim a win. See [benchstat Reference](./references/benchstat.md).

## Documenting Results in Commits

When a change has a measurable performance impact, paste the benchstat output in the commit body — it documents *why* the optimization exists and lets reviewers verify the claim.

```
perf(courses): cut row→DTO allocations 50% with a presized slice

          │    old     │              new               │
          │   B/op     │   B/op      vs base            │
ScanCourses  1.024Ki ± 0%  0.512Ki ± 0%  -50.00% (p=0.000 n=10)
```

Rules: include only benchmarks the change actually affects; never paste a `~` row as evidence; include the `goos/goarch/cpu` line so results are reproducible; use the `perf(scope):` commit type.

## Profiling From Benchmarks

Generate profiles directly from a benchmark run — no running server needed:

```bash
go test -bench=BenchmarkScanCourses -cpuprofile=cpu.prof ./internal/modules/courses/...
go tool pprof cpu.prof

go test -bench=BenchmarkScanCourses -memprofile=mem.prof ./internal/modules/courses/...
go tool pprof -alloc_objects mem.prof    # GC churn; -inuse_space for live heap

go test -bench=BenchmarkScanCourses -trace=trace.out ./internal/modules/courses/...
go tool trace trace.out
```

See [pprof Reference](./references/pprof.md) and [Trace Reference](./references/trace.md).

## Reference Files

- **[benchstat Reference](./references/benchstat.md)** — statistical comparison of runs (p-values, confidence intervals, regression detection). Use to prove a change is real, not lucky.
- **[pprof Reference](./references/pprof.md)** — interactive/non-interactive CPU, memory, and goroutine profile analysis: *where* time and memory go.
- **[Trace Reference](./references/trace.md)** — execution tracer: goroutine scheduling, GC phases, blocking — *when* and *why* code runs, when pprof isn't enough.
- **[Compiler Analysis](./references/compiler-analysis.md)** — escape analysis (`go build -gcflags=-m`), inlining decisions, assembly — verify the compiler did what you intended when you see unexpected allocations.
- **[Diagnostic Tools](./references/tools.md)** — focused diagnostics: `fieldalignment` (struct padding), GODEBUG flags, the race detector.

> **Production profiling is out of scope here.** beer-lms has no Prometheus/OpenTelemetry/Pyroscope instrumentation wired into its code (OTel appears only as an indirect dependency). To profile a *running* service you'd expose `net/http/pprof` behind auth and capture live profiles — that's a deployment concern, not covered by this measurement skill.

## Cross-References

- → `go-performance` skill — the optimization patterns to apply once a bottleneck is measured.
- → `go-safety` rule — capacity hints (`make([]T, 0, n)`) and `decimal`-over-`float64`; benchmark to justify the hint.
- → `go-testing` skill — general testing; benchmarks live alongside the unit tests.

## References

- [Go testing/benchmarks](https://pkg.go.dev/testing#hdr-Benchmarks)
- [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [pprof](https://github.com/google/pprof)
