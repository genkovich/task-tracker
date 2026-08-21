package ports

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// PublicStateService is the read-only, token-scoped port PublicHandler
// depends on (AC-09/AC-11) — satisfied by app.StateService.
type PublicStateService interface {
	GetPublicBoardState(ctx context.Context, token string) (*PublicBoardState, error)
}

// PublicHandler serves the read-only public-viewer routes (AC-09, AC-11).
// It deliberately registers no mutating route under /api/v1/public — AC-10
// is enforced structurally by there being no Post/Put/Delete call here.
type PublicHandler struct {
	stateService PublicStateService
}

// NewPublicHandler wires a PublicHandler against the given read-only state
// service.
func NewPublicHandler(stateService PublicStateService) *PublicHandler {
	return &PublicHandler{stateService: stateService}
}

// RegisterRoutes mounts the public, unauthenticated viewer routes
// (contracts/openapi.yaml getPublicBoard), relative to the caller's mount
// point (the server's shared /api/v1 registrar group).
func (h *PublicHandler) RegisterRoutes(r chi.Router) {
	r.Get("/public/{token}/board", h.handleGetPublicBoard)
}

// @Summary  Get public board state
// @Tags     public
// @Produce  json
// @Param    token path     string true "Public link token"
// @Success  200   {object} PublicBoardStateResponse
// @Failure  404   {object} httputil.ErrorResponse
// @Router   /public/{token}/board [get]
func (h *PublicHandler) handleGetPublicBoard(w http.ResponseWriter, r *http.Request) {
	setNoIndexHeader(w)
	// Never a long-lived cache — a viewer must always see the board's
	// current state, not a stale snapshot served by an intermediary after
	// the link was revoked or the board changed.
	w.Header().Set("Cache-Control", "no-store")
	token := chi.URLParam(r, "token")

	state, err := h.stateService.GetPublicBoardState(r.Context(), token)
	if err != nil {
		httputil.WriteError(w, mapPublicError(err))
		return
	}

	httputil.WriteJSON(w, toPublicBoardStateResponse(state), http.StatusOK)
}

// setNoIndexHeader marks a public (token-scoped) response as not indexable:
// the token is a capability URL and must never end up in a search index
// (spec §6.1 abuse case "витік public link у індекс пошуковика").
func setNoIndexHeader(w http.ResponseWriter) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
}

// mapPublicError maps domain errors surfaced on the public-viewer path.
// Deliberately its own mapping (not the shared board errorMap, T8): an
// unknown/revoked token reads as board.link_invalid here (AC-11,
// openapi.yaml PublicLinkInvalid) rather than the team-editor's
// board.link_not_found.
func mapPublicError(err error) error {
	if errors.Is(err, domain.ErrLinkNotFound) {
		return &apperr.Error{
			Code:       "board.link_invalid",
			Message:    "this link is no longer available",
			StatusCode: http.StatusNotFound,
		}
	}
	return err
}

// PublicBoardStateResponse is the wire shape for PublicBoardState
// (contracts/openapi.yaml) — columns and tasks only, deliberately no
// public_link field (AC-09).
type PublicBoardStateResponse struct {
	Columns []PublicColumnResponse `json:"columns"`
}

// PublicColumnResponse mirrors the Column schema (contracts/openapi.yaml).
type PublicColumnResponse struct {
	ID       uuid.UUID            `json:"id"`
	Name     string               `json:"name"`
	Position int16                `json:"position"`
	Tasks    []PublicTaskResponse `json:"tasks"`
}

// PublicTaskResponse mirrors the Task schema (contracts/openapi.yaml).
type PublicTaskResponse struct {
	ID        uuid.UUID `json:"id"`
	ColumnID  uuid.UUID `json:"column_id"`
	Title     string    `json:"title"`
	Assignee  *string   `json:"assignee"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toPublicBoardStateResponse(state *PublicBoardState) PublicBoardStateResponse {
	resp := PublicBoardStateResponse{Columns: make([]PublicColumnResponse, 0, len(state.Columns))}
	for _, col := range state.Columns {
		resp.Columns = append(resp.Columns, toPublicColumnResponse(col))
	}
	return resp
}

func toPublicColumnResponse(col ColumnState) PublicColumnResponse {
	resp := PublicColumnResponse{
		ID:       col.ID,
		Name:     col.Name,
		Position: col.Position,
		Tasks:    make([]PublicTaskResponse, 0, len(col.Tasks)),
	}
	for _, tk := range col.Tasks {
		resp.Tasks = append(resp.Tasks, PublicTaskResponse{
			ID:        tk.ID,
			ColumnID:  tk.ColumnID,
			Title:     tk.Title,
			Assignee:  tk.Assignee,
			CreatedAt: tk.CreatedAt,
			UpdatedAt: tk.UpdatedAt,
		})
	}
	return resp
}
