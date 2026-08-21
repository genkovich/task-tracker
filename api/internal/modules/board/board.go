// Package board wires the board module's own layers (infra -> app -> ports)
// behind a single Handler, following the per-module New(...) convention
// (CLAUDE.md §Architecture, sad.md §5: "New(...) *ports.Handler, wired in
// api/cmd/api/main.go — no authMW (ADR-0001 context, no accounts)").
package board

import (
	"net/http"

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
	board   *ports.BoardHandler
	task    *ports.TaskHandler
	comment *ports.CommentHandler
	link    *ports.LinkHandler
	sse     *ports.SSEHandler
	public  *ports.PublicHandler
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
	commentSvc := app.NewCommentService(repo, hub)
	linkSvc := app.NewLinkService(repo, hub)
	stateSvc := app.NewStateService(repo)

	return &Handler{
		board:   ports.NewBoardHandler(boardSvc, stateSvc),
		task:    ports.NewTaskHandler(taskSvc, stateSvc),
		comment: ports.NewCommentHandler(commentSvc),
		link:    ports.NewLinkHandler(linkSvc),
		sse:     ports.NewSSEHandler(hub, stateSvc, stateSvc),
		public:  ports.NewPublicHandler(stateSvc),
	}
}

// RegisterRoutes mounts the team-editor request/response board routes
// (boards/task/comment/link) on r — the server's shared /api/v1 registrar
// group, which carries the per-request timeout. The public-viewer routes are
// deliberately NOT mounted here — see RegisterHighTrafficRoutes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.board.RegisterRoutes(r)
	h.task.RegisterRoutes(r)
	h.comment.RegisterRoutes(r)
	h.link.RegisterRoutes(r)
}

// RegisterStreamingRoutes mounts the team-editor SSE stream (ADR-0002) on r —
// the server's streaming group: same rate limit and metrics as everything
// else, but no per-request timeout, which would cut every stream at the
// timeout mark. The public-viewer SSE stream is deliberately NOT mounted
// here — see RegisterHighTrafficRoutes.
func (h *Handler) RegisterStreamingRoutes(r chi.Router) {
	h.sse.RegisterRoutes(r)
}

// RegisterHighTrafficRoutes mounts the public-viewer reads (board fetch and
// task detail, tasks TSK-12) and the viewer SSE stream on the server's
// higher-rate-limit subtree (server.HighTrafficRouteRegistrar): many viewers
// behind a handful of shared IPs (a workshop room on one venue Wi-Fi) can open
// all of them at once, far past the default 60 req/min — and opening cards
// multiplies that further. The fetches keep the standard per-request timeout
// (timeoutMW); the SSE stream, like the team-editor one, must not — a timeout
// would cut it mid-flight.
func (h *Handler) RegisterHighTrafficRoutes(r chi.Router, timeoutMW func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(timeoutMW)
		h.public.RegisterRoutes(r)
	})
	h.sse.RegisterPublicRoutes(r)
}
