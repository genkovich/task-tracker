// Package board wires the board module's own layers (infra -> app -> ports)
// behind a single Handler, following the per-module New(...) convention
// (CLAUDE.md §Architecture, sad.md §5: "New(...) *ports.Handler, wired in
// api/cmd/api/main.go — no authMW (ADR-0001 context, no accounts)").
package board

import (
	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

// Handler aggregates every board HTTP surface behind one registrar: the
// team-editor boards/task/public-link routes, the public-viewer read-only
// routes (ADR-0003) — both via server.RouteRegistrar — and the team-editor +
// public-viewer SSE streams (ADR-0002) via server.StreamingRouteRegistrar,
// so the server keeps them off the per-request timeout. Passed into
// server.New(...) in cmd/api/main.go like every other module.
type Handler struct {
	board  *ports.BoardHandler
	task   *ports.TaskHandler
	link   *ports.LinkHandler
	sse    *ports.SSEHandler
	public *ports.PublicHandler
}

// New wires the board module from its infra (Postgres repo + in-process
// board-scoped SSE hub, ADR-0002) up through app (board/task/link/state
// use-case services) to the aggregate ports Handler. Board is deliberately
// unauthenticated (ADR-0001, no accounts) — the caller registers Handler
// without authMW.
func New(db *database.DB) *Handler {
	repo := infra.NewPostgresRepository(db)
	hub := infra.NewHub()

	boardSvc := app.NewBoardService(repo)
	taskSvc := app.NewTaskService(repo, hub)
	linkSvc := app.NewLinkService(repo, hub)
	stateSvc := app.NewStateService(repo)

	return &Handler{
		board:  ports.NewBoardHandler(boardSvc, stateSvc),
		task:   ports.NewTaskHandler(taskSvc),
		link:   ports.NewLinkHandler(linkSvc),
		sse:    ports.NewSSEHandler(hub, stateSvc, stateSvc),
		public: ports.NewPublicHandler(stateSvc),
	}
}

// RegisterRoutes mounts the request/response board routes (team-editor
// boards/task/link, public-viewer board) on r — the server's shared /api/v1
// registrar group, which carries the per-request timeout.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.board.RegisterRoutes(r)
	h.task.RegisterRoutes(r)
	h.link.RegisterRoutes(r)
	h.public.RegisterRoutes(r)
}

// RegisterStreamingRoutes mounts the long-lived SSE routes (ADR-0002) on r —
// the server's streaming group: same rate limit and metrics as everything
// else, but no per-request timeout, which would cut every stream at the
// timeout mark.
func (h *Handler) RegisterStreamingRoutes(r chi.Router) {
	h.sse.RegisterRoutes(r)
}
