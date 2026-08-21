// Package domain models the board module's entities: Board aggregates (many,
// created from the dashboard — boards BRD-02), their fixed Columns, the Tasks
// within them (with their details and Comments — tasks TSK-01…TSK-09), and the
// optional per-board PublicLink used for read-only sharing. No framework
// imports (chi, pgx, HTTP) — see .claude/rules/go-structs-interfaces.md.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxTitleLength bounds a task title in characters — mirrors the contract's
// maxLength: 200 (contracts/openapi.yaml Task.title) and the tasks.title
// column width; without the domain check an oversized title surfaced as an
// opaque DB error → 500 instead of 422.
const MaxTitleLength = 200

// MaxAssigneeLength bounds a task assignee in characters — mirrors the
// contract's maxLength: 200 (contracts/openapi.yaml Task.assignee) and the
// tasks.assignee column width, symmetric with MaxTitleLength; without the
// domain check an oversized assignee surfaced as an opaque DB error → 500
// instead of 422.
const MaxAssigneeLength = 200

// MaxBoardNameLength bounds a board name in characters — mirrors the boards
// contract's maxLength: 200 (BoardCreate.name) and the boards.name column
// width, symmetric with MaxTitleLength.
const MaxBoardNameLength = 200

// MaxDescriptionLength bounds a task description in characters (tasks
// TSK-02). The column itself is TEXT — the bound lives here on purpose, so
// moving it later is a code change, not a migration.
const MaxDescriptionLength = 4000

// MaxCommentAuthorLength bounds a comment author in characters — mirrors the
// task_comments.author column width and MaxAssigneeLength: both are free text
// naming a person with no account behind it (ADR-0001).
const MaxCommentAuthorLength = 200

// MaxCommentBodyLength bounds a comment body in characters — mirrors the
// task_comments.body column width (tasks TSK-09).
const MaxCommentBodyLength = 2000

var (
	// ErrTitleRequired — a task title must be non-empty (AC-02).
	ErrTitleRequired = errors.New("board.task_title_required")
	// ErrTitleTooLong — a task title must be at most MaxTitleLength characters.
	ErrTitleTooLong = errors.New("board.task_title_too_long")
	// ErrAssigneeTooLong — a task assignee must be at most MaxAssigneeLength
	// characters.
	ErrAssigneeTooLong = errors.New("board.task_assignee_too_long")
	// ErrDescriptionTooLong — a task description must be at most
	// MaxDescriptionLength characters (TSK-02).
	ErrDescriptionTooLong = errors.New("board.task_description_too_long")
	// ErrPriorityInvalid — a task priority must be one of the three fixed
	// values (TSK-04).
	ErrPriorityInvalid = errors.New("board.task_priority_invalid")
	// ErrCommentAuthorRequired — a comment author must be non-empty (TSK-09).
	ErrCommentAuthorRequired = errors.New("board.comment_author_required")
	// ErrCommentAuthorTooLong — a comment author must be at most
	// MaxCommentAuthorLength characters (TSK-09).
	ErrCommentAuthorTooLong = errors.New("board.comment_author_too_long")
	// ErrCommentBodyRequired — a comment body must be non-empty (TSK-09).
	ErrCommentBodyRequired = errors.New("board.comment_body_required")
	// ErrCommentBodyTooLong — a comment body must be at most
	// MaxCommentBodyLength characters (TSK-09).
	ErrCommentBodyTooLong = errors.New("board.comment_body_too_long")
	// ErrCommentNotFound — no comment exists for the given id.
	ErrCommentNotFound = errors.New("board.comment_not_found")
	// ErrBoardNameRequired — a board name must be non-empty (BRD-03).
	ErrBoardNameRequired = errors.New("board.name_required")
	// ErrBoardNameTooLong — a board name must be at most MaxBoardNameLength
	// characters (BRD-03).
	ErrBoardNameTooLong = errors.New("board.name_too_long")
	// ErrBoardNotFound — no board exists for the given id.
	ErrBoardNotFound = errors.New("board.not_found")
	// ErrTaskNotFound — no task exists for the given id.
	ErrTaskNotFound = errors.New("board.task_not_found")
	// ErrColumnNotFound — no column exists for the given id.
	ErrColumnNotFound = errors.New("board.column_not_found")
	// ErrLinkNotFound — no public link exists for the given board/token.
	ErrLinkNotFound = errors.New("board.link_not_found")
	// ErrLinkAlreadyActive — the board already has an active public link.
	ErrLinkAlreadyActive = errors.New("board.link_already_active")
)

// Board is the aggregate root. Boards are many (boards BRD-02); every board
// always carries exactly the three fixed columns (CONTEXT.md invariant).
type Board struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// NewBoard constructs a Board with a non-empty, length-bounded name (BRD-03).
func NewBoard(name string) (*Board, error) {
	if err := validateBoardName(name); err != nil {
		return nil, err
	}

	return &Board{
		ID:   uuid.Must(uuid.NewV7()),
		Name: name,
	}, nil
}

func validateBoardName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrBoardNameRequired
	}
	if utf8.RuneCountInString(name) > MaxBoardNameLength {
		return ErrBoardNameTooLong
	}
	return nil
}

