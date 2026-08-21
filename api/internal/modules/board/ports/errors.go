package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

// mapTaskError maps the task domain sentinels (T5's TaskService) surfaced on
// the team-editor task routes (contracts/openapi.yaml createTask/editTask/
// deleteTask/moveTask) to their documented wire codes/status.
func mapTaskError(err error) error {
	switch {
	case errors.Is(err, domain.ErrTitleRequired):
		return &apperr.Error{
			Code:       "task.title_required",
			Message:    "task title is required",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrTitleTooLong):
		return &apperr.Error{
			Code:       "task.title_too_long",
			Message:    "task title must be at most 200 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrAssigneeTooLong):
		return &apperr.Error{
			Code:       "task.assignee_too_long",
			Message:    "task assignee must be at most 200 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrTaskNotFound):
		return &apperr.Error{
			Code:       "task.not_found",
			Message:    "task not found",
			StatusCode: http.StatusNotFound,
		}
	case errors.Is(err, domain.ErrColumnNotFound):
		return &apperr.Error{
			Code:       "board.column_not_found",
			Message:    "target column does not exist",
			StatusCode: http.StatusUnprocessableEntity,
		}
	default:
		return err
	}
}
