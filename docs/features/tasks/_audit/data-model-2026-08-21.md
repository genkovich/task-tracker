# data-model audit — tasks — 2026-08-21

## Staged migrations (NOT in the live tree)

- `docs/features/tasks/migrations/01_create_cards.up.sql` / `.down.sql`
- `docs/features/tasks/migrations/02_create_public_links.up.sql` / `.down.sql`

Migrations are staged — not yet in the live `api/migrations/` tree. `implement` promotes them (assigning the real sequence number) when it runs the `layer: migration` task.

## Promote-time convention hint

The repo's live `api/migrations/` uses **sequential 6-digit ordinals** (`000001`…`000005`, `api/migrations/`, `golang-migrate` — statement-per-file transaction semantics). Next real number at promotion time ≈ `000006` (`implement` re-checks at promote time in case another feature lands first).

## Conventions detected + followed

Derived from `api/migrations/000002_create_users.up.sql` (no `architecture-map.md` exists — `survey` hasn't run on this repo):

| Topic | Repo convention | Applied here |
|---|---|---|
| Naming | `NNNNNN_verb_entity` | staged as `NN_verb_entity` (feature-local ordinal, per this skill's staging discipline) |
| PK | `UUID`, app-generated (`uuid.NewV7()`, `api/CLAUDE.md`) | followed for `cards.id` / `public_links.id`; **deviated** for `public_links.token` — see below |
| Audit columns | `created_at` + `updated_at` `TIMESTAMPTZ NOT NULL DEFAULT now()` | followed on `cards`; `public_links` has `created_at` only (no `updated_at` — rows are insert-only, never mutated after create except the one `disabled_at` write) |
| Constraints | no `CHECK`, no triggers anywhere in the repo | followed — `column_status` validity is enforced at the app layer (domain sentinel), not a DB `CHECK` |
| String types | `VARCHAR(N)` sized to a known field limit, `TEXT` for free-form | followed — `VARCHAR(200)`/`VARCHAR(100)` sized from spec §6 NFR field-length limits |

**Deliberate deviation:** `public_links.token` is `crypto/rand`-backed (or UUIDv4), explicitly **not** the repo's default `uuid.NewV7()` — v7 encodes a creation timestamp, which would make the token's age inferable and weaken the unpredictability ADR-0003 requires. This is documented in ADR-0003 and `data-model.md`, not a silent divergence.

## Self-check (4/4 pass)

1. **Naming** — matches the repo's `verb_entity` shape (feature-local ordinal per staging discipline). Pass.
2. **Down reversibility** — every `CREATE TABLE` has a matching `DROP TABLE IF EXISTS`. Pass.
3. **FK indexes** — no `REFERENCES` in either migration (the two tables are unrelated, no FK). Trivially pass — nothing to index.
4. **Convention adherence** — column/type vocabulary matches `api/migrations/000002_create_users.up.sql` (`UUID`, `VARCHAR(N)`, `TIMESTAMPTZ ... DEFAULT now()`), no unexplained divergence beyond the documented token-generation deviation above. Pass.

## Drift detection

N/A — the `tasks` Go module does not exist in `api/internal/modules/` yet (confirmed via the design-stage brownfield scan and a fresh `ls`). There is no domain layer to diff the schema against. `implement` is where the domain structs and this schema converge for the first time; nothing to drift-check at this stage.

## `<!-- TBD -->` items

None — every column, constraint, and index in `data-model.md` is decided, no open placeholders.

## Next stage

`api tasks` — the OpenAPI contract will expose `cards` CRUD + drag, and `public_links` generate/disable/resolve, per `sad.md` §5's `ports` handler list.
