package ports

import (
	"context"
	"encoding/json"
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

// BoardAppService is the dashboard port BoardHandler depends on (boards
// BRD-01/BRD-02) — satisfied by app.BoardService.
type BoardAppService interface {
	ListBoards(ctx context.Context) ([]BoardSummary, error)
	CreateBoard(ctx context.Context, name string) (*BoardState, error)
}

// BoardHandler serves the team-editor board routes: the dashboard list and
// create (boards BRD-01/BRD-02) and the per-board state read (BRD-04).
type BoardHandler struct {
	boardService BoardAppService
	stateService BoardStateService
}

// NewBoardHandler wires a BoardHandler against the given board and state
// services.
func NewBoardHandler(boardService BoardAppService, stateService BoardStateService) *BoardHandler {
	return &BoardHandler{boardService: boardService, stateService: stateService}
}

// RegisterRoutes mounts the board routes (boards contract listBoards/
// createBoard/getBoard), relative to the caller's mount point (the server's
// shared /api/v1 registrar group).
func (h *BoardHandler) RegisterRoutes(r chi.Router) {
	r.Get("/boards", h.handleListBoards)
	r.Post("/boards", h.handleCreateBoard)
	r.Get("/boards/{boardId}", h.handleGetBoard)
}

// @Summary  List boards
// @Tags     board
// @Produce  json
// @Success  200 {array} BoardSummaryResponse
// @Router   /boards [get]
func (h *BoardHandler) handleListBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := h.boardService.ListBoards(r.Context())
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	resp := make([]BoardSummaryResponse, 0, len(boards))
	for _, b := range boards {
		resp = append(resp, BoardSummaryResponse(b))
	}
	httputil.WriteJSON(w, resp, http.StatusOK)
}

// @Summary  Create board
// @Tags     board
// @Accept   json
// @Produce  json
// @Success  201 {object} BoardStateResponse
// @Failure  422 {object} httputil.ErrorResponse
// @Router   /boards [post]
func (h *BoardHandler) handleCreateBoard(w http.ResponseWriter, r *http.Request) {
	var req BoardCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	state, err := h.boardService.CreateBoard(r.Context(), req.Name)
	if err != nil {
		httputil.WriteError(w, mapBoardError(err))
		return
	}

	httputil.WriteJSON(w, toBoardStateResponse(state), http.StatusCreated)
}

// @Summary  Get board state
// @Tags     board
// @Produce  json
// @Param    boardId path string true "Board id"
// @Success  200 {object} BoardStateResponse
// @Failure  404 {object} httputil.ErrorResponse
// @Router   /boards/{boardId} [get]
func (h *BoardHandler) handleGetBoard(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseBoardID(w, r)
	if !ok {
		return
	}

	state, err := h.stateService.GetBoardState(r.Context(), boardID)
	if err != nil {
		httputil.WriteError(w, mapBoardError(err))
		return
	}

	httputil.WriteJSON(w, toBoardStateResponse(state), http.StatusOK)
}

// parseBoardID reads the {boardId} path param; on a non-UUID value it writes
// the documented 400 validation.invalid_board_id and reports !ok.
func parseBoardID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	boardID, err := uuid.Parse(chi.URLParam(r, "boardId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_board_id", "invalid board id")
		return uuid.Nil, false
	}
	return boardID, true
}

// BoardStateResponse is the wire shape for BoardState (boards contract) —
// the board's identity plus every column with its tasks, plus the board's
// current public link, or null (SCR-01/SCR-04).
type BoardStateResponse struct {
	ID         uuid.UUID              `json:"id"`
	Name       string                 `json:"name"`
	CreatedAt  time.Time              `json:"created_at"`
	Columns    []PublicColumnResponse `json:"columns"`
	PublicLink *PublicLinkResponse    `json:"public_link"`
}

// BoardSummaryResponse mirrors the BoardSummary schema (boards contract) —
// one dashboard row.
type BoardSummaryResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	TaskCount int       `json:"task_count"`
}

// PublicLinkResponse mirrors the PublicLink schema (contracts/openapi.yaml).
type PublicLinkResponse struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

func toBoardStateResponse(state *BoardState) BoardStateResponse {
	resp := BoardStateResponse{
		ID:         state.ID,
		Name:       state.Name,
		CreatedAt:  state.CreatedAt,
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
