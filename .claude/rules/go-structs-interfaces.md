---
paths: ["api/**/*.go"]
---

# Go structs & interfaces — api

<!-- Adapted from samber/cc-skills-golang@golang-structs-interfaces v1.1.3 (upstream 466ea6d). RULE form. Evals: .claude/evals/golang-structs-interfaces/. -->
<!-- RULE form: always-on. The consumer-side-interface + return-concrete discipline is what -->
<!-- keeps this repo's module layers decoupled and unit-testable with hand-written fakes. -->

Design for testability and clarity, not abstraction for its own sake. The two load-bearing rules:
**accept interfaces, return concrete structs**, and **define each interface where it is consumed**.
"The bigger the interface, the weaker the abstraction."

## MUST

- **Accept interfaces, return concrete types.** Constructors return the concrete struct: `NewCourseService(...) *CourseService`, `NewHandler(...) *Handler`, `NewPostgresCourseRepository(...) *PostgresCourseRepository`. Never return an interface from a constructor — callers lose access to fields/methods, and you can always assign a concrete value to an interface variable upstream.
- **Define interfaces on the consumer side, not the implementor.** This is the spine of the module layout here:
  - `app/ports.go` declares `CourseRepository` (and `Clock`) — the *service* consumes them; the infra adapter merely satisfies them.
  - `ports/handler.go` declares `CourseAppService` — the *handler* consumes it; the app service satisfies it without importing `ports`.
- **Keep interfaces small (1–3 methods).** Single-method ports take the `-er` form — `TokenValidator`, `OrgMemberLister`, `Clock`. Compose larger contracts from small ones rather than declaring one fat interface.
- **One receiver style per type — be consistent.** If any method needs a pointer receiver (mutation, or the struct holds a `sync.Mutex`/large state), all methods use a pointer. This repo's settled choices: `(s *Server)`, `(h *Handler)`, `(r *PostgresCourseRepository)`, `(t *Tx)`, `(c *Claims)`.
- **Type-assert with comma-ok** (`v, ok := x.(T)`) and honor canonical method names — `String()` for `fmt.Stringer`, `Close()` for `io.Closer`. Don't invent `ToString()`/`Release()`.

## SHOULD

- **Don't create an interface until a second implementation or a test fake demands it.** "Discover interfaces, don't design them." A port with one production impl is justified *here* because unit tests inject hand-written fakes (e.g. a fake `CourseRepository`, an injectable `Clock`) — that testability need is the trigger.
- **Make the zero value useful**, or lazy-init in methods. A struct field that's a nil map panics on first write; either design it usable at zero value or guard with `if m == nil { m = make(...) }` (see `go-safety.md`).
- **Embed to promote an API ("is-a"); use a named field for a dependency ("has-a").** Services hold their ports as named fields (`svc CourseAppService`, `repo CourseRepository`) — they *use* them, they don't expose them — so named field, not embedding.
- **Add a compile-time interface check next to the type** when satisfaction matters: `var _ CourseAppService = (*app.CourseService)(nil)`. Costs nothing at runtime; the build breaks the moment the type drifts.
- **Tag every exported field of a serialized struct** — `json:"…"` on DTOs (`*Request`/`*Response`), `db:"…"` where used. Untagged exported fields leak Go casing onto the wire.
- **Prefer generics over `any`** for type-safe helpers; reserve `any` for true boundaries (JSON decode, reflection).

## beer-lms specifics

- The four-layer module shape encodes these rules — `domain` (entities + `Err*` sentinels, no I/O), `app` (`*Service` + consumer-side `*Repository`/`Clock` ports + `*Params`), `ports` (HTTP `Handler` + `*Request`/`*Response` + consumer-side `CourseAppService`), `infra` (`Postgres*Repository` satisfying the app port). New modules mirror it.
- `outbox.OrgMemberLister` (in `internal/platform/outbox/relay.go`) is the small-consumer-interface pattern in the platform layer: the relay declares the 1-method port it needs, and the wiring layer supplies an adapter — see `go-design-patterns.md`.

## Enforce / see also

`revive`, `staticcheck`, `govet` catch interface/receiver inconsistencies — see the `go-lint` skill. Naming of interfaces/receivers is in `go-naming.md`; constructor/option/DI patterns in `go-design-patterns.md`. For depth see upstream `references/` in `golang-structs-interfaces`.
