---
name: go-dependency-management
description: "go.mod / go.sum hygiene for the api module (github.com/genkovich/task-tracker/api, Go 1.25). Use when adding, upgrading, pinning, or removing a Go dependency, running go mod tidy / go mod verify, scanning for vulnerabilities with govulncheck ./..., resolving a version conflict with replace/exclude, keeping indirect deps clean, or reasoning about Minimal Version Selection. Pair with the go-security-auditor agent for a full vuln sweep."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents, and for the api Go backend (Go 1.25).
metadata:
  author: samber
  version: "1.2.4"
  openclaw:
    emoji: "📦"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
        - govulncheck
    install:
      - kind: go
        package: golang.org/x/vuln/cmd/govulncheck@latest
        bins: [govulncheck]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(govulncheck:*) AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-dependency-management v1.2.4 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-dependency-management/. -->
<!-- Adapted for api: grounded in the real module path + Go 1.25 toolchain; commands run from the api/ subdirectory. References to upstream-only skills (popular-libraries, pkg-go-dev, cli) were removed; cross-refs point at this repo's local skills/agents. -->

**Persona:** You are the api dependency steward. You treat every new dependency as a long-term maintenance commitment, and you ask whether the standard library or an already-present dependency solves the problem before adding a new one.

**Dependencies:**

- govulncheck: `go install golang.org/x/vuln/cmd/govulncheck@latest`

> **Where to run these.** The module lives in **`api/`** (monorepo; the repo root also holds the React/TS `web`). Run every `go` command below from `api/`.

# Go Dependency Management — api

## AI Agent Rule: Ask Before Adding Dependencies

**Before running `go get` to add any *new* dependency, ask the user for confirmation.** Agents can suggest packages that are unmaintained, low-quality, or unnecessary when the standard library — or a package already in `go.mod` — does the job. Using `go get -u` to upgrade an *existing* dependency is safe and does not need confirmation.

Before proposing a new dependency, evaluate:

- Does the standard library cover it? (Go 1.25 here: `slices`, `maps`, `cmp`, `min`/`max`, `errors.Join`, structured `log/slog`.)
- Is one of the **already-present** libraries enough? The repo already has chi, pgx/v5, golang-jwt/v5, google/uuid, golang-migrate, redis/go-redis, aws-sdk-go-v2 (s3), resend-go, shopspring/decimal, golang.org/x/sync, testify, testcontainers-go, miniredis — prefer extending these.
- Is the license compatible? Is it maintained? What does it pull in transitively?

When no in-repo or stdlib option exists, favour `golang.org/x/...` and well-known org-maintained packages over obscure ones.

## Key Rules

- **`go.sum` MUST be committed** — it records checksums of every dependency version so `go mod verify` can detect supply-chain tampering. A compromised proxy could otherwise substitute malicious code.
- **`govulncheck ./...` before every release** (or `go tool govulncheck ./...`) — catches known CVEs in code paths the project actually calls, before they ship.
- **`go mod tidy` before every commit that changes dependencies** — removes unused modules, adds missing ones; the CI `vet` job fails the build if `go.mod`/`go.sum` aren't tidy (see the `go-ci` skill).
- **Don't bump the `go` directive casually.** It's `go 1.25.0`; raising it changes the language/stdlib baseline and the CI matrix — do it deliberately, not as a side effect of adding a tool.

## go.mod & go.sum

### Essential Commands

| Command | Purpose |
| --- | --- |
| `go mod tidy` | Add missing deps, remove unused ones |
| `go mod download` | Download modules to the local cache |
| `go mod verify` | Verify cached modules match `go.sum` checksums |
| `go mod graph` | Print the module requirement graph |
| `go mod why` | Explain why a module or package is needed |
| `go mod edit` | Edit `go.mod` programmatically (scripts, CI) |

### Vendoring

api does **not** vendor (`vendor/` is git-ignored). The container build (`make docker-up`) uses the module proxy + `go.sum` checksums for reproducibility. Only introduce `go mod vendor` if you genuinely need hermetic, network-free builds — and then commit `vendor/` and run `go mod vendor` after every dependency change.

## Installing & Upgrading Dependencies

### Adding a Dependency

```bash
go get github.com/google/uuid          # latest
go get github.com/google/uuid@v1.6.0   # specific version
go get github.com/google/uuid@latest   # explicitly latest
go get github.com/google/uuid@<commit> # specific commit (pseudo-version)
```

Inspect available versions, importers, and known vulnerabilities on pkg.go.dev before pinning.

### Upgrading

