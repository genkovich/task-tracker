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
	CreateTask(ctx context.Context, boardID uuid.UUID, details domain.TaskDetails) (*domain.Task, error)
	EditTask(ctx context.Context, taskID uuid.UUID, details domain.TaskDetails) (*domain.Task, error)
	MoveTask(ctx context.Context, taskID, columnID uuid.UUID) (*domain.Task, error)
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
}

// TaskDetailService is the task-detail read port TaskHandler depends on
// (tasks TSK-01/TSK-08) — satisfied by app.StateService.
type TaskDetailService interface {
	GetTaskDetail(ctx context.Context, taskID uuid.UUID) (*TaskDetail, error)
}

// TaskHandler serves the team-editor task CRUD + move + detail routes
// (contracts/openapi.yaml createTask/editTask/deleteTask/moveTask/getTask).
type TaskHandler struct {
	taskService   TaskAppService
	detailService TaskDetailService
}

// NewTaskHandler wires a TaskHandler against the given task and detail
// services.
func NewTaskHandler(taskService TaskAppService, detailService TaskDetailService) *TaskHandler {
	return &TaskHandler{taskService: taskService, detailService: detailService}
}

// RegisterRoutes mounts the team-editor task routes, relative to the
// caller's mount point (the server's shared /api/v1 registrar group).
// POST /tasks is additionally rate-limited to taskCreateRateLimit
// requests/minute per client (spec §6.1 abuse case) — scoped to this one
// route via a chi sub-group so it doesn't throttle edit/move/delete.
func (h *TaskHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(httprate.Limit(
			taskCreateRateLimit,
			time.Minute,
			// XFF-aware key: behind Caddy every client shares RemoteAddr,
			// so keying by it would give the whole team one bucket.
			httprate.WithKeyFuncs(httputil.ClientIPKey),
			httprate.WithLimitHandler(handleTaskRateLimited),
		))
		r.Post("/tasks", h.handleCreateTask)
	})

	r.Get("/tasks/{taskId}", h.handleGetTask)
	r.Patch("/tasks/{taskId}", h.handleEditTask)
	r.Delete("/tasks/{taskId}", h.handleDeleteTask)
	r.Post("/tasks/{taskId}/move", h.handleMoveTask)
}

// parseTaskID reads the {taskId} path param; on a non-UUID value it writes
// the documented 400 validation.invalid_task_id and reports !ok.
func parseTaskID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_task_id", "invalid task id")
		return uuid.Nil, false
	}
	return taskID, true
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
// @Failure  404 {object} httputil.ErrorResponse
// @Failure  422 {object} httputil.ErrorResponse
// @Failure  429 {object} httputil.ErrorResponse
// @Router   /tasks [post]
func (h *TaskHandler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	boardID, err := uuid.Parse(req.BoardID)
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_board_id", "invalid board id")
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_due_date", "due date must be a calendar day (YYYY-MM-DD)")
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), boardID, domain.TaskDetails{
		Title:       req.Title,
		Assignee:    req.Assignee,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     dueDate,
	})
	if err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	httputil.WriteJSON(w, toTaskResponse(task), http.StatusCreated)
}

// @Summary  Get task detail
// @Tags     board
// @Produce  json
// @Param    taskId path     string true "Task id"
// @Success  200    {object} TaskDetailResponse
// @Failure  404    {object} httputil.ErrorResponse
// @Router   /tasks/{taskId} [get]
func (h *TaskHandler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	detail, err := h.detailService.GetTaskDetail(r.Context(), taskID)
	if err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	httputil.WriteJSON(w, toTaskDetailResponse(detail), http.StatusOK)
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
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	var req TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_due_date", "due date must be a calendar day (YYYY-MM-DD)")
		return
	}

	task, err := h.taskService.EditTask(r.Context(), taskID, domain.TaskDetails{
		Title:       req.Title,
		Assignee:    req.Assignee,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     dueDate,
	})
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
	taskID, ok := parseTaskID(w, r)
	if !ok {
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
	taskID, ok := parseTaskID(w, r)
	if !ok {
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

	task, err := h.taskService.MoveTask(r.Context(), taskID, columnID)
	if err != nil {
		httputil.WriteError(w, mapTaskError(err))
		return
	}

	httputil.WriteJSON(w, toTaskResponse(task), http.StatusOK)
}
