package ports

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// CommentAppService is the comment use-case port CommentHandler depends on
// (tasks TSK-08/TSK-10) — satisfied by app.CommentService.
type CommentAppService interface {
	ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error)
	AddComment(ctx context.Context, taskID uuid.UUID, author, body string) (*domain.Comment, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
}

// CommentHandler serves the team-editor comment routes (tasks contract
// listTaskComments/addTaskComment/deleteTaskComment). Its own handler rather
// than three more routes on TaskHandler, which already carries task CRUD, the
// move route and the create rate limit.
//
// These are editor routes by construction: there is no comment route under
// /api/v1/public/{token}/..., which is what keeps a viewer read-only
// (TSK-12) — a routing fact, not a runtime check.
type CommentHandler struct {
	commentService CommentAppService
}

// NewCommentHandler wires a CommentHandler against the given comment service.
func NewCommentHandler(commentService CommentAppService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// RegisterRoutes mounts the comment routes relative to the caller's mount
// point (the server's shared /api/v1 registrar group).
func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tasks/{taskId}/comments", h.handleListComments)
	r.Post("/tasks/{taskId}/comments", h.handleAddComment)
	r.Delete("/tasks/{taskId}/comments/{commentId}", h.handleDeleteComment)
}

// @Summary  List task comments
// @Tags     board
// @Produce  json
// @Param    taskId path     string true "Task id"
// @Success  200    {array}  CommentResponse
// @Failure  404    {object} httputil.ErrorResponse
// @Router   /tasks/{taskId}/comments [get]
func (h *CommentHandler) handleListComments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	comments, err := h.commentService.ListComments(r.Context(), taskID)
	if err != nil {
		httputil.WriteError(w, mapCommentError(err))
		return
	}

	httputil.WriteJSON(w, toCommentResponses(comments), http.StatusOK)
}

// @Summary  Add task comment
// @Tags     board
// @Accept   json
// @Produce  json
// @Param    taskId path     string true "Task id"
// @Success  201    {object} CommentResponse
// @Failure  404    {object} httputil.ErrorResponse
// @Failure  422    {object} httputil.ErrorResponse
// @Router   /tasks/{taskId}/comments [post]
func (h *CommentHandler) handleAddComment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	var req CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	comment, err := h.commentService.AddComment(r.Context(), taskID, req.Author, req.Body)
	if err != nil {
		httputil.WriteError(w, mapCommentError(err))
		return
	}

	httputil.WriteJSON(w, toCommentResponse(*comment), http.StatusCreated)
}

// @Summary  Delete task comment
// @Tags     board
// @Param    taskId    path string true "Task id"
// @Param    commentId path string true "Comment id"
// @Success  204
// @Failure  404 {object} httputil.ErrorResponse
// @Router   /tasks/{taskId}/comments/{commentId} [delete]
func (h *CommentHandler) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	if _, ok := parseTaskID(w, r); !ok {
		return
	}

	commentID, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_comment_id", "invalid comment id")
		return
	}

	if err := h.commentService.DeleteComment(r.Context(), commentID); err != nil {
		httputil.WriteError(w, mapCommentError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
