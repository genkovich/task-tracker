// Package domain models the board module's entities: the single Board
// aggregate, its fixed Columns, the Tasks within them, and the optional
// PublicLink used for read-only sharing. No framework imports (chi, pgx,
// HTTP) — see .claude/rules/go-structs-interfaces.md.
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrTitleRequired — a task title must be non-empty (AC-02).
	ErrTitleRequired = errors.New("board.task_title_required")
	// ErrTaskNotFound — no task exists for the given id.
	ErrTaskNotFound = errors.New("board.task_not_found")
	// ErrColumnNotFound — no column exists for the given id.
	ErrColumnNotFound = errors.New("board.column_not_found")
	// ErrLinkNotFound — no public link exists for the given board/token.
	ErrLinkNotFound = errors.New("board.link_not_found")
	// ErrLinkAlreadyActive — the board already has an active public link.
	ErrLinkAlreadyActive = errors.New("board.link_already_active")
)

// Board is the aggregate root — the product always has exactly one row
// (CONTEXT.md invariant).
type Board struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

// Column is one of the fixed display slots ("To Do" / "In Progress" /
// "Done", ADR-0004) a board's tasks live in. There is no CRUD for columns —
// this type deliberately carries no method that would imply otherwise.
type Column struct {
	ID        uuid.UUID
	BoardID   uuid.UUID
	Name      string
	Position  int16
	CreatedAt time.Time
}

// Task is a unit of work. Its ColumnID is its only status field — there is
// no separate status column (data-model.md).
type Task struct {
	ID        uuid.UUID
	ColumnID  uuid.UUID
	Title     string
	Assignee  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTask constructs a Task in the given column, enforcing a non-empty
// title (AC-02).
func NewTask(columnID uuid.UUID, title string, assignee *string) (*Task, error) {
	if !isValidTitle(title) {
		return nil, ErrTitleRequired
	}

	return &Task{
		ID:       uuid.Must(uuid.NewV7()),
		ColumnID: columnID,
		Title:    title,
		Assignee: assignee,
	}, nil
}

// SetTitle updates the task's title, enforcing the same non-empty
// invariant as NewTask (AC-02 applies to edits too).
func (t *Task) SetTitle(title string) error {
	if !isValidTitle(title) {
		return ErrTitleRequired
	}

	t.Title = title
	return nil
}

func isValidTitle(title string) bool {
	return strings.TrimSpace(title) != ""
}

// PublicLink grants read-only access to a board via an opaque token
// (ADR-0003). At most one active link per board.
type PublicLink struct {
	ID        uuid.UUID
	BoardID   uuid.UUID
	Token     string
	CreatedAt time.Time
}
