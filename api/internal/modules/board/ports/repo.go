package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// ColumnState is a column with its tasks attached, ordered by
// idx_columns_board_id_position / idx_tasks_column_id (data-model.md) — the
// shape GetBoardState returns for rendering the whole board in one call.
type ColumnState struct {
	domain.Column
	Tasks []domain.Task
}

// BoardState bundles every column (with its tasks) plus the board's current
// public link, if any — matches the BoardState schema in
// contracts/openapi.yaml.
type BoardState struct {
	Columns    []ColumnState
	PublicLink *domain.PublicLink
}

// PublicBoardState is the read-only viewer shape returned by
// StateService.GetPublicBoardState (AC-09) — columns and tasks only,
// deliberately no PublicLink field, matching the PublicBoardState schema in
// contracts/openapi.yaml (T6: "must not expose team-editor-only fields").
type PublicBoardState struct {
	Columns []ColumnState
}

// Repository is the persistence port app (T5/T6) depends on. infra's
// PostgresRepository is the sole implementation (sad.md §5).
type Repository interface {
	// GetBoardState returns every column (ordered left-to-right) with its
	// tasks, plus the board's current public link (SCR-01, SCR-04).
	GetBoardState(ctx context.Context, boardID uuid.UUID) (*BoardState, error)

	// LeftmostColumnID resolves the position=0 column for boardID — where a
	// newly created task lands (AC-01).
	LeftmostColumnID(ctx context.Context, boardID uuid.UUID) (uuid.UUID, error)

	// InsertTask persists a new task. Returns domain.ErrColumnNotFound if
	// task.ColumnID does not exist.
	InsertTask(ctx context.Context, task *domain.Task) error

	// UpdateTask persists edits to an existing task's title/assignee.
	// Returns domain.ErrTaskNotFound if no such task exists.
	UpdateTask(ctx context.Context, task *domain.Task) error

	// MoveTask updates a task's column_id — a plain single-row UPDATE, no
	// version/lock column (last-write-wins by design, data-model.md).
	// Returns domain.ErrColumnNotFound if columnID does not exist (AC-05),
	// domain.ErrTaskNotFound if taskID does not exist.
	MoveTask(ctx context.Context, taskID, columnID uuid.UUID) error

	// DeleteTask hard-deletes a task row (AC-06). Returns
	// domain.ErrTaskNotFound if no such task exists.
	DeleteTask(ctx context.Context, taskID uuid.UUID) error

	// ColumnExists reports whether columnID exists.
	ColumnExists(ctx context.Context, columnID uuid.UUID) (bool, error)

	// IssuePublicLink persists a new public link for a board. Returns
	// domain.ErrLinkAlreadyActive if the board already has one (AC-07).
	IssuePublicLink(ctx context.Context, link *domain.PublicLink) error

	// RevokePublicLink hard-deletes the board's active public link (AC-08).
	// Returns domain.ErrLinkNotFound if there is none.
	RevokePublicLink(ctx context.Context, boardID uuid.UUID) error

	// PublicLinkByToken looks up a public link by its opaque token
	// (AC-09/AC-11). Returns domain.ErrLinkNotFound if token is unknown.
	PublicLinkByToken(ctx context.Context, token string) (*domain.PublicLink, error)

	// PublicLinkByBoard looks up a board's active public link, if any
	// (AC-08: RevokePublicLink needs the token to close before deleting the
	// row). Returns domain.ErrLinkNotFound if the board has no active link.
	PublicLinkByBoard(ctx context.Context, boardID uuid.UUID) (*domain.PublicLink, error)
}
