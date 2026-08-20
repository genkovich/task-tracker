---
name: go-lint
description: "golangci-lint setup and usage for api — ships a conservative, drop-in assets/.golangci.yml (golangci-lint v2, Go 1.25) tuned to the repo's rules and its existing //nolint directives, plus how to install it, run make lint (golangci-lint run ./...), read the output, and suppress findings correctly. Use when configuring golangci-lint, adding a quality gate, interpreting a lint warning, or writing a //nolint directive. Also use when the user mentions golangci-lint, go vet, staticcheck, or revive."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code or similar AI coding agents. Targets api (Go 1.25, golangci-lint v2).
metadata:
  author: samber
  version: "1.2.2"
  openclaw:
    emoji: "🧹"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
        - golangci-lint
    install:
      - kind: brew
        formula: golangci-lint
        bins: [golangci-lint]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
---

<!-- Ported from samber/cc-skills-golang@golang-lint v1.2.2 (upstream 466ea6d). SKILL form. Evals: .claude/evals/golang-lint/. -->

**Persona:** You are a Go code-quality engineer. You treat linting as a first-class part of the workflow — not a post-hoc cleanup step — and you keep the config low-noise so it stays trusted.

**Modes:**

- **Setup mode** — adopting the config, choosing linters, wiring CI: follow "In beer-lms" then "Configuration".
- **Coding mode** — writing new Go: run `golangci-lint run ./...` (or `make lint`) on your changes before committing.
- **Interpret/fix mode** — reading output, suppressing a false positive, fixing existing findings: start from "Interpreting Output" and "Suppressing Lint Warnings".

