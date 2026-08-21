package domain_test

import (
	"errors"
	"strings"
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

// Review 2026-08-21, root K: a title over MaxTitleLength characters must be
// rejected in the domain — without this check it used to surface as an
// opaque DB error → 500 instead of 422. The boundary (exactly 200 runes,
// non-ASCII included) stays valid.
func TestNewTask_TitleLength_Boundary(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	atLimit := strings.Repeat("ї", domain.MaxTitleLength)
	if _, err := domain.NewTask(columnID, atLimit, nil); err != nil {
		t.Fatalf("NewTask(title of exactly %d runes) error = %v, want nil", domain.MaxTitleLength, err)
	}

	overLimit := strings.Repeat("ї", domain.MaxTitleLength+1)
	got, err := domain.NewTask(columnID, overLimit, nil)
	if !errors.Is(err, domain.ErrTitleTooLong) {
		t.Fatalf("NewTask(title of %d runes) error = %v, want errors.Is(err, domain.ErrTitleTooLong)", domain.MaxTitleLength+1, err)
	}
	if got != nil {
		t.Fatalf("NewTask(oversized title) task = %+v, want nil on error", got)
	}
}

// SetTitle enforces the same length invariant as NewTask.
func TestTask_SetTitle_TooLong_ReturnsErrTitleTooLong(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	task, err := domain.NewTask(columnID, "ok", nil)
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}

	if err := task.SetTitle(strings.Repeat("a", domain.MaxTitleLength+1)); !errors.Is(err, domain.ErrTitleTooLong) {
		t.Fatalf("SetTitle(oversized) error = %v, want errors.Is(err, domain.ErrTitleTooLong)", err)
	}
	if task.Title != "ok" {
		t.Fatalf("SetTitle(oversized) must leave the title unchanged, got %q", task.Title)
	}
}
