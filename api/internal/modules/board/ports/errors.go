package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

// mapBoardError maps the board domain sentinels surfaced on the dashboard
// and per-board routes (boards contract createBoard/getBoard) to their
// documented wire codes/status.
func mapBoardError(err error) error {
	switch {
	case errors.Is(err, domain.ErrBoardNameRequired):
		return &apperr.Error{
			Code:       "board.name_required",
			Message:    "board name is required",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrBoardNameTooLong):
		return &apperr.Error{
			Code:       "board.name_too_long",
			Message:    "board name must be at most 200 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrBoardNotFound):
		return &apperr.Error{
			Code:       "board.not_found",
			Message:    "board not found",
			StatusCode: http.StatusNotFound,
		}
	default:
		return err
	}
}

// mapTaskError maps the task domain sentinels (T5's TaskService) surfaced on
// the team-editor task routes (contracts/openapi.yaml createTask/editTask/
// deleteTask/moveTask) to their documented wire codes/status.
func mapTaskError(err error) error {
	switch {
	case errors.Is(err, domain.ErrBoardNotFound):
		return mapBoardError(err)
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
	case errors.Is(err, domain.ErrDescriptionTooLong):
		return &apperr.Error{
			Code:       "task.description_too_long",
			Message:    "task description must be at most 4000 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrPriorityInvalid):
		return &apperr.Error{
			Code:       "task.priority_invalid",
			Message:    "task priority must be one of low, medium, high",
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

// mapCommentError maps the comment domain sentinels (T5's CommentService)
// surfaced on the team-editor comment routes (tasks contract
// listTaskComments/addTaskComment/deleteTaskComment) to their documented wire
// codes/status. A comment on a task that no longer exists reads as
// task.not_found — the caller's mistake is the task id, not the comment.
func mapCommentError(err error) error {
	switch {
	case errors.Is(err, domain.ErrTaskNotFound):
		return mapTaskError(err)
	case errors.Is(err, domain.ErrCommentAuthorRequired):
		return &apperr.Error{
			Code:       "comment.author_required",
			Message:    "comment author is required",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrCommentAuthorTooLong):
		return &apperr.Error{
			Code:       "comment.author_too_long",
			Message:    "comment author must be at most 200 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrCommentBodyRequired):
		return &apperr.Error{
			Code:       "comment.body_required",
			Message:    "comment body is required",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrCommentBodyTooLong):
		return &apperr.Error{
			Code:       "comment.body_too_long",
			Message:    "comment body must be at most 2000 characters",
			StatusCode: http.StatusUnprocessableEntity,
		}
	case errors.Is(err, domain.ErrCommentNotFound):
		return &apperr.Error{
			Code:       "comment.not_found",
			Message:    "comment not found",
			StatusCode: http.StatusNotFound,
		}
	default:
		return err
	}
}
