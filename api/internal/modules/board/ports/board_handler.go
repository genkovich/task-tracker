package ports

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// BoardStateService is the read port BoardHandler depends on (AC-07/AC-08 —
// BoardState must reflect the current public link) — satisfied by
// app.StateService.
type BoardStateService interface {
	GetBoardState(ctx context.Context, boardID uuid.UUID) (*BoardState, error)
}

// BoardHandler serves the team-editor read route for the single board
// (CONTEXT.md invariant: the product always has exactly one board).
type BoardHandler struct {
	stateService BoardStateService
	boardID      uuid.UUID
}

// NewBoardHandler wires a BoardHandler against the given state service and
// the single board's id.
func NewBoardHandler(stateService BoardStateService, boardID uuid.UUID) *BoardHandler {
	return &BoardHandler{stateService: stateService, boardID: boardID}
}

// RegisterRoutes mounts the team-editor board-state route
// (contracts/openapi.yaml getBoard), relative to the caller's mount point
// (the server's shared /api/v1 registrar group).
func (h *BoardHandler) RegisterRoutes(r chi.Router) {
	r.Get("/board", h.handleGetBoard)
}

// @Summary  Get board state
// @Tags     board
// @Produce  json
// @Success  200 {object} BoardStateResponse
// @Router   /board [get]
func (h *BoardHandler) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	state, err := h.stateService.GetBoardState(r.Context(), h.boardID)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteJSON(w, toBoardStateResponse(state), http.StatusOK)
}

// BoardStateResponse is the wire shape for BoardState
// (contracts/openapi.yaml) — every column with its tasks, plus the board's
// current public link, or null (SCR-01/SCR-04).
type BoardStateResponse struct {
	Columns    []PublicColumnResponse `json:"columns"`
	PublicLink *PublicLinkResponse    `json:"public_link"`
}

// PublicLinkResponse mirrors the PublicLink schema (contracts/openapi.yaml).
type PublicLinkResponse struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

func toBoardStateResponse(state *BoardState) BoardStateResponse {
	resp := BoardStateResponse{
		Columns:    make([]PublicColumnResponse, 0, len(state.Columns)),
		PublicLink: toPublicLinkResponse(state.PublicLink),
	}
	for _, col := range state.Columns {
		resp.Columns = append(resp.Columns, toPublicColumnResponse(col))
	}
	return resp
}

func toPublicLinkResponse(link *domain.PublicLink) *PublicLinkResponse {
	if link == nil {
		return nil
	}
	return &PublicLinkResponse{Token: link.Token, CreatedAt: link.CreatedAt}
}
