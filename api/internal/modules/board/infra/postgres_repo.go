package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

// PostgresRepository implements ports.Repository against pgx/v5, using the
// indexes fixed by data-model.md (idx_columns_board_id_position,
// idx_tasks_column_id, the two public_links UNIQUEs).
type PostgresRepository struct {
	db *database.DB
}

// NewPostgresRepository wires a PostgresRepository against db.
func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

var _ ports.Repository = (*PostgresRepository)(nil)

// ListBoards returns every board (oldest first) with its total task count —
// the dashboard rows (boards BRD-01).
func (r *PostgresRepository) ListBoards(ctx context.Context) ([]ports.BoardSummary, error) {
	rows, err := r.db.Query(ctx,
		`SELECT b.id, b.name, b.created_at, COUNT(t.id)
		 FROM boards b
		 LEFT JOIN columns c ON c.board_id = b.id
		 LEFT JOIN tasks t ON t.column_id = c.id
		 GROUP BY b.id, b.name, b.created_at
		 ORDER BY b.created_at ASC, b.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list boards: %w", err)
	}
	defer rows.Close()

	boards := []ports.BoardSummary{}
	for rows.Next() {
		var b ports.BoardSummary
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt, &b.TaskCount); err != nil {
			return nil, fmt.Errorf("list boards: %w", err)
		}
		boards = append(boards, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list boards: %w", err)
	}
	return boards, nil
}

// CreateBoard persists a new board together with its fixed columns in one
// transaction (boards BRD-02) — either the board and all its columns exist,
// or nothing does.
func (r *PostgresRepository) CreateBoard(ctx context.Context, board *domain.Board, columns []domain.Column) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create board: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO boards (id, name) VALUES ($1, $2) RETURNING created_at`,
		board.ID, board.Name,
	).Scan(&board.CreatedAt)
	if err != nil {
		return fmt.Errorf("create board: insert board: %w", err)
	}

	for i := range columns {
		err = tx.QueryRow(ctx,
			`INSERT INTO columns (id, board_id, name, position)
			 VALUES ($1, $2, $3, $4)
			 RETURNING created_at`,
			columns[i].ID, columns[i].BoardID, columns[i].Name, columns[i].Position,
		).Scan(&columns[i].CreatedAt)
		if err != nil {
			return fmt.Errorf("create board: insert column %q: %w", columns[i].Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("create board: commit: %w", err)
	}
	return nil
}

// GetBoardState returns the board's id/name plus every column (ordered
// left-to-right) with its tasks, plus the board's current public link, if
// any. domain.ErrBoardNotFound for an unknown boardID (boards BRD-04).
func (r *PostgresRepository) GetBoardState(ctx context.Context, boardID uuid.UUID) (*ports.BoardState, error) {
	var board domain.Board
	err := r.db.QueryRow(ctx,
		`SELECT id, name, created_at FROM boards WHERE id = $1`, boardID,
	).Scan(&board.ID, &board.Name, &board.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBoardNotFound
		}
		return nil, fmt.Errorf("get board state: board row: %w", err)
	}

	columns, err := r.columnsForBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("get board state: list columns: %w", err)
	}

	for i := range columns {
		tasks, err := r.tasksForColumn(ctx, columns[i].ID)
		if err != nil {
			return nil, fmt.Errorf("get board state: list tasks: %w", err)
		}
		columns[i].Tasks = tasks
	}

	link, err := r.publicLinkForBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("get board state: public link: %w", err)
	}

	return &ports.BoardState{
		ID:         board.ID,
		Name:       board.Name,
		CreatedAt:  board.CreatedAt,
		Columns:    columns,
		PublicLink: link,
	}, nil
}

