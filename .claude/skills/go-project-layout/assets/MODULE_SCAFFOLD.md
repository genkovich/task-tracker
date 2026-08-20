# New module scaffold — api modular monolith

Copy this skeleton when adding a new domain module. Replace `widget` / `Widget` with your
domain noun (lowercase singular package, MixedCaps type). Every path is under
`api/internal/modules/<domain>/`. This mirrors the real `courses` module — read it
alongside this file.

> api is a **modular monolith with manual DI**. A module is four packages with a
> strict dependency direction: `domain` ← `app` ← {`ports`, `infra`}. The composition root is
> `cmd/api/main.go` and nowhere else. Do NOT use the generic `pkg/` template — business code
> lives in `internal/`.

## File tree

```
internal/modules/widget/
├── widget.go                       # module wiring: New(db) *ports.Handler
├── domain/
│   ├── widget.go                   # entity + pure rules (no I/O, no framework imports)
│   ├── errors.go                   # Err* sentinels owned by this module
│   └── widget_test.go              # package domain_test — pure unit tests
├── app/
│   ├── ports.go                    # CONSUMER-side interfaces (WidgetRepository, Clock) + *Params
│   ├── service.go                  # WidgetService + NewWidgetService(...)
│   └── service_test.go             # package app_test — fakes implement the ports
├── ports/
│   ├── handler.go                  # chi Handler + NewHandler(svc) + RegisterOrgRoutes(r)
│   ├── dto.go                      # *Request / *Response (json tags)
│   ├── errors.go                   # mapError + errorMap (domain sentinel -> apperr.Error)
│   └── handler_test.go             # package ports_test — handler tests
└── infra/
    ├── postgres_widget_repository.go              # PostgresWidgetRepository + constructor
    └── postgres_widget_repository_integration_test.go  // first line: //go:build integration
```

Platform-wide concerns do NOT go in a module — they live in `internal/platform/<concern>/`
(`apperr`, `authmw`, `orgmw`, `httputil`, `database`, `logging`, `outbox`, `idempotency`,
`ratelimit`, `redis`, `storage`, `mailer`, `config`). If your module needs a new cross-cutting
capability, add a platform package, don't widen a module.

---

## 1. domain — entities + sentinels (no imports beyond stdlib + uuid/decimal)

`domain/errors.go` — sentinels are owned here. The error string is the wire-ish dotted code:

```go
// Package domain contains the core business types and rules for the widget module.
package domain

import "errors"

var ErrWidgetNotFound  = errors.New("widget.widget_not_found")
var ErrNameRequired    = errors.New("widget.name_required")
var ErrForbidden       = errors.New("widget.forbidden")
```

`domain/widget.go` — the entity and its invariants. Validation returns sentinels; no DB, no HTTP.

```go
package domain

import "github.com/google/uuid"

type Widget struct {
	ID      uuid.UUID
	OrgID   uuid.UUID
	Name    string
}

// Validate enforces the entity invariants, returning a sentinel on failure.
func (w *Widget) Validate() error {
	if w.Name == "" {
		return ErrNameRequired
	}
	return nil
}
```

## 2. app — use case service + consumer-side ports

`app/ports.go` — declare the interfaces the **service consumes** (not the implementor). Keep them
small; include an injectable `Clock` for determinism (mirror `courses/app/ports.go`).

```go
package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/widget/domain"
)

// WidgetRepository is the persistence port consumed by WidgetService.
// infra.PostgresWidgetRepository satisfies it without importing this package.
type WidgetRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Widget, error) // domain.ErrWidgetNotFound if absent
	Create(ctx context.Context, w *domain.Widget) error
}

// Clock is injectable so tests control "now".
type Clock interface{ Now() time.Time }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// RealClock returns a Clock backed by the system clock.
func RealClock() Clock { return realClock{} }

// CreateWidgetParams groups the inputs for the create use case (keep handlers' arg lists short).
type CreateWidgetParams struct {
	OrgID uuid.UUID
	Name  string
}
```

`app/service.go` — accept the ports, return the concrete struct.

```go
package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/widget/domain"
)

type WidgetService struct {
	repo  WidgetRepository
	clock Clock
}

func NewWidgetService(repo WidgetRepository, clock Clock) *WidgetService {
	return &WidgetService{repo: repo, clock: clock}
}

func (s *WidgetService) Create(ctx context.Context, p CreateWidgetParams) (*domain.Widget, error) {
	w := &domain.Widget{ID: uuid.Must(uuid.NewV7()), OrgID: p.OrgID, Name: p.Name}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}
```

