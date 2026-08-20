---
name: go-ci
description: "GitHub Actions CI for the api Go backend. Use when setting up or improving the repo's pipeline, adding a quality gate, or debugging a workflow. Ships assets/ci.yml — a ready-to-commit .github/workflows/ci.yml that runs the repo's real make targets (make vet, make lint, make test, make test-integration) plus govulncheck, on Go 1.25 with the module cache, scoped to the api/ subdirectory. Also ships dependabot.yml, codecov.yml, and an optional CodeQL config."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents, and for the api Go backend (Go 1.25) in a GitHub monorepo.
metadata:
  author: samber
  version: "1.3.1"
  openclaw:
    emoji: "🚀"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
        - gh
    install:
      - kind: brew
        formula: gh
        bins: [gh]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(gh:*) AskUserQuestion
---

<!-- Ported from samber/cc-skills-golang@golang-continuous-integration v1.3.1 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-continuous-integration/. -->
<!-- Adapted for api: the multi-file upstream workflow set was consolidated into a single assets/ci.yml driven by the repo's real make targets, scoped to the api/ subdirectory (the Go module is not at the repo root). Out-of-scope upstream assets were dropped: GoReleaser (cli/lib/monorepo), Docker build-push, release.yml, Renovate, Copilot/Claude-app review workflows, and the dependabot auto-merge workflow (elevated permissions). -->

**Persona:** You are a Go DevOps engineer for api. You treat CI as a quality gate, and you keep CI honest by running the *same* commands developers run locally — the Makefile targets — so green-on-laptop means green-in-CI.

**Modes:**

- **Setup** — the repo has no `.github/workflows/` yet: drop in `assets/ci.yml` at `.github/workflows/ci.yml`, then optionally add `dependabot.yml` / `codecov.yml`.
- **Improve** — auditing an existing pipeline: read the current workflow first, identify gaps against the job table below, add targeted jobs without duplicating steps.

# Go Continuous Integration — api

The deliverable is **[assets/ci.yml](assets/ci.yml)** → commit it to **`.github/workflows/ci.yml`** (workflows must live at the repo root). It runs the repo's real make targets.

## Key fact: the Go module is in a subdirectory

This is a monorepo — the Go module is in **`api/`** (the repo root also holds `web/`, a React/TS app, out of scope). So `ci.yml`:

- sets `defaults.run.working-directory: api`,
- caches against `cache-dependency-path: api/go.sum`,
- scopes triggers with `paths: ["api/**", ".github/workflows/ci.yml"]` so backend CI doesn't fire on web-only changes,
- passes `working-directory` / `work-dir` to the actions that don't honour `defaults` (golangci-lint-action, govulncheck-action).

If you ever add a root `go.work`, revisit these paths.

## What ci.yml runs

| Job | Command | Maps to | Notes |
| --- | --- | --- | --- |
| **vet** | `make vet` (+ `go mod verify`, `go mod tidy` drift check) | `make vet` | fails if `go.mod`/`go.sum` aren't tidy |
| **lint** | `golangci/golangci-lint-action` | `make lint` | reads `.golangci.yml` (from the `go-lint` skill) |
| **test** | `go test -race -shuffle=on -coverprofile` | `make test` | CI adds `-race -shuffle=on` that the bare target omits |
| **integration** | `go test -tags integration -p 4 -count=1 ./...` | `make test-integration` | needs Docker (testcontainers) — see below |
| **govulncheck** | `golang/govulncheck-action` | — | known-CVE scan of called code paths |

The unit `test` and `integration` jobs intentionally run **more** than the bare make targets: `make test` is `go test ./...`, but CI adds `-race` (data races are undefined behaviour) and `-shuffle=on` (catches inter-test coupling); the integration job adds `-count=1` (cached results can hide flaky service interactions). The make targets stay simple for local use; CI hardens them. Adapt the Go version matrix only if you want to test forward compatibility — the repo targets Go 1.25, so `ci.yml` pins `GO_VERSION: "1.25"` (add `"1.26"`, `"stable"` to a matrix if desired, with `fail-fast: false`).

## Integration tests need Docker (testcontainers)

The integration job runs `//go:build integration` tests that use **testcontainers-go** to start a throwaway `postgres:18-alpine`; Redis is faked in-process with **miniredis** (no Redis container needed). `ubuntu-latest` ships a working Docker daemon, so testcontainers works with **no `services:` block**. The tests self-`t.Skip` when Docker is unavailable, so the job degrades safely. (This is why `ci.yml` does not declare a `services: postgres` — testcontainers manages the container lifecycle itself, including the seed admin UUID and `database.RunMigrations`.)

## Linting

