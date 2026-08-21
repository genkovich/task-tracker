package tasks

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

// Module wires the tasks board feature: card CRUD + drag, public-link
// generate/disable/resolve, and the SSE fan-out that keeps every open board
// page live (ADR-0001). No route it registers requires authentication —
// board editing has no login by design (spec §3 Non-goals).
type Module struct {
	card  *ports.CardHandler
	link  *ports.PublicLinkHandler
	sse   *ports.SSEHandler
	board *ports.PublicBoardHandler
}

func New(db *database.DB) *Module {
	broadcaster := app.NewBroadcaster()

	cardRepo := infra.NewPostgresCardRepository(db)
	linkRepo := infra.NewPostgresPublicLinkRepository(db)

	cardSvc := app.NewCardService(cardRepo, broadcaster)
	linkSvc := app.NewPublicLinkService(linkRepo, broadcaster)

	return &Module{
		card:  ports.NewCardHandler(cardSvc),
		link:  ports.NewPublicLinkHandler(linkSvc),
		sse:   ports.NewSSEHandler(broadcaster),
		board: ports.NewPublicBoardHandler(cardSvc, linkSvc, broadcaster),
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.card.RegisterRoutes(r)
	m.link.RegisterRoutes(r)
	m.sse.RegisterRoutes(r)
	m.board.RegisterRoutes(r)
}

func (m *Module) RegisterLongLivedRoutes(r chi.Router) {
	m.sse.RegisterLongLivedRoutes(r)
}

// RegisterHighTrafficRoutes puts the public board view + its SSE stream on
// the server's higher-rate-limit subtree — many viewers behind a handful of
// shared IPs may hit it at once (server.HighTrafficRouteRegistrar).
func (m *Module) RegisterHighTrafficRoutes(r chi.Router, timeoutMW func(http.Handler) http.Handler) {
	m.board.RegisterHighTrafficRoutes(r, timeoutMW)
}
