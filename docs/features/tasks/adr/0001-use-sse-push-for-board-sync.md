---
status: Accepted
owner: "genkovich"
reviewers: []
updated_at: "2026-08-20"
feature_size: "S"
ticket: "docs/features/tasks/spec.md"
---

# 0001 — Push board changes to open pages over Server-Sent Events

- **Status:** Accepted
- **Date:** 2026-08-20
- **Deciders:** genkovich (Architect), during the `design` Socratic walk

## Context

Every open board page — the team member's editor and a viewer's public-link page — must reflect a team member's change without the person doing anything. Spec §6 NFR sets a ≤5 s freshness bar for an already-open viewer page, and AC-08/AC-12 require a viewer to see the team's just-made change on open, and to auto-transition to not-found when the link is disabled while they're watching. The repo (a fresh base-tpl scaffold) has zero realtime infrastructure today — no WebSocket, no SSE, no push mechanism of any kind (confirmed by the brownfield scan).

## Decision drivers

- Freshness NFR: "Свіжість борди в глядача, поки сторінка вже відкрита ... ≤ 5 с" (spec §6).
- AC-08 (viewer sees the team's just-made change, not a stale snapshot) and AC-12 (viewer's page auto-transitions to not-found within a short time, no manual refresh) — both need the server to notify the client, not the other way around.
- Availability quality goal (§1) — the mechanism must not become a new source of downtime for a size-S feature.

## Considered options

1. **Client polling** — every open page (editor and viewer) re-fetches board state on a timer. Zero new infrastructure; trivial to implement and reason about.
2. **Server-Sent Events (SSE)** — the server pushes each board change down a long-lived one-way HTTP stream the moment it happens; the browser's built-in `EventSource` handles reconnects.
3. **WebSocket** — a full-duplex connection the server pushes over; the same push benefit as SSE but with bidirectional plumbing this feature doesn't need, since all writes already go through the existing REST endpoints.

## Decision outcome

**Chosen: Option 2, SSE.** The decision driver was end-to-end freshness with headroom under real network conditions, not just the stated 5 s average case — a push channel gives near-instant delivery instead of "at most one polling interval late," directly serving both AC-08 and AC-12 (the auto-transition-to-not-found case in particular reads naturally as "the disable event itself is one more pushed message"). SSE was chosen over WebSocket because the feature's data flow is one-directional (server → client only; all writes remain plain REST calls), so SSE's simpler one-way model, native browser reconnect, and plain-HTTP semantics are enough — a full-duplex WebSocket would add plumbing (upgrade handshake, connection-state management) this feature never uses.

## Consequences

**Positive**
- Viewers and the editor both see changes essentially immediately, with real headroom under the 5 s NFR, not up against it.
- `EventSource` reconnects automatically in the browser — no custom retry/backoff logic to write.
- One-way semantics keep the server implementation simple: a small in-process broadcaster (publish on write, fan out to subscriber channels), no connection-state machine.

**Negative**
- New infrastructure for a size-S feature: an in-process pub/sub broadcaster plus an SSE handler that didn't exist in the repo before.
- The SSE route needs an explicit exemption from the shared 30 s request-timeout middleware (`internal/server/server.go`), since the connection is meant to stay open indefinitely.

**Neutral**
- The in-process broadcaster only fans out within one `api` process. This is fine at today's single-instance deployment; if the API is ever horizontally scaled, the broadcast would need to move to a shared channel (e.g. Postgres `LISTEN`/`NOTIFY`) — tracked as accepted debt in sad.md §11, not built now.

## Links

- Spec: [[../spec.md]]
- SAD: [[../sad.md]] §4, §5, §6, §7
- Related ADR: none