`golangci-lint` runs on every PR via `golangci/golangci-lint-action`. It reads **`.golangci.yml`** — that config is shipped by the **`go-lint` skill**, not this one. Place it at `api/.golangci.yml`; the action picks it up from the `working-directory`.

## Security & vulnerability scanning

- **`govulncheck`** runs as its own job (`golang/govulncheck-action` with `work-dir: api`). It reports only vulnerabilities in code paths the project actually calls. Fix or suppress findings with justification — don't ignore them. For a deeper sweep, pair with the **`go-security-auditor` agent** and the `go-dependency-management` skill.
- **CodeQL (optional):** [assets/codeql-config.yml](assets/codeql-config.yml) selects the `security-and-quality` query suite. To enable it, add a CodeQL job (`github/codeql-action` init/autobuild/analyze with `languages: go`) and reference this config; results land in the repo Security tab. Not in `ci.yml` by default to keep the required pipeline fast.
- **gosec (optional):** add `securego/gosec` as a job uploading SARIF if you want SAST beyond govulncheck (see the `go-security` rule for what it catches).

## Coverage (optional)

The `test` job already produces `coverage.out` and uploads it via `codecov/codecov-action`. To enforce thresholds, commit [assets/codecov.yml](assets/codecov.yml) at the repo root and set `CODECOV_TOKEN` in repo secrets. `fail_ci_if_error: false` keeps a Codecov outage from breaking the build.

## Dependency update automation

[assets/dependabot.yml](assets/dependabot.yml) → `.github/dependabot.yml`. It targets `gomod` at **`/api`** (the module root) and `github-actions` at the repo root, grouping minor/patch Go updates into one weekly PR. Renovate is a more configurable alternative but is intentionally not shipped here. Auto-merge is deliberately **not** included — it needs `contents: write`/`pull-requests: write`; gate merges with branch protection + required status checks instead (see [repo-security.md](references/repo-security.md)).

## AI-driven code review (optional, not shipped)

The Claude Code / Copilot PR-review workflows from upstream were dropped because they require the
Anthropic GitHub App (`/install-github-app`) and elevated PR-write permissions — out of scope for the
"run the real make targets" pipeline. The repo already has multi-agent review via the local `/codereview`
and `/code-review ultra` commands. If you later want CI-side AI review, add it as a separate workflow
with a tight trigger and least-privilege permissions; keep `ci.yml` focused on deterministic gates.

## Repository security settings

Branch protection, workflow permissions, secrets, and environments are the foundation under any CI — documented in [repo-security.md](references/repo-security.md). At minimum: protect `main`, require the `vet`/`lint`/`test` checks, and keep `permissions: contents: read` as the default in every workflow.

## In beer-lms

- **There is no `.github/workflows/` in the repo yet** — committing `assets/ci.yml` to `.github/workflows/ci.yml` is the first pipeline. Verify with `ls .github/workflows/` before assuming one exists.
- **The make targets are the contract.** `ci.yml` calls `make vet`/`make lint` semantics and the `go test` forms behind `make test` / `make test-integration`, so a green local `make vet && make lint && make test` predicts a green CI. Keep them in lockstep — if a target changes, update `ci.yml`.
- **`working-directory: api` everywhere.** Forgetting it is the #1 failure mode here — the module isn't at the repo root. Every `run:` inherits it via `defaults`; the two actions that ignore `defaults` get it explicitly.
- **Integration tests + Docker.** Don't add a `services: postgres` block — `dbtest.StartPostgres` (testcontainers) spins up `postgres:18-alpine` itself and `t.Skip`s without Docker; a stray service container just wastes time and diverges from local runs.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Running CI from the repo root | Set `working-directory: api` (module is in a subdir) |
| Missing `-race` in CI tests | The `test` job adds `-race` even though `make test` omits it |
| No `-shuffle=on` | Randomize test order to catch inter-test dependencies |
| Caching integration results | Use `-count=1` (already in the integration job) |
| `go mod tidy` not checked | The `vet` job runs `go mod tidy` + `git diff --exit-code` |
| Adding a `services: postgres` for integration | testcontainers manages it; just ensure Docker is present |
| Not pinning action versions | Pin majors (`@v5`, not `@master`); Dependabot bumps them |
| No `permissions` block | `permissions: contents: read` by default, widen per job |
| Ignoring govulncheck findings | Fix or suppress with justification |

## Related skills

→ `go-lint` skill (ships `.golangci.yml`). → `go-dependency-management` skill (`go mod tidy`, govulncheck, version pinning). → `go-testing` skill (the test forms CI runs). → `go-security` rule + `go-security-auditor` agent (gosec/CodeQL depth). → `go-benchmark` skill (adding a benchmark-regression job).

---

Upstream attribution: adapted from [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang) `golang-continuous-integration` (MIT, © samber).