func (r *PostgresRepository) columnsForBoard(ctx context.Context, boardID uuid.UUID) ([]ports.ColumnState, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, board_id, name, position, created_at
		 FROM columns WHERE board_id = $1 ORDER BY position ASC`, boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []ports.ColumnState{}
	for rows.Next() {
		var c ports.ColumnState
		if err := rows.Scan(&c.ID, &c.BoardID, &c.Name, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		columns = append(columns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

// tasksForColumn reads the card-level view of a column's tasks. The
// description body is deliberately not selected — only whether there is one
// (tasks spec §6) — and the comment count comes from one aggregate over the
// whole column rather than a per-task query, which on a 100-task board would
// mean 100 extra round trips per render (data-model.md §Access patterns).
func (r *PostgresRepository) tasksForColumn(ctx context.Context, columnID uuid.UUID) ([]ports.TaskListItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.column_id, t.title, t.assignee, t.priority, t.due_date,
		        t.description <> '' AS has_description, COUNT(c.id) AS comment_count,
		        t.created_at, t.updated_at
		 FROM tasks t
		 LEFT JOIN task_comments c ON c.task_id = t.id
		 WHERE t.column_id = $1
		 GROUP BY t.id
		 ORDER BY t.created_at ASC`, columnID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []ports.TaskListItem{}
	for rows.Next() {
		var (
			t        ports.TaskListItem
			priority string
		)
		if err := rows.Scan(&t.ID, &t.ColumnID, &t.Title, &t.Assignee, &priority, &t.DueDate,
			&t.HasDescription, &t.CommentCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Priority = domain.Priority(priority)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// TaskByID returns one task in full plus the board it belongs to — the board
// id is resolved in the same read, so a token-scoped caller never has a gap
// between "which task is this" and "whose board is it" (tasks TSK-13).
func (r *PostgresRepository) TaskByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, uuid.UUID, error) {
	var (
		task     domain.Task
		priority string
		boardID  uuid.UUID
	)
	err := r.db.QueryRow(ctx,
		`SELECT t.id, t.column_id, t.title, t.assignee, t.description, t.priority, t.due_date,
		        t.created_at, t.updated_at, c.board_id
		 FROM tasks t
		 JOIN columns c ON c.id = t.column_id
		 WHERE t.id = $1`, taskID,
	).Scan(&task.ID, &task.ColumnID, &task.Title, &task.Assignee, &task.Description, &priority,
		&task.DueDate, &task.CreatedAt, &task.UpdatedAt, &boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, domain.ErrTaskNotFound
		}
		return nil, uuid.Nil, fmt.Errorf("task by id: %w", err)
	}
	task.Priority = domain.Priority(priority)
	return &task, boardID, nil
}

// ListComments returns a task's comments oldest first (tasks TSK-08), served
// by idx_task_comments_task_id_created_at.
func (r *PostgresRepository) ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, task_id, author, body, created_at
		 FROM task_comments WHERE task_id = $1 ORDER BY created_at ASC, id ASC`, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return comments, nil
}

// InsertComment persists a comment and returns the board its task sits on,
// for the board-scoped broadcast (BRD-05). An FK violation means the task is
// gone → domain.ErrTaskNotFound, never a raw 500 (test-plan.md edge case).
func (r *PostgresRepository) InsertComment(ctx context.Context, comment *domain.Comment) (uuid.UUID, error) {
	var boardID uuid.UUID
	err := r.db.QueryRow(ctx,
		`INSERT INTO task_comments (id, task_id, author, body)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at,
		           (SELECT c.board_id FROM columns c
		            JOIN tasks t ON t.column_id = c.id
		            WHERE t.id = task_comments.task_id)`,
		comment.ID, comment.TaskID, comment.Author, comment.Body,
	).Scan(&comment.CreatedAt, &boardID)
	if err != nil {
		if database.IsPgForeignKeyViolation(err) {
			return uuid.Nil, domain.ErrTaskNotFound
		}
		return uuid.Nil, fmt.Errorf("insert comment: %w", err)
	}
	return boardID, nil
}

// DeleteComment hard-deletes a comment belonging to taskID (tasks TSK-10) and
// returns the board its task sits on, for the board-scoped broadcast. task_id
// is part of the WHERE clause, not decoration: the route names both ids, and
// keying on the comment alone would let any comment be deleted through any
// task's path.
func (r *PostgresRepository) DeleteComment(ctx context.Context, taskID, commentID uuid.UUID) (uuid.UUID, error) {
	var boardID uuid.UUID
	err := r.db.QueryRow(ctx,
		`DELETE FROM task_comments WHERE id = $1 AND task_id = $2
		 RETURNING (SELECT c.board_id FROM columns c
		            JOIN tasks t ON t.column_id = c.id
		            WHERE t.id = task_comments.task_id)`,
		commentID, taskID,
	).Scan(&boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrCommentNotFound
		}
		return uuid.Nil, fmt.Errorf("delete comment: %w", err)
	}
	return boardID, nil
}

func (r *PostgresRepository) publicLinkForBoard(ctx context.Context, boardID uuid.UUID) (*domain.PublicLink, error) {
	var link domain.PublicLink
	err := r.db.QueryRow(ctx,
		`SELECT id, board_id, token, created_at FROM public_links WHERE board_id = $1`, boardID,
	).Scan(&link.ID, &link.BoardID, &link.Token, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &link, nil
}

// LeftmostColumnID resolves the position=0 column for boardID (AC-01).
// No rows means no such board (every board always carries its three fixed
// columns — CONTEXT.md invariant), hence domain.ErrBoardNotFound.
func (r *PostgresRepository) LeftmostColumnID(ctx context.Context, boardID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM columns WHERE board_id = $1 ORDER BY position ASC LIMIT 1`, boardID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrBoardNotFound
		}
		return uuid.Nil, fmt.Errorf("leftmost column: %w", err)
	}
	return id, nil
}

