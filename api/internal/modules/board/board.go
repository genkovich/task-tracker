// Package board wires the board module's own layers (infra -> app -> ports)
// behind a single Handler, following the per-module New(...) convention
// (CLAUDE.md §Architecture, sad.md §5: "New(...) *ports.Handler, wired in
// api/cmd/api/main.go — no authMW (ADR-0001 context, no accounts)").
package board

import (
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

// BoardID is the single board's fixed id (CONTEXT.md invariant: the product
// always has exactly one board), seeded by migration 000007_seed_board.
var BoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// Handler aggregates every board HTTP surface behind one RouteRegistrar:
// the team-editor board/task/public-link routes, the team-editor and
// public-viewer SSE streams (ADR-0002), and the public-viewer read-only
// routes (ADR-0003). Each sub-handler's RegisterRoutes already registers
// full "/api/v1/..." paths (ports/*.go), so Handler must be mounted on the
// server's root router directly, not nested under the shared "/api/v1"
// registrar group other modules use — see cmd/api/main.go.
type Handler struct {
	board  *ports.BoardHandler
	task   *ports.TaskHandler
	link   *ports.LinkHandler
	sse    *ports.SSEHandler
	public *ports.PublicHandler
}

// New wires the board module from its infra (Postgres repo + in-process SSE
// hub, ADR-0002) up through app (task/link/state use-case services) to the
// aggregate ports Handler. Board is deliberately unauthenticated (ADR-0001,
// no accounts) — the caller registers Handler without authMW.
func New(db *database.DB) *Handler {
	repo := infra.NewPostgresRepository(db)
	hub := infra.NewHub()

	taskSvc := app.NewTaskService(repo, hub, BoardID)
	linkSvc := app.NewLinkService(repo, hub)
	stateSvc := app.NewStateService(repo)

	return &Handler{
		board:  ports.NewBoardHandler(stateSvc, BoardID),
		task:   ports.NewTaskHandler(taskSvc),
		link:   ports.NewLinkHandler(linkSvc, BoardID),
		sse:    ports.NewSSEHandler(hub, stateSvc),
		public: ports.NewPublicHandler(stateSvc),
	}
}

// RegisterRoutes mounts every board route (team-editor board/task/link,
// team-editor + public-viewer SSE, public-viewer board) on r.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.board.RegisterRoutes(r)
	h.task.RegisterRoutes(r)
	h.link.RegisterRoutes(r)
	h.sse.RegisterRoutes(r)
	h.public.RegisterRoutes(r)
}