## 3. ports — HTTP transport (chi handler + DTOs + error mapping)

`ports/errors.go` — table mapping domain sentinels to `apperr.Error` (mirror `courses/ports/errors.go`).
Codes are dotted snake_case (`widget.not_found`, `validation.invalid_body`).

```go
package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/widget/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

var errorMap = []struct {
	target error
	appErr apperr.Error
}{
	{domain.ErrWidgetNotFound, apperr.Error{Code: "widget.not_found", Message: "widget not found", StatusCode: http.StatusNotFound}},
	{domain.ErrNameRequired, apperr.Error{Code: "widget.name_required", Message: "name is required", StatusCode: http.StatusBadRequest}},
	{domain.ErrForbidden, apperr.Error{Code: "widget.forbidden", Message: "you do not have permission to perform this action", StatusCode: http.StatusForbidden}},
}

// mapError translates a domain error to a wire error, falling through unknown errors (-> 500).
func mapError(err error) error {
	for _, m := range errorMap {
		if errors.Is(err, m.target) {
			e := m.appErr
			return &e
		}
	}
	return err
}
```

`ports/handler.go` — early-return guards, then the happy path. The handler consumes a small
interface it declares itself (`WidgetAppService`), which `*app.WidgetService` satisfies. Pick the
registrar method by route shape (see step 5).

```go
package ports

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/widget/app"
	"github.com/genkovich/task-tracker/api/internal/modules/widget/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
	"github.com/genkovich/task-tracker/api/internal/platform/orgmw"
)

// WidgetAppService is the consumer-side port for the app layer.
type WidgetAppService interface {
	Create(ctx context.Context, p app.CreateWidgetParams) (*domain.Widget, error)
}

type Handler struct{ svc WidgetAppService }

func NewHandler(svc WidgetAppService) *Handler { return &Handler{svc: svc} }

// RegisterOrgRoutes mounts this module under /orgs/{orgId}/ (server.OrgScopedRouteRegistrar).
func (h *Handler) RegisterOrgRoutes(r chi.Router) {
	r.Post("/widgets", h.handleCreateWidget)
}

func (h *Handler) handleCreateWidget(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return
	}

	oc, ok := orgmw.OrgCtx(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "org.missing_context", "organization context is required")
		return
	}

	var req CreateWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	_ = claims // use claims.UserID / claims.IsAdmin() and oc.OrgRole / oc.IsMentor for authorization

	widget, err := h.svc.Create(r.Context(), app.CreateWidgetParams{OrgID: oc.OrgID, Name: req.Name})
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}
	httputil.WriteJSON(w, toWidgetResponse(widget), http.StatusCreated)
}
```

> The helper names above are the **real** ones in `internal/platform/httputil`:
> `WriteJSON(w, data, status)` (note the arg order: data then status), `WriteError(w, err)`, and
> `WriteValidationError(w, code, message)`. Bodies are decoded with `json.NewDecoder(r.Body).Decode`.
> `orgmw.OrgCtx(ctx)` returns `(*OrgContext, bool)` — always comma-ok, with `oc.OrgID`, `oc.OrgRole`,
> `oc.IsMentor`. This is the exact handler shape used in `internal/modules/courses/ports/handler.go`.

`ports/dto.go`:

```go
package ports

import "github.com/genkovich/task-tracker/api/internal/modules/widget/domain"

type CreateWidgetRequest struct {
	Name string `json:"name"`
}

type WidgetResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toWidgetResponse(w *domain.Widget) WidgetResponse {
	return WidgetResponse{ID: w.ID.String(), Name: w.Name}
}
```

## 4. infra — Postgres adapter (satisfies app.WidgetRepository)

```go
package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/widget/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type PostgresWidgetRepository struct{ db *database.DB }

func NewPostgresWidgetRepository(db *database.DB) *PostgresWidgetRepository {
	return &PostgresWidgetRepository{db: db}
}

func (r *PostgresWidgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Widget, error) {
	var w domain.Widget
	err := r.db.QueryRow(ctx, `SELECT id, org_id, name FROM widgets WHERE id = $1`, id).
		Scan(&w.ID, &w.OrgID, &w.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get widget: %w", err)
	}
	return &w, nil
}

func (r *PostgresWidgetRepository) Create(ctx context.Context, w *domain.Widget) error {
	_, err := r.db.Exec(ctx, `INSERT INTO widgets (id, org_id, name) VALUES ($1, $2, $3)`, w.ID, w.OrgID, w.Name)
	if err != nil {
		// Unique/FK violations -> domain sentinel via the platform helpers; never parse error text.
		// if database.IsPgUniqueViolation(err) { return domain.ErrWidgetExists }
		return fmt.Errorf("create widget: %w", err)
	}
	return nil
}
```

