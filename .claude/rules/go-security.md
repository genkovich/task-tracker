---
paths: ["api/**/*.go"]
---

# Go security — api

<!-- Adapted from samber/cc-skills-golang@golang-security v1.1.8 (upstream 466ea6d). RULE form (audit half -> go-security-auditor). Evals: .claude/evals/golang-security/. -->

Security follows **defense in depth**: protect at multiple layers, validate all inputs,
use secure defaults, and lean on the standard library's security-aware design. Every layer
protects itself even when an upstream layer already validates.

## MUST

- **Parameterized SQL only.** Pass user data as `$1, $2 …` through `pgx`; never build a query by string-concatenating input. Identifiers that can't be parameters come from an allow-list, never raw input.
- **Authenticate before authorizing.** Extract the bearer token, validate it, then check permissions **server-side on every protected handler** — never trust a client header or a client-side check.
- **Secrets come from env/config, never hardcoded.** No credentials, signing keys, or tokens in source — they leak into history, CI logs, and backups.
- **Validate input at the trust boundary** and **bound request size** (the body limit is already enforced — keep handlers from reading unbounded input another way).
- **Don't roll your own crypto.** Use `crypto/*` and `golang.org/x/crypto`; compare secrets with `crypto/subtle.ConstantTimeCompare`, not `==`; tokens from `crypto/rand`, never `math/rand`.
- **Fail closed.** On any error in an auth/crypto/authorization path, deny — never proceed as if it passed; always check crypto errors.

## SHOULD

- **Keep the security headers and rate limit on every route** (set in the server middleware, below) — don't bypass the global stack for a "quick" endpoint.
- **Return generic errors to clients,** log technical detail server-side — stack traces and DB errors help an attacker map the system (see `go-errors`).
- **Least privilege** for DB roles, S3/IAM, and tokens; per-environment secrets (a staging breach must not compromise prod).
- **Run `govulncheck` and `gosec`** before shipping risky changes (crypto, I/O, auth) — see the `go-lint` skill.

## beer-lms specifics

- **`authmw` (`internal/platform/authmw`)** is the auth layer: `extractBearerToken` reads `Authorization: Bearer …`, a `TokenValidator` validates it (golang-jwt/v5, in the auth module), and `*Claims{UserID, Email, Role}` lands in context. `Claims.IsAdmin()` gates admin actions. Context keys are the unexported `contextKey struct{}` — never bare strings.
- **Global middleware stack (`internal/server`)** every request passes: `securityHeaders` (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'`), then `requestSizeLimit(1 << 20)` (1 MB via `http.MaxBytesReader`), then `httprate.LimitByIP(60, time.Minute)`. New routes inherit these — keep it that way.
- **`pgx` `$1` parameterization** is the norm across `infra/` repos; constraint violations are detected with `database.IsPgUniqueViolation` / `IsPgForeignKeyViolation`, not by parsing error text.
- **Config from env** (`internal/platform/config`) — DSN, JWT secret, S3, Resend keys are read from the environment, never literals in code.

## Enforce / see also

`gosec`, `govulncheck`, `bodyclose`, `sqlclosecheck`, `nilerr` — see the `go-lint` skill.
For depth (injection, crypto, filesystem, secrets, threat modeling), upstream `references/` in `golang-security`.
For a full security audit, use the `go-security-auditor` agent.