**Dependencies:** `golangci-lint` v2 — `brew install golangci-lint` (or `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).

## In beer-lms

- **There is no committed `.golangci.yml`.** `make lint` runs `golangci-lint run ./...` inside `api/`, so today it lints on golangci-lint's defaults.
- **To adopt the curated config**, copy the shipped asset to where `make lint` will find it (the working dir of `make lint` is `api/`, so the file must sit at `api/.golangci.yml`):
  ```bash
  cp .claude/skills/go-lint/assets/.golangci.yml api/.golangci.yml
  make lint        # = golangci-lint run ./...   (run from api/)
  ```
- **The config is deliberately conservative** so adoption is a clean pass, not a flood. Enabled linters and why each maps to a repo rule:

  | Linter | Why (rule) |
  | --- | --- |
  | `govet` | copylocks / lostcancel / loopclosure / printf — `go-safety`, `go-concurrency`, `go-context` |
  | `staticcheck` | deprecations + real bugs — all rules |
  | `unused` | dead identifiers — `go-code-style` |
  | `errcheck` (+`check-type-assertions`) | every error checked; comma-ok asserts — `go-errors`, `go-safety` |
  | `ineffassign` | assignments never read — `go-code-style` |
  | `errorlint` | `errors.Is/As` + `%w`, never `err == sentinel` — `go-errors` |
  | `errname` | `ErrFoo` / `FooError` naming — `go-naming` |
  | `nilerr` | `return nil` when err is non-nil — `go-errors` |
  | `revive` (light ruleset) | MixedCaps, stutter, error-strings, receiver-naming, indent-error-flow — `go-naming`, `go-code-style`, `go-structs-interfaces` |
  | `misspell`, `predeclared`, `unconvert` | spelling, builtin shadowing, redundant casts — `go-naming`, `go-code-style` |
  | `forcetypeassert` | bare `x.(T)` — `go-safety` |
  | `containedctx` | ctx stored in a struct — `go-context` |
  | `bodyclose` | unclosed HTTP bodies — `go-concurrency` (resources) |
  | `testifylint`, `thelper` | testify usage + `t.Helper()` — `go-tests` |
  | `nolintlint` | well-formed `//nolint` directives |

- **`nolintlint` is tuned for the repo's existing directives.** It is set `require-specific: true` (the linter name is mandatory — no bare `//nolint`) but `require-explanation: false`, because the repo uses `defer tx.Rollback(ctx) //nolint:errcheck` **without** a reason (and `//nolint:ineffassign,wastedassign // …` with one). Both pass as-is. Tighten `require-explanation` to `true` only after backfilling reasons.
- **Left OFF on purpose** (high-noise / opinionated): `gosec`, `funlen`, `gocyclo`, `gocognit`, `dupl`, `wsl_v5`, `godot`, `mnd`, `wrapcheck`, `err113`, the full `revive` ruleset, `paralleltest`, and the `database/sql`-only `sqlclosecheck`/`rowserrcheck` (this repo is pgx-native, so those would never fire usefully). Enable them incrementally once the baseline is green — e.g. add `paralleltest` when you're ready to enforce `t.Parallel()`, or `gosec` behind a security pass.
- **`docs/` (generated swagger) and `migrations/` are excluded** from the relevant linters; `_test.go` relaxes `errcheck`/`forcetypeassert`.

> **Verification note:** `golangci-lint` is not installed in the authoring environment, so this config was **schema-checked by hand against the v2 format**, not run through `golangci-lint config verify`. Before relying on it, run `golangci-lint config verify` and a first `golangci-lint run ./...` to confirm it's clean on the current tree (see below).

## Quick Reference

```bash
golangci-lint run ./...                 # run all enabled linters (= make lint)
golangci-lint run --fix ./...           # auto-fix what's fixable
golangci-lint fmt ./...                 # run the formatters stage (gofmt/gofumpt)
golangci-lint config verify             # validate the .golangci.yml schema
golangci-lint run --enable-only govet ./...   # one linter only
golangci-lint linters                   # list available linters in your binary
```

## Configuration

The [shipped .golangci.yml](./assets/.golangci.yml) is a golangci-lint **v2** config: `default: none` + an explicit `enable:` list (so a tool upgrade can't silently switch on new linters), a `settings:` block (errcheck/nolintlint/revive), an `exclusions:` block (curated v2 presets + path/test rules), and a separate top-level `formatters:` stage. For the catalog of what each linter checks, see the **[linter reference](./references/linter-reference.md)**.

If your `golangci-lint` is still v1, run `golangci-lint migrate` after copying, or adjust the schema. The shipped config targets v2 (current golangci-lint).

## Suppressing Lint Warnings

Use `//nolint` sparingly — fix the root cause first.

```go
// Good: specific linter (+ optional reason). This is the repo's settled tx idiom:
defer tx.Rollback(ctx) //nolint:errcheck

// Good: specific linters WITH a reason:
argIdx++ //nolint:ineffassign,wastedassign // will be used by future filter params

// Bad: bare suppression — nolintlint flags this:
//nolint
_ = doThing()
```

Rules enforced by `nolintlint` in this config:

1. **`//nolint` MUST name the linter(s)** — `//nolint:errcheck`, never bare `//nolint`.
2. A justification (`// reason`) is encouraged but **not required** in this config (so the existing reason-less `//nolint:errcheck` lines pass). Prefer adding one anyway for anything non-obvious.
3. **Never suppress security/resource linters** (`bodyclose`, and `gosec`/`sqlclosecheck` if you later enable them) without a strong, written reason.

For patterns and anti-patterns, see **[nolint directives](./references/nolint-directives.md)**.

## Development Workflow

1. Run `golangci-lint run ./...` (or `make lint`) after every significant change.
2. `golangci-lint run --fix ./...` to auto-fix the mechanical findings.
3. `make fmt` (gofmt) before committing — accept formatter output verbatim (`go-code-style` rule).
4. **Wire `make lint` into CI** as a required gate so regressions can't merge. (For incremental adoption on a large diff, set `issues.new-from-rev: HEAD~1` to lint only changed code.)

## Interpreting Output

Each issue is:

```
internal/modules/courses/ports/handler.go:42:10: message describing the issue (linter-name)
```

The `(linter-name)` tells you which linter fired — look it up in the [reference](./references/linter-reference.md), fix it, or suppress with `//nolint:linter-name // reason` if it's a genuine false positive. `golangci-lint run --verbose` adds timing/context.

## Common Issues

| Problem | Solution |
| --- | --- |
| Config schema error after copy | `golangci-lint config verify`; if v1 binary, `golangci-lint migrate` |
| Too many issues on existing code | Set `issues.new-from-rev: HEAD~1` to lint only new code, clean up gradually |
| Linter not found | `golangci-lint linters` — the name may need a newer binary (this config targets v2) |
| `nolint` directive flagged | Add the linter name (`//nolint:errcheck`); bare `//nolint` is rejected |
| Slow on the whole module | Lower `run.concurrency` or add paths to `linters.exclusions.paths` |

## Cross-References

- → `go-code-style` rule — formatting/clarity conventions the formatters + revive enforce.
- → `go-naming` rule — identifier/error naming (revive, errname, predeclared).
- → `go-errors` rule — error-handling conventions (errcheck, errorlint, nilerr).
- → [linter reference](./references/linter-reference.md) — full per-linter catalog.

External: [golangci-lint docs](https://golangci-lint.run/), [linters list](https://golangci-lint.run/usage/linters/).