`widget.go` (module root) — the wiring seam returned to `cmd/api`. Mirror `courses/courses.go`:

```go
// Package widget wires the widget module and exposes the HTTP handler that
// satisfies server.OrgScopedRouteRegistrar.
package widget

import (
	"github.com/genkovich/task-tracker/api/internal/modules/widget/app"
	"github.com/genkovich/task-tracker/api/internal/modules/widget/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/widget/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

func New(db *database.DB) *ports.Handler {
	repo := infra.NewPostgresWidgetRepository(db)
	svc := app.NewWidgetService(repo, app.RealClock())
	return ports.NewHandler(svc)
}
```

## 5. Register in cmd/api/main.go

Construct the module and pass its handler to `server.New(...)` alongside the others. The handler is
picked up by whichever registrar interface it implements — choose by route shape:

| Route shape | Implement | Method |
| --- | --- | --- |
| Public (no auth) | `server.RouteRegistrar` | `RegisterRoutes(r chi.Router)` |
| Authenticated, not org-scoped | `server.ProtectedRouteRegistrar` | `RegisterProtectedRoutes(r chi.Router)` |
| Nested under `/orgs/{orgId}/`, needs org membership | `server.OrgScopedRouteRegistrar` | `RegisterOrgRoutes(r chi.Router)` |

A handler may implement more than one. In `cmd/api/main.go`:

```go
widgetHandler := widget.New(db)
// ... then include widgetHandler in the variadic module list passed to server.New(...),
// exactly as the existing module handlers are passed.
```

> The three interfaces are defined in `internal/server/server.go`. Read how the existing handlers
> are threaded into `server.New(...)` in `cmd/api/main.go` and follow that exact call shape — the
> registration mechanism is the contract; don't invent a new registry.

## 6. Migrations

Schema changes are NOT written by hand into `migrations/`. Use the SDD data-model flow (it stages
paired `*.up.sql` / `*.down.sql`), or `make migrate-create name=create_widgets` for the file pair,
then follow the `migrations` rule (every `up` has a tested `down`; expand-contract for risky
changes). The Go binary `cmd/migrate` and `database.RunMigrations(migrations.FS, dsn)` apply them
from the embedded FS.

---

## Initialization checklist (new module)

- [ ] Pick the domain noun: lowercase singular package `widget`, MixedCaps type `Widget`.
- [ ] `domain/` — entity + `Err*` sentinels + `Validate()`; pure, no framework imports; `*_test.go` in `package domain_test`.
- [ ] `app/ports.go` — small **consumer-side** interfaces (`WidgetRepository`, `Clock`) + `*Params` structs.
- [ ] `app/service.go` — `NewWidgetService(...)` returns `*WidgetService`; depends on `domain` only.
- [ ] `ports/` — `Handler` + `NewHandler`, `dto.go` (`*Request`/`*Response` with json tags), `errors.go` (`errorMap` + `mapError`).
- [ ] Pick the registrar method by route shape (public / protected / org-scoped).
- [ ] `infra/` — `PostgresWidgetRepository` + `NewPostgresWidgetRepository(db)`; pgx not-found → domain sentinel; constraint violations via `database.IsPgUniqueViolation`/`IsPgForeignKeyViolation`.
- [ ] `widget.go` — `New(db) *ports.Handler` wiring seam (mirror `courses/courses.go`).
- [ ] Register the handler in `cmd/api/main.go` and pass it to `server.New(...)`.
- [ ] Add a compile-time check where satisfaction matters: `var _ WidgetAppService = (*app.WidgetService)(nil)`.
- [ ] Migrations via the data-model flow / `make migrate-create`; every `up` has a `down`.
- [ ] Unit tests with hand-written fakes + injected `Clock`; integration test gated `//go:build integration` using `dbtest.StartPostgres`.
- [ ] `make fmt && make vet && make lint && make test` green; `make test-integration` if Docker is available.
```
