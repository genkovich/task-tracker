package ports

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// taskCreateRateLimit is the spec §6.1 abuse-case ceiling on task creation
// from a single client (AC-02 sibling case, contracts/openapi.yaml 429
// task.rate_limited).
const taskCreateRateLimit = 30

// TaskAppService is the task use-case port TaskHandler depends on
// (create/edit/move/delete, AC-01..AC-06) — satisfied by app.TaskService.
type TaskAppService interface {
	CreateTask(ctx context.Context, title string, assignee *string) (*domain.Task, error)
	EditTask(ctx context.Context, taskID uuid.UUID, title string, assignee *string) (*domain.Task, error)
	MoveTask(ctx context.Context, taskID, columnID uuid.UUID) error
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
}

// TaskHandler serves the team-editor task CRUD + move routes
// (contracts/openapi.yaml createTask/editTask/deleteTask/moveTask).
type TaskHandler struct {
	taskService TaskAppService
}

// NewTaskHandler wires a TaskHandler against the given task service.
func NewTaskHandler(taskService TaskAppService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// RegisterRoutes mounts the team-editor task routes. POST /api/v1/tasks is
// additionally rate-limited to taskCreateRateLimit requests/minute per
// client (spec §6.1 abuse case) — scoped to this one route via a chi
// sub-group so it doesn't throttle edit/move/delete.
func (h *TaskHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(httprate.Limit(
			taskCreateRateLimit,
			time.Minute,
			httprate.WithKeyFuncs(httprate.KeyByIP),
			httprate.WithLimitHandler(handleTaskRateLimited),
		))
		r.Post("/api/v1/tasks", h.handleCreateTask)
	})

	r.Patch("/api/v1/tasks/{taskId}", h.handleEditTask)
	r.Delete("/api/v1/tasks/{taskId}", h.handleDeleteTask)
	r.Post("/api/v1/tasks/{taskId}/move", h.handleMoveTask)
}

// handleTaskRateLimited writes the documented 429 task.rate_limited body
// (contracts/openapi.yaml) in place of httprate's default plaintext
// response.
func handleTaskRateLimited(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteError(w, &apperr.Error{
		Code:       "task.rate_limited",
		Message:    "too many tasks created, slow down",
		StatusCode: http.StatusTooManyRequests,
	})
}

// @Summary  Create task
// @Tags     board
// @Accept   json
// @Produce  json
// @Success  201 {object} TaskResponse
// @Failure  422 {object} httputil.ErrorResponse
// @Failure  429 {object} httputil.ErrorResponse
// @Router   /tasks [post]
func (h *TaskHandler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), req.Title, req.Assignee)
	if err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	httputil.WriteJSON(w, toTaskResponse(task), http.StatusCreated)
}

// @Summary  Edit task
// @Tags     board
// @Accept   json
// @Produce  json
// @Success  200 {object} TaskResponse
// @Failure  404 {object} httputil.ErrorResponse
// @Failure  422 {object} httputil.ErrorResponse
// @Router   /tasks/{taskId} [patch]
func (h *TaskHandler) handleEditTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_task_id", "invalid task id")
		return
	}

	var req TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	task, err := h.taskService.EditTask(r.Context(), taskID, req.Title, req.Assignee)
	if err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	httputil.WriteJSON(w, toTaskResponse(task), http.StatusOK)
}

// @Summary  Delete task
// @Tags     board
// @Success  204
// @Failure  404 {object} httputil.ErrorResponse
// @Router   /tasks/{taskId} [delete]
func (h *TaskHandler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_task_id", "invalid task id")
		return
	}

	if err := h.taskService.DeleteTask(r.Context(), taskID); err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary  Move task
// @Tags     board
// @Accept   json
// @Produce  json
// @Success  200 {object} TaskResponse
// @Failure  404 {object} httputil.ErrorResponse
// @Failure  422 {object} httputil.ErrorResponse
// @Router   /tasks/{taskId}/move [post]
func (h *TaskHandler) handleMoveTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_task_id", "invalid task id")
		return
	}

	var req TaskMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	columnID, err := uuid.Parse(req.ColumnID)
	if err != nil {
		httputil.WriteError(w, mapTaskError(domain.ErrColumnNotFound))
		return
	}

	if err := h.taskService.MoveTask(r.Context(), taskID, columnID); err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	// TaskAppService.MoveTask (T5) reports success/failure only — it does not
	// return the moved task, so title/assignee/timestamps aren't available
	// here to echo back. id/column_id are known-good (the move succeeded
	// against exactly this pair).
	httputil.WriteJSON(w, TaskResponse{ID: taskID.String(), ColumnID: columnID.String()}, http.StatusOK)
}

func toTaskResponse(task *domain.Task) TaskResponse {
	return TaskResponse{
		ID:        task.ID.String(),
		ColumnID:  task.ColumnID.String(),
		Title:     task.Title,
		Assignee:  task.Assignee,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
}