// DefaultColumns returns the fixed column set every board carries (CONTEXT.md
// invariant: exactly three fixed columns; ADR-0004 — no column CRUD), ordered
// left to right.
func DefaultColumns(boardID uuid.UUID) []Column {
	names := []string{"To Do", "In Progress", "Done"}
	columns := make([]Column, 0, len(names))
	for i, name := range names {
		columns = append(columns, Column{
			ID:       uuid.Must(uuid.NewV7()),
			BoardID:  boardID,
			Name:     name,
			Position: int16(i), //nolint:gosec // i is 0..2
		})
	}
	return columns
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

// Priority ranks a task against its neighbours in the same column (tasks
// TSK-03). A closed set of three — there is no numeric scale and no custom
// level.
type Priority string

// The three priorities a task can carry. A task created without an explicit
// choice is PriorityMedium.
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// ParsePriority turns a wire value into a Priority. The empty string means
// "not chosen" and yields PriorityMedium (TSK-03); anything outside the three
// values is ErrPriorityInvalid (TSK-04) — the schema's CHECK constraint is
// only a second line behind this one.
func ParsePriority(raw string) (Priority, error) {
	switch p := Priority(raw); p {
	case "":
		return PriorityMedium, nil
	case PriorityLow, PriorityMedium, PriorityHigh:
		return p, nil
	default:
		return "", ErrPriorityInvalid
	}
}

// Task is a unit of work. Its ColumnID is its only status field — there is
// no separate status column (data-model.md).
type Task struct {
	ID          uuid.UUID
	ColumnID    uuid.UUID
	Title       string
	Assignee    *string
	Description string
	Priority    Priority
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskDetails carries everything a caller may set on a task, so create and
// edit take one argument instead of five positional ones that are trivially
// swappable at the call site. Priority is the raw wire value — the empty
// string is a legitimate "not chosen" (TSK-03), which is why it is not a
// Priority yet.
type TaskDetails struct {
	Title       string
	Assignee    *string
	Description string
	Priority    string
	DueDate     *time.Time
}

// NewTask constructs a Task in the given column from the supplied details,
// enforcing a non-empty (AC-02), length-bounded title and every tasks-feature
// bound alongside it.
func NewTask(columnID uuid.UUID, details TaskDetails) (*Task, error) {
	task := &Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID}
	if err := task.SetDetails(details); err != nil {
		return nil, err
	}
	return task, nil
}

// SetDetails replaces every editable field of the task, enforcing the same
// invariants as NewTask (AC-02 applies to edits too). Everything is validated
// before anything is written, so a rejected edit leaves the task exactly as it
// was — a half-applied edit would be worse than no edit at all.
func (t *Task) SetDetails(details TaskDetails) error {
	if err := validateTitle(details.Title); err != nil {
		return err
	}
	if err := validateAssignee(details.Assignee); err != nil {
		return err
	}
	if err := validateDescription(details.Description); err != nil {
		return err
	}
	priority, err := ParsePriority(details.Priority)
	if err != nil {
		return err
	}

	t.Title = details.Title
	t.Assignee = details.Assignee
	t.Description = details.Description
	t.Priority = priority
	t.DueDate = truncateToDay(details.DueDate)
	return nil
}

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return ErrTitleRequired
	}
	if utf8.RuneCountInString(title) > MaxTitleLength {
		return ErrTitleTooLong
	}
	return nil
}

func validateAssignee(assignee *string) error {
	if assignee != nil && utf8.RuneCountInString(*assignee) > MaxAssigneeLength {
		return ErrAssigneeTooLong
	}
	return nil
}

func validateDescription(description string) error {
	if utf8.RuneCountInString(description) > MaxDescriptionLength {
		return ErrDescriptionTooLong
	}
	return nil
}

// truncateToDay drops the time of day from a due date: the column is DATE and
// a deadline is set by the day, so keeping hours would be invented precision —
// and would make the same deadline read differently across time zones.
func truncateToDay(due *time.Time) *time.Time {
	if due == nil {
		return nil
	}
	day := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	return &day
}

// Comment is one message under a task (tasks TSK-08). Author is free text
// with no account behind it (ADR-0001, same as a task's assignee), and a
// comment is never edited — hence no UpdatedAt.
type Comment struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	Author    string
	Body      string
	CreatedAt time.Time
}

// NewComment constructs a comment under taskID, enforcing a non-empty,
// length-bounded author and body (TSK-09).
func NewComment(taskID uuid.UUID, author, body string) (*Comment, error) {
	if strings.TrimSpace(author) == "" {
		return nil, ErrCommentAuthorRequired
	}
	if utf8.RuneCountInString(author) > MaxCommentAuthorLength {
		return nil, ErrCommentAuthorTooLong
	}
	if strings.TrimSpace(body) == "" {
		return nil, ErrCommentBodyRequired
	}
	if utf8.RuneCountInString(body) > MaxCommentBodyLength {
		return nil, ErrCommentBodyTooLong
	}

	return &Comment{
		ID:     uuid.Must(uuid.NewV7()),
		TaskID: taskID,
		Author: author,
		Body:   body,
	}, nil
}

// PublicLink grants read-only access to a board via an opaque token
// (ADR-0003). At most one active link per board.
type PublicLink struct {
	ID        uuid.UUID
	BoardID   uuid.UUID
	Token     string
	CreatedAt time.Time
}
