package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// TaskListItem is a task as it appears in a board listing. Deliberately not
// domain.Task: the board state carries no description body (tasks spec §6 —
// a refetch on every SSE event would otherwise drag every description across
// the wire), only the derived flag and count the card draws its markers from.
type TaskListItem struct {
	ID             uuid.UUID
	ColumnID       uuid.UUID
	Title          string
	Assignee       *string
	Priority       domain.Priority
	DueDate        *time.Time
	HasDescription bool
	CommentCount   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TaskDetail is one task in full — description included — with its comments,
// oldest first (tasks TSK-01/TSK-08). Returned by both the team-editor and
// the token-scoped viewer read; the viewer's copy is identical, which is the
// point: the same content, minus every affordance to change it.
type TaskDetail struct {
	Task     domain.Task
	Comments []domain.Comment
}

// ColumnState is a column with its tasks attached, ordered by
// idx_columns_board_id_position / idx_tasks_column_id (data-model.md) — the
// shape GetBoardState returns for rendering the whole board in one call.
type ColumnState struct {
	domain.Column
	Tasks []TaskListItem
}

// BoardState bundles one board's identity (id/name — boards contract
// BoardState) with every column (and its tasks) plus the board's current
// public link, if any.
type BoardState struct {
	ID         uuid.UUID
	Name       string
	CreatedAt  time.Time
	Columns    []ColumnState
	PublicLink *domain.PublicLink
}

// BoardSummary is one dashboard row (boards BRD-01, contract BoardSummary):
// the board plus its total task count across all columns.
type BoardSummary struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	TaskCount int
}

// PublicBoardState is the read-only viewer shape returned by
// StateService.GetPublicBoardState (AC-09) — columns and tasks only,
// deliberately no PublicLink field, matching the PublicBoardState schema in
// contracts/openapi.yaml (T6: "must not expose team-editor-only fields").
// BoardID is internal routing context (which board's SSE bucket to join,
// boards BRD-05) — never serialized to the viewer.
type PublicBoardState struct {
	BoardID uuid.UUID
	Columns []ColumnState
}

// Repository is the persistence port app (T5/T6) depends on. infra's
// PostgresRepository is the sole implementation (sad.md §5).
type Repository interface {
	// ListBoards returns every board (oldest first) with its total task
	// count — the dashboard rows (boards BRD-01).
	ListBoards(ctx context.Context) ([]BoardSummary, error)

	// CreateBoard persists a new board together with its fixed columns in
	// one transaction (boards BRD-02) and fills their CreatedAt fields.
	CreateBoard(ctx context.Context, board *domain.Board, columns []domain.Column) error

	// GetBoardState returns the board's id/name plus every column (ordered
	// left-to-right) with its tasks, plus the board's current public link
	// (SCR-01, SCR-04). Returns domain.ErrBoardNotFound if boardID does not
	// exist (boards BRD-04).
	GetBoardState(ctx context.Context, boardID uuid.UUID) (*BoardState, error)

	// LeftmostColumnID resolves the position=0 column for boardID — where a
	// newly created task lands (AC-01). Returns domain.ErrBoardNotFound if
	// boardID has no columns (i.e. no such board).
	LeftmostColumnID(ctx context.Context, boardID uuid.UUID) (uuid.UUID, error)

	// InsertTask persists a new task. Returns domain.ErrColumnNotFound if
	// task.ColumnID does not exist.
	InsertTask(ctx context.Context, task *domain.Task) error

	// TaskByID returns one task in full (description included) together with
	// the id of the board it belongs to — the board id is what lets a
	// token-scoped caller check the task is actually on the board behind the
	// token (tasks TSK-13) without a second round trip. Returns
	// domain.ErrTaskNotFound if no such task exists.
	TaskByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, uuid.UUID, error)

	// ListComments returns a task's comments, oldest first (tasks TSK-08),
	// via idx_task_comments_task_id_created_at. An unknown task id yields an
	// empty slice, not an error — the caller has already resolved the task.
	ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error)

	// InsertComment persists a new comment and returns the board id its task
	// belongs to, for the board-scoped broadcast (boards BRD-05). Returns
	// domain.ErrTaskNotFound if comment.TaskID does not exist.
	InsertComment(ctx context.Context, comment *domain.Comment) (uuid.UUID, error)

	// DeleteComment hard-deletes a comment (tasks TSK-10) and returns the
	// board id its task belongs to. Returns domain.ErrCommentNotFound if no
	// such comment exists.
	DeleteComment(ctx context.Context, commentID uuid.UUID) (uuid.UUID, error)

	// UpdateTask persists edits to an existing task's details (title,
	// assignee, description, priority, due date) and fills task with the
	// stored row's remaining fields (column_id, created_at, updated_at), so
	// the caller holds a complete Task; it also
	// returns the task's board id, so the caller knows which board's SSE
	// bucket to notify (boards BRD-05). Returns domain.ErrTaskNotFound if no
	// such task exists.
	UpdateTask(ctx context.Context, task *domain.Task) (uuid.UUID, error)

	// MoveTask updates a task's column_id and updated_at — a single-row
	// UPDATE, no version/lock column (last-write-wins by design,
	// data-model.md) — and returns the complete moved task plus its board
	// id. Returns domain.ErrColumnNotFound if columnID does not exist
	// (AC-05), domain.ErrTaskNotFound if taskID does not exist.
	MoveTask(ctx context.Context, taskID, columnID uuid.UUID) (*domain.Task, uuid.UUID, error)

	// DeleteTask hard-deletes a task row (AC-06) and returns the board id
	// the task belonged to. Returns domain.ErrTaskNotFound if no such task
	// exists.
	DeleteTask(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)

	// IssuePublicLink persists a new public link for a board. Returns
	// domain.ErrLinkAlreadyActive if the board already has one (AC-07),
	// domain.ErrBoardNotFound if link.BoardID does not exist.
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
