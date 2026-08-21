package ports

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// LinkAppService is the public-link issue/revoke port LinkHandler depends on
// (AC-07/AC-08) — satisfied by app.LinkService.
type LinkAppService interface {
	IssuePublicLink(ctx context.Context, boardID uuid.UUID) (*domain.PublicLink, error)
	RevokePublicLink(ctx context.Context, boardID uuid.UUID) error
}

// LinkHandler serves the team-editor per-board public-link issue/revoke
// routes (boards BRD-06).
type LinkHandler struct {
	linkService LinkAppService
}

// NewLinkHandler wires a LinkHandler against the given link service.
func NewLinkHandler(linkService LinkAppService) *LinkHandler {
	return &LinkHandler{linkService: linkService}
}

// RegisterRoutes mounts the team-editor public-link routes (boards contract
// issuePublicLink/revokePublicLink), relative to the caller's mount point
// (the server's shared /api/v1 registrar group).
func (h *LinkHandler) RegisterRoutes(r chi.Router) {
	r.Post("/boards/{boardId}/public-link", h.handleIssuePublicLink)
	r.Delete("/boards/{boardId}/public-link", h.handleRevokePublicLink)
}

// @Summary  Issue public link
// @Tags     board
// @Produce  json
// @Param    boardId path string true "Board id"
// @Success  201 {object} PublicLinkResponse
// @Failure  404 {object} httputil.ErrorResponse
// @Failure  409 {object} httputil.ErrorResponse
// @Router   /boards/{boardId}/public-link [post]
func (h *LinkHandler) handleIssuePublicLink(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseBoardID(w, r)
	if !ok {
		return
	}

	link, err := h.linkService.IssuePublicLink(r.Context(), boardID)
	if err != nil {
		httputil.WriteError(w, mapLinkError(err))
		return
	}

	httputil.WriteJSON(w, PublicLinkResponse{Token: link.Token, CreatedAt: link.CreatedAt}, http.StatusCreated)
}

// @Summary  Revoke public link
// @Tags     board
// @Param    boardId path string true "Board id"
// @Success  204
// @Failure  404 {object} httputil.ErrorResponse
// @Router   /boards/{boardId}/public-link [delete]
func (h *LinkHandler) handleRevokePublicLink(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseBoardID(w, r)
	if !ok {
		return
	}

	if err := h.linkService.RevokePublicLink(r.Context(), boardID); err != nil {
		httputil.WriteError(w, mapLinkError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// mapLinkError maps the public-link domain sentinels surfaced on the
// team-editor path (contracts/openapi.yaml).
func mapLinkError(err error) error {
	switch {
	case errors.Is(err, domain.ErrBoardNotFound):
		return mapBoardError(err)
	case errors.Is(err, domain.ErrLinkAlreadyActive):
		return &apperr.Error{
			Code:       "board.link_already_active",
			Message:    "an active public link already exists — revoke it first",
			StatusCode: http.StatusConflict,
		}
	case errors.Is(err, domain.ErrLinkNotFound):
		return &apperr.Error{
			Code:       "board.link_not_found",
			Message:    "no active public link to revoke",
			StatusCode: http.StatusNotFound,
		}
	default:
		return err
	}
}