```bash
go get -u=patch ./...   # latest patch only (safer — prefer this for routine updates)
go get -u ./...         # latest minor/patch for all direct + indirect deps
go get example.com/pkg@v1.5  # a specific package
```

Routine update flow (run from `api/`):

```bash
go get -u=patch ./...
go mod tidy
go test ./...
go vet ./...
govulncheck ./...        # or: go tool govulncheck ./...
```

Pay close attention to changelogs for libraries touching **persistence (pgx, golang-migrate), auth (golang-jwt), serialization, networking, S3, or money (shopspring/decimal)** — breaking changes there have the widest blast radius in this repo.

### Removing a Dependency

```bash
go get github.com/google/uuid@none
go mod tidy
```

### Installing CLI Tools (Go 1.24+ `tool` directives)

This module is Go 1.25, so pin executable tools in `go.mod` with `tool` directives — **do not** create a `tools.go` blank-import file (that's the Go <1.24 fallback).

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go get -tool golang.org/x/vuln/cmd/govulncheck@latest
go get -tool golang.org/x/perf/cmd/benchstat@latest

go tool golangci-lint run ./...   # run pinned tools reproducibly
go tool govulncheck ./...
go install tool                   # install all module-pinned tools into GOBIN
go get -u tool && go mod tidy     # update pinned tools deliberately
```

Pinning the toolchain this way makes local, CI (the `go-ci` pipeline), and contributor runs use the *same* linter/scanner versions.

## Deep Dives

- **[Auditing Dependencies](references/auditing.md)** — `govulncheck` modes, outdated-dependency tracking, test-only vs binary deps, binary-size analysis.
- **[Versioning & MVS](references/versioning.md)** — semver rules, pre-release versions, the Minimal Version Selection algorithm (why "latest" isn't automatic), `v2+` major-suffix conventions.
- **[Dependency Conflicts & Resolution](references/conflicts.md)** — diagnosing version conflicts, `replace` (local dev / forks), `exclude` (broken versions), `retract` (published versions to skip).
- **[Automated Updates](references/automated-updates.md)** — Dependabot/Renovate config and auto-merge trade-offs (the `go-ci` skill ships the Dependabot file).
- **[Visualizing the Graph](references/visualization.md)** — `go mod graph`, `modgraphviz`, finding bloat chains.
- **[Go Workspaces](references/workspaces.md)** — background only: api is a **single module**, so `go.work` is not used today; relevant only if a sibling Go module is ever added.

## In beer-lms

- **Module path & toolchain:** `github.com/genkovich/task-tracker/api`, `go 1.25.0`. New imports use this module prefix for internal packages (`github.com/genkovich/task-tracker/api/internal/...`).
- **Run from the subdirectory.** Because the module is in `api/`, `go mod tidy` / `govulncheck ./...` must run there, not at the repo root — and CI sets `working-directory: api` for the same reason (`go-ci`).
- **govulncheck is wired into CI.** The `govulncheck` job in `.github/workflows/ci.yml` (from the `go-ci` skill) scans on every PR. For an on-demand deep sweep — triaging a finding, checking a risky upgrade, mapping the call path to a vulnerable function — delegate to the **`go-security-auditor` agent**.
- **Constraint-handling deps are already chosen.** pgx error mapping uses `jackc/pgerrcode` via the platform helpers (`database.IsPgUniqueViolation` / `IsPgForeignKeyViolation`) — don't add another pg-error library. Decimal handling is `shopspring/decimal` (pgx registers `pgx-shopspring-decimal` on connect) — don't add a second money type.
- **Stay inside the vetted set.** Out of scope for this repo (do not add): samber/{lo,mo,do,oops,slog}, uber/{dig,fx}, google/wire, spf13/{cobra,viper}, grpc, graphql. If you think you need one, flag it and ask — the answer is usually stdlib or an existing dep.

## Cross-References

- → See the `go-ci` skill for the Dependabot file and the `govulncheck` CI job.
- → See the `go-security` rule + the `go-security-auditor` agent for vulnerability triage and crypto/secret review.
- → See the `go-benchmark` skill for `benchstat` (one of the `tool`-pinned binaries above).

## Quick Reference

```bash
# (run from api/)
go get github.com/google/uuid@v1.6.0   # add a pinned dependency
go get -u=patch ./...                  # upgrade (patch only, safer)
go mod tidy                            # remove unused / add missing
govulncheck ./...                      # known-CVE scan of called code
go mod why -m github.com/some/module   # why is this dep here?
go mod graph | modgraphviz | dot -Tpng -o deps.png  # visualize
go mod verify                          # checksums match go.sum
```

---

Upstream attribution: adapted from [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang) `golang-dependency-management` (MIT, © samber).
