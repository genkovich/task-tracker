package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// AC-02 (US-01) — error: creating a task with an empty title is blocked
// and the caller is told the title is required.
func TestNewTask_EmptyTitle_ReturnsErrTitleRequired(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	got, err := domain.NewTask(columnID, "", nil)

	if !errors.Is(err, domain.ErrTitleRequired) {
		t.Fatalf("NewTask(empty title) error = %v, want errors.Is(err, domain.ErrTitleRequired)", err)
	}
	if got != nil {
		t.Fatalf("NewTask(empty title) task = %+v, want nil on error", got)
	}
}

// AC-02 (US-01) — a whitespace-only title is not a valid non-empty title either.
func TestNewTask_WhitespaceOnlyTitle_ReturnsErrTitleRequired(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	got, err := domain.NewTask(columnID, "   ", nil)

	if !errors.Is(err, domain.ErrTitleRequired) {
		t.Fatalf("NewTask(whitespace title) error = %v, want errors.Is(err, domain.ErrTitleRequired)", err)
	}
	if got != nil {
		t.Fatalf("NewTask(whitespace title) task = %+v, want nil on error", got)
	}
}

// AC-01/AC-02 happy path counterpart: a non-empty title succeeds and the task
// carries the given column as its status.
func TestNewTask_NonEmptyTitle_Succeeds(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	assignee := "Alex"

	got, err := domain.NewTask(columnID, "Write the report", &assignee)

	if err != nil {
		t.Fatalf("NewTask(non-empty title) unexpected error = %v", err)
	}
	if got == nil {
		t.Fatal("NewTask(non-empty title) task = nil, want non-nil")
	}
	if got.ID == uuid.Nil {
		t.Fatal("NewTask(non-empty title) ID is zero value, want a generated UUID")
	}
	if got.ColumnID != columnID {
		t.Fatalf("NewTask(non-empty title) ColumnID = %v, want %v", got.ColumnID, columnID)
	}
	if got.Title != "Write the report" {
		t.Fatalf("NewTask(non-empty title) Title = %q, want %q", got.Title, "Write the report")
	}
}

// SetTitle enforces the same non-empty invariant on update (AC-02 applies to
// edits too, per data-model.md "non-empty enforced at app layer").
func TestTask_SetTitle_EmptyTitle_ReturnsErrTitleRequired(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	task, err := domain.NewTask(columnID, "Original title", nil)
	if err != nil {
		t.Fatalf("NewTask() unexpected error = %v", err)
	}

	err = task.SetTitle("")

	if !errors.Is(err, domain.ErrTitleRequired) {
		t.Fatalf("SetTitle(empty) error = %v, want errors.Is(err, domain.ErrTitleRequired)", err)
	}
	if task.Title != "Original title" {
		t.Fatalf("SetTitle(empty) mutated Title to %q, want unchanged %q", task.Title, "Original title")
	}
}
