---
paths: ["api/migrations/*.sql"]
---

# Migration rules — beer-lms

<!-- Bootstrapped by sdlc/plugin/skills/generate-data-model on 2026-05-23. -->
<!-- Adapted from rules-migrations-baseline.md to match existing mentorship-module conventions. -->
<!-- Edit freely. Deviations from skill defaults are documented inline. -->

## Filenames

- Format: `<NNNNNN>_<verb>_<entity>.up.sql` + matching `.down.sql`. Sequential numbering, zero-padded to 6 digits.
- Current head: `000019_add_indexes`. Next available: `000020+`.
- **Deviation from skill default** (timestamp `YYYYMMDDhhmmss_*.up.sql`): repo started with sequential, continued for 19 migrations. Switching mid-stream would introduce mixed-naming reading-load. Trade-off: parallel feature branches can collide on the same next-number — mitigation: small team (1-2 ppl), coordinate on branch.

## Hard rules

- Every `.up.sql` has a matching `.down.sql` that fully reverses it.
- Every `REFERENCES other_table(id)` is paired with an index on the FK column (inline `CREATE INDEX`, or composite index whose first column is the FK).
- `CREATE TABLE` / `CREATE INDEX` / `ALTER TABLE ADD COLUMN` use `IF NOT EXISTS` whenever PostgreSQL supports it.
- Seed `INSERT` uses `ON CONFLICT DO NOTHING`.

## Allowed (deliberately permissive vs skill baseline)

These are **explicitly opted-in** for consistency with `internal/modules/mentorship/` (5 weeks production):

- `CHECK (status IN ('draft','published','archived'))` constraints on enum-style columns — DB serves as a second line of defense; app-layer validation remains the primary guard.
- `DEFAULT '<business literal>'` (e.g. `status DEFAULT 'draft'`) — lets handler INSERT without specifying every column; pattern matches `mentorship_sessions.status`.
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` on mutable entities — application sets `updated_at = now()` on `UPDATE`; column exists for indexing recent-changes queries and for cursor-pagination by `updated_at`.

## Forbidden

- `CREATE TRIGGER` / stored procedures — business logic stays in Go.
- `DEFAULT 'business literal'` on columns where the literal encodes business meaning **beyond a state-machine starting state** (e.g. `expiry_days DEFAULT 30` — NO; `status DEFAULT 'draft'` — OK).
- Sequences as PK (no `BIGSERIAL` / `SERIAL`).
- Multi-DB topology (read replicas, sharding) — single Postgres per environment.

## Defaults

- **PK:** `UUID` type, generated app-side via `google/uuid` `uuid.Must(uuid.NewV7())`. No DB-level `DEFAULT gen_random_uuid()` for new tables (that would yield UUID v4 — not v7). Per SAD §2.
- **Timestamps:** `TIMESTAMPTZ NOT NULL DEFAULT now()`.
- **Strings:** `VARCHAR(N)` bounded; `TEXT` only for URLs / long descriptions / opaque content.
- **Soft delete:** NOT used. Hard delete + audit table if business requires history (see `user_preference_audit`, `comment_audit`).
- **Audit columns:** `created_at` + `updated_at` on mutable entities; `created_at` only on immutable / event-log tables (audit, completions).
- **Naming:** `plural snake_case` tables (`users`, `lesson_blocks`, `comment_audit`), `snake_case` columns.
- **JSONB:** only for semantically opaque payload (e.g. `lesson_blocks.payload` for polymorphic block content per ADR-0001). Structured fields → first-class columns.

## Zero-downtime patterns (for existing-table changes)

- New NOT NULL column → 3-step (add nullable → backfill → SET NOT NULL).
- **Exception used in this repo (000014, 000020):** if backfill default has *no semantic meaning* (e.g. `is_mentor BOOLEAN NOT NULL DEFAULT false`, `is_methodist BOOLEAN NOT NULL DEFAULT false`) — single-step ALTER is acceptable since the default is structurally honest ("not a methodist yet"). For business-meaningful defaults — always use 3-step.
- New index on existing table → `CREATE INDEX CONCURRENTLY` (one statement per file — golang-migrate transaction wrapper does not allow CONCURRENTLY inside a tx).
- Rename / drop column → 3-step (add new + dual-write → backfill → drop old). Each phase = separate PR + deploy.

## Seeds

- **Bootstrap** (admin user, default org): hardcoded deterministic UUID v7 in a migration file. Existing pattern: `000003_seed_admin.up.sql`.
- **Lookup data** (statuses, currencies): separate migration, `INSERT ... ON CONFLICT DO NOTHING`.
- **Test fixtures:** NOT in `migrations/`. Repo's existing pattern is mock-based unit tests (`internal/modules/*/app/app_test.go`) — Go fixture factories not yet introduced. Introduce `internal/testfixtures/<entity>.go` when integration tests against a real DB are added.
- **PII guard:** no real-looking emails / names. Use `admin@example.test`, `user-<uuid>@example.test`, `Test User`.

## Out of scope

- Multi-DB (read replicas, sharding).
- Partitioning.
- Materialized views.

Owned by SRE / DBA; decided per-project with a separate ADR.
