---
status: Accepted
owner: "genkovich"
reviewers: []
updated_at: "2026-08-20"
feature_size: "S"
ticket: "docs/features/tasks/spec.md"
---

# 0002 — Resolve concurrent card writes as last-write-wins via a server-assigned timestamp

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

AC-07 requires that when two team members drag the same card to different columns nearly at once, the card ends up in the column from whichever change reached the system last, and both members see that final column — no error, no rejection for the "losing" write. AC-15 requires the same shape for a delete racing a drag on the same card: the delete wins, and the drag on an already-deleted card changes nothing. The write path (`tasks` module) and the database schema both need to agree on what "last" means before either is built.

## Decision drivers

- AC-07 / AC-15 domain invariant (spec §5) — a losing write must be silently overwritten, never rejected with an error.
- No accounts, no session/auth infrastructure for editing (spec §3 Non-goals, §6.1) — the client side of any request is inherently untrusted.
- Effort budget: size S (spec §1) — the mechanism should not need a version column, conflict-resolution logic, or a retry protocol.

## Considered options

1. **Server-assigned timestamp, unconditional overwrite** — every write runs `UPDATE ... SET column = $1, updated_at = now() WHERE id = $2` with no version check. "Last" means "the last write the server itself processed," which is naturally the last one to arrive over HTTP.
2. **Client-supplied timestamp, server keeps the highest** — the client sends its own timestamp with each write; the server compares and keeps whichever is newer.

## Decision outcome

**Chosen: Option 1.** With no login and no session state for editing (spec §3 Non-goals), any client's declared timestamp is unverifiable — a client with a fast or deliberately-skewed clock could always win under Option 2, which is a real risk once the write path has zero authentication behind it. Option 1 makes the server the single, trustworthy source of ordering and matches AC-07/AC-15's wording exactly ("the change that reaches the system last wins," not "the change with the largest declared timestamp wins"). It's also the simplest implementation available: no version column, no conflict branch, no retry story.

## Consequences

**Positive**
- No version column, no rejected writes, no client-side retry/reconciliation logic — the write path is a single unconditional `UPDATE`.
- Ordering authority sits entirely with the server, which is the only participant with no incentive to lie about time — important given editing has no authentication at all.
- Directly matches AC-07/AC-15's stated behavior with no translation layer.

**Negative**
- Depends on the database processing writes in true arrival order on one instance — correct for the current single-Postgres-instance deployment, but would need re-examination if the database were ever sharded or given a replica that could reorder writes relative to the primary.
- A losing write is invisible to the person who made it — they see no error, their change simply doesn't persist. This matches the spec's intent (no error UX for AC-07/AC-15) but is worth knowing if a future data-model iteration ever wants edit history.

**Neutral**
- Switching to optimistic locking later (rejecting a losing write instead of overwriting) is possible but would require a spec change first — AC-07/AC-15 explicitly rule out a reject-and-retry UX today.

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4, §6, §10
- Related ADR: none
