---
status: Accepted
owner: "genkovich"
reviewers: ["Security Lead"]
updated_at: "2026-08-20"
feature_size: "S"
ticket: "docs/features/tasks/spec.md"
---

# 0003 — Use an opaque random token with a `disabled_at` flag for the public board link

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

The public link is the feature's one authorization boundary (spec §6.1), and spec §6.1 explicitly calls for a Security review scoped to exactly this decision. Three acceptance criteria constrain the shape: AC-09 requires the link to be unpredictable and to stay valid until explicitly disabled; AC-04 requires disabling to take effect immediately for anyone holding the old link; AC-05 requires a disabled or never-valid link to resolve identically to any nonexistent address, with no signal that the board ever existed there.

## Decision drivers

- Unpredictability requirement (spec AC-09, §6.1 abuse case: "публічний лінк розлетівся далі за призначену аудиторію... адреса лінка непередбачувана заздалегідь").
- Immediate, permanent-until-re-enabled invalidation (AC-04) with no confirmation of prior existence for a disabled link (AC-05).
- No accounts/sessions anywhere in this feature (spec §3 Non-goals) — the token itself is the entire access-control mechanism.

## Considered options

1. **Opaque random token + `disabled_at` flag** — a 128-bit random (or UUIDv4) token stored in a `public_links` row; the row carries a nullable `disabled_at` timestamp. Every request checks `disabled_at IS NULL` before serving the board.
2. **Signed stateless token (JWT-style)** — the server issues a token whose validity is verified by signature alone, no database row, no per-request lookup.

## Decision outcome

**Chosen: Option 1.** AC-09 says the link "lishaаyetsya dijsnym, doky joho ne vymkneno" — stays valid until disabled, with no expiry implied — which a stateless signed token can't express without adding back exactly the state Option 2 was chosen to avoid (either a revocation list, defeating "stateless," or a short expiry that contradicts AC-09's "valid until disabled" wording). Option 1's single indexed lookup is cheap at this feature's scale (≥30 concurrent viewers, way below where a DB lookup per view would matter), and "disable" becomes one `UPDATE ... SET disabled_at = now()` — no data destroyed, no separate delete/undo story, and every open viewer's next check (poll or, per ADR-0001, SSE push) sees the same flag instantly.

## Consequences

**Positive**
- The token itself carries no information (no timestamp, no sequential id) — nothing to infer from it if intercepted.
- Disabling is a single, reversible-in-principle row update; re-enabling the same link is a deliberate non-feature (spec doesn't require it) but the data shape doesn't foreclose it either.
- One indexed lookup per request is simple to reason about and to security-review.

**Negative**
- Every board-view request costs one DB round trip to check `disabled_at`, unlike a self-verifying signed token. Acceptable at this feature's stated scale.

**Neutral**
- A generated-but-never-issued or already-disabled token must return the exact same not-found response as a token that was never valid at all (AC-05) — this is an implementation discipline the `ports` handler must uphold, not a property the storage shape provides automatically.

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4, §5, §6, §8
- Related ADR: none