// InsertTask persists a new task.
func (r *PostgresRepository) InsertTask(ctx context.Context, task *domain.Task) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO tasks (id, column_id, title, assignee, description, priority, due_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		task.ID, task.ColumnID, task.Title, task.Assignee, task.Description, string(task.Priority), task.DueDate,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if database.IsPgForeignKeyViolation(err) {
			return domain.ErrColumnNotFound
		}
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// UpdateTask persists edits to an existing task's title/assignee, fills
// task with the row's remaining columns, so callers hand back a complete
// Task (contracts/openapi.yaml Task requires column_id/created_at too), and
// returns the task's board id for the board-scoped broadcast (BRD-05).
func (r *PostgresRepository) UpdateTask(ctx context.Context, task *domain.Task) (uuid.UUID, error) {
	var boardID uuid.UUID
	err := r.db.QueryRow(ctx,
		`UPDATE tasks
		 SET title = $1, assignee = $2, description = $3, priority = $4, due_date = $5, updated_at = now()
		 WHERE id = $6
		 RETURNING column_id, created_at, updated_at,
		           (SELECT board_id FROM columns WHERE columns.id = tasks.column_id)`,
		task.Title, task.Assignee, task.Description, string(task.Priority), task.DueDate, task.ID,
	).Scan(&task.ColumnID, &task.CreatedAt, &task.UpdatedAt, &boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrTaskNotFound
		}
		return uuid.Nil, fmt.Errorf("update task: %w", err)
	}
	return boardID, nil
}

// MoveTask updates a task's column_id and updated_at — a single-row UPDATE,
// no version/lock column (last-write-wins by design, data-model.md) — and
// returns the complete moved row plus its (new) board id.
func (r *PostgresRepository) MoveTask(ctx context.Context, taskID, columnID uuid.UUID) (*domain.Task, uuid.UUID, error) {
	task := domain.Task{ID: taskID}
	var (
		priority string
		boardID  uuid.UUID
	)
	err := r.db.QueryRow(ctx,
		`UPDATE tasks SET column_id = $1, updated_at = now()
		 WHERE id = $2
		 RETURNING column_id, title, assignee, description, priority, due_date, created_at, updated_at,
		           (SELECT board_id FROM columns WHERE columns.id = tasks.column_id)`,
		columnID, taskID,
	).Scan(&task.ColumnID, &task.Title, &task.Assignee, &task.Description, &priority, &task.DueDate,
		&task.CreatedAt, &task.UpdatedAt, &boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, domain.ErrTaskNotFound
		}
		if database.IsPgForeignKeyViolation(err) {
			return nil, uuid.Nil, domain.ErrColumnNotFound
		}
		return nil, uuid.Nil, fmt.Errorf("move task: %w", err)
	}
	task.Priority = domain.Priority(priority)
	return &task, boardID, nil
}

// DeleteTask hard-deletes a task row (AC-06) and returns the board id the
// task belonged to, for the board-scoped broadcast (BRD-05).
func (r *PostgresRepository) DeleteTask(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error) {
	var boardID uuid.UUID
	err := r.db.QueryRow(ctx,
		`DELETE FROM tasks WHERE id = $1
		 RETURNING (SELECT board_id FROM columns WHERE columns.id = tasks.column_id)`,
		taskID,
	).Scan(&boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrTaskNotFound
		}
		return uuid.Nil, fmt.Errorf("delete task: %w", err)
	}
	return boardID, nil
}

// IssuePublicLink persists a new public link for a board (AC-07). An FK
// violation means link.BoardID does not exist → domain.ErrBoardNotFound.
func (r *PostgresRepository) IssuePublicLink(ctx context.Context, link *domain.PublicLink) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO public_links (id, board_id, token)
		 VALUES ($1, $2, $3)
		 RETURNING created_at`,
		link.ID, link.BoardID, link.Token,
	).Scan(&link.CreatedAt)
	if err != nil {
		if database.IsPgUniqueViolation(err) {
			return domain.ErrLinkAlreadyActive
		}
		if database.IsPgForeignKeyViolation(err) {
			return domain.ErrBoardNotFound
		}
		return fmt.Errorf("issue public link: %w", err)
	}
	return nil
}

// RevokePublicLink hard-deletes the board's active public link (AC-08).
func (r *PostgresRepository) RevokePublicLink(ctx context.Context, boardID uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM public_links WHERE board_id = $1`, boardID)
	if err != nil {
		return fmt.Errorf("revoke public link: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrLinkNotFound
	}
	return nil
}

// PublicLinkByToken looks up a public link by its opaque token
// (AC-09/AC-11).
func (r *PostgresRepository) PublicLinkByToken(ctx context.Context, token string) (*domain.PublicLink, error) {
	var link domain.PublicLink
	err := r.db.QueryRow(ctx,
		`SELECT id, board_id, token, created_at FROM public_links WHERE token = $1`, token,
	).Scan(&link.ID, &link.BoardID, &link.Token, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLinkNotFound
		}
		return nil, fmt.Errorf("public link by token: %w", err)
	}
	return &link, nil
}

// PublicLinkByBoard looks up a board's active public link, so RevokePublicLink
// callers know which token to close before deleting the row (AC-08).
func (r *PostgresRepository) PublicLinkByBoard(ctx context.Context, boardID uuid.UUID) (*domain.PublicLink, error) {
	link, err := r.publicLinkForBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("public link by board: %w", err)
	}
	if link == nil {
		return nil, domain.ErrLinkNotFound
	}
	return link, nil
}
