package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// titled is the minimal valid details set — every test that only cares about
// one field starts from it and overrides that field.
func titled(title string) domain.TaskDetails {
	return domain.TaskDetails{Title: title}
}

// AC-02 (US-01) — error: creating a task with an empty title is blocked
// and the caller is told the title is required.
func TestNewTask_EmptyTitle_ReturnsErrTitleRequired(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	got, err := domain.NewTask(columnID, titled(""))

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

	got, err := domain.NewTask(columnID, titled("   "))

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

	details := titled("Write the report")
	details.Assignee = &assignee

	got, err := domain.NewTask(columnID, details)
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

// SetDetails enforces the same non-empty invariant on update (AC-02 applies to
// edits too, per data-model.md "non-empty enforced at app layer").
func TestTask_SetDetails_EmptyTitle_ReturnsErrTitleRequired(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	task, err := domain.NewTask(columnID, titled("Original title"))
	if err != nil {
		t.Fatalf("NewTask() unexpected error = %v", err)
	}

	err = task.SetDetails(titled(""))

	if !errors.Is(err, domain.ErrTitleRequired) {
		t.Fatalf("SetDetails(empty title) error = %v, want errors.Is(err, domain.ErrTitleRequired)", err)
	}
	if task.Title != "Original title" {
		t.Fatalf("SetDetails(empty title) mutated Title to %q, want unchanged %q", task.Title, "Original title")
	}
}

// Review 2026-08-21, root K: a title over MaxTitleLength characters must be
// rejected in the domain — without this check it used to surface as an
// opaque DB error → 500 instead of 422. The boundary (exactly 200 runes,
// non-ASCII included) stays valid.
func TestNewTask_TitleLength_Boundary(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	atLimit := strings.Repeat("ї", domain.MaxTitleLength)
	if _, err := domain.NewTask(columnID, titled(atLimit)); err != nil {
		t.Fatalf("NewTask(title of exactly %d runes) error = %v, want nil", domain.MaxTitleLength, err)
	}

	overLimit := strings.Repeat("ї", domain.MaxTitleLength+1)
	got, err := domain.NewTask(columnID, titled(overLimit))
	if !errors.Is(err, domain.ErrTitleTooLong) {
		t.Fatalf("NewTask(title of %d runes) error = %v, want errors.Is(err, domain.ErrTitleTooLong)", domain.MaxTitleLength+1, err)
	}
	if got != nil {
		t.Fatalf("NewTask(oversized title) task = %+v, want nil on error", got)
	}
}

// Review 2026-08-21 re2, #1: an assignee over MaxAssigneeLength characters
// must be rejected in the domain, symmetric with the title — without the
// check it surfaced as an opaque DB error (VARCHAR(200)) → 500 instead of
// 422. The boundary (exactly 200 runes, non-ASCII included) stays valid.
func TestNewTask_AssigneeLength_Boundary(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	atLimit := strings.Repeat("ї", domain.MaxAssigneeLength)
	details := titled("ok")
	details.Assignee = &atLimit
	if _, err := domain.NewTask(columnID, details); err != nil {
		t.Fatalf("NewTask(assignee of exactly %d runes) error = %v, want nil", domain.MaxAssigneeLength, err)
	}

	overLimit := strings.Repeat("ї", domain.MaxAssigneeLength+1)
	details.Assignee = &overLimit
	got, err := domain.NewTask(columnID, details)
	if !errors.Is(err, domain.ErrAssigneeTooLong) {
		t.Fatalf("NewTask(assignee of %d runes) error = %v, want errors.Is(err, domain.ErrAssigneeTooLong)", domain.MaxAssigneeLength+1, err)
	}
	if got != nil {
		t.Fatalf("NewTask(oversized assignee) task = %+v, want nil on error", got)
	}
}

// SetDetails enforces the same length invariant on edit (the contract's
// maxLength: 200 applies to TaskUpdate too); nil clears the assignee.
func TestTask_SetDetails_Assignee_Boundary(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	task, err := domain.NewTask(columnID, titled("ok"))
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}

	atLimit := strings.Repeat("ї", domain.MaxAssigneeLength)
	details := titled("ok")
	details.Assignee = &atLimit
	if err := task.SetDetails(details); err != nil {
		t.Fatalf("SetDetails(assignee of exactly %d runes) error = %v, want nil", domain.MaxAssigneeLength, err)
	}

	overLimit := strings.Repeat("ї", domain.MaxAssigneeLength+1)
	details.Assignee = &overLimit
	if err := task.SetDetails(details); !errors.Is(err, domain.ErrAssigneeTooLong) {
		t.Fatalf("SetDetails(oversized assignee) error = %v, want errors.Is(err, domain.ErrAssigneeTooLong)", err)
	}
	if task.Assignee == nil || *task.Assignee != atLimit {
		t.Fatal("SetDetails(oversized assignee) must leave the assignee unchanged")
	}

	if err := task.SetDetails(titled("ok")); err != nil {
		t.Fatalf("SetDetails(no assignee) error = %v, want nil (clearing is always valid)", err)
	}
	if task.Assignee != nil {
		t.Fatal("SetDetails without an assignee must clear it")
	}
}

// SetDetails enforces the same title length invariant as NewTask.
func TestTask_SetDetails_TitleTooLong_ReturnsErrTitleTooLong(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	task, err := domain.NewTask(columnID, titled("ok"))
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}

	if err := task.SetDetails(titled(strings.Repeat("a", domain.MaxTitleLength+1))); !errors.Is(err, domain.ErrTitleTooLong) {
		t.Fatalf("SetDetails(oversized title) error = %v, want errors.Is(err, domain.ErrTitleTooLong)", err)
	}
	if task.Title != "ok" {
		t.Fatalf("SetDetails(oversized title) must leave the title unchanged, got %q", task.Title)
	}
}

// TSK-02: a description over MaxDescriptionLength characters is rejected;
// exactly at the limit (non-ASCII included) stays valid.
func TestNewTask_DescriptionLength_Boundary(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())

	details := titled("ok")
	details.Description = strings.Repeat("ї", domain.MaxDescriptionLength)
	if _, err := domain.NewTask(columnID, details); err != nil {
		t.Fatalf("NewTask(description of exactly %d runes) error = %v, want nil", domain.MaxDescriptionLength, err)
	}

	details.Description = strings.Repeat("ї", domain.MaxDescriptionLength+1)
	got, err := domain.NewTask(columnID, details)
	if !errors.Is(err, domain.ErrDescriptionTooLong) {
		t.Fatalf("NewTask(description of %d runes) error = %v, want errors.Is(err, domain.ErrDescriptionTooLong)", domain.MaxDescriptionLength+1, err)
	}
	if got != nil {
		t.Fatalf("NewTask(oversized description) task = %+v, want nil on error", got)
	}
}

// TSK-02: a rejected description must leave every other field untouched —
// SetDetails validates everything before it writes anything.
func TestTask_SetDetails_OversizedDescription_LeavesTaskUnchanged(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	original := titled("Original title")
	original.Description = "kept"
	original.Priority = string(domain.PriorityHigh)
	task, err := domain.NewTask(columnID, original)
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}

	rejected := titled("New title")
	rejected.Description = strings.Repeat("a", domain.MaxDescriptionLength+1)
	rejected.Priority = string(domain.PriorityLow)

	if err := task.SetDetails(rejected); !errors.Is(err, domain.ErrDescriptionTooLong) {
		t.Fatalf("SetDetails(oversized description) error = %v, want ErrDescriptionTooLong", err)
	}
	if task.Title != "Original title" || task.Description != "kept" || task.Priority != domain.PriorityHigh {
		t.Fatalf("rejected SetDetails half-applied the edit: %+v", task)
	}
}

// TSK-03: the three priorities parse; the empty string means "not chosen"
// and yields medium.
func TestParsePriority_AcceptedValues(t *testing.T) {
	cases := map[string]domain.Priority{
		"":       domain.PriorityMedium,
		"low":    domain.PriorityLow,
		"medium": domain.PriorityMedium,
		"high":   domain.PriorityHigh,
	}

	for raw, want := range cases {
		got, err := domain.ParsePriority(raw)
		if err != nil {
			t.Fatalf("ParsePriority(%q) error = %v, want nil", raw, err)
		}
		if got != want {
			t.Fatalf("ParsePriority(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TSK-04: anything outside the closed set is rejected — including values that
// merely look plausible.
func TestParsePriority_UnknownValue_ReturnsErrPriorityInvalid(t *testing.T) {
	for _, raw := range []string{"urgent", "LOW", "0", "critical"} {
		got, err := domain.ParsePriority(raw)
		if !errors.Is(err, domain.ErrPriorityInvalid) {
			t.Fatalf("ParsePriority(%q) error = %v, want errors.Is(err, domain.ErrPriorityInvalid)", raw, err)
		}
		if got != "" {
			t.Fatalf("ParsePriority(%q) = %q, want the zero value on error", raw, got)
		}
	}
}

// TSK-03: a task created without an explicit priority is medium, not empty —
// the card always has a marker to draw.
func TestNewTask_DefaultsToMediumPriority(t *testing.T) {
	task, err := domain.NewTask(uuid.Must(uuid.NewV7()), titled("ok"))
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}
	if task.Priority != domain.PriorityMedium {
		t.Fatalf("NewTask default Priority = %q, want %q", task.Priority, domain.PriorityMedium)
	}
}

// TSK-05/TSK-07: a due date is optional, keeps only its calendar day (the
// column is DATE), and can be cleared by leaving it out of the details.
func TestTask_DueDate_TruncatedToDayAndClearable(t *testing.T) {
	columnID := uuid.Must(uuid.NewV7())
	due := time.Date(2026, time.September, 1, 17, 43, 12, 0, time.FixedZone("EEST", 3*60*60))

	details := titled("ok")
	details.DueDate = &due
	task, err := domain.NewTask(columnID, details)
	if err != nil {
		t.Fatalf("NewTask error = %v", err)
	}
	if task.DueDate == nil {
		t.Fatal("NewTask(with due date) DueDate = nil, want the given day")
	}
	gotY, gotM, gotD := task.DueDate.Date()
	if gotY != 2026 || gotM != time.September || gotD != 1 {
		t.Fatalf("DueDate = %v, want 2026-09-01", task.DueDate)
	}
	if h, m, s := task.DueDate.Clock(); h != 0 || m != 0 || s != 0 {
		t.Fatalf("DueDate kept a time of day (%02d:%02d:%02d), want midnight", h, m, s)
	}

	if err := task.SetDetails(titled("ok")); err != nil {
		t.Fatalf("SetDetails(no due date) error = %v, want nil", err)
	}
	if task.DueDate != nil {
		t.Fatalf("SetDetails without a due date must clear it, got %v", task.DueDate)
	}
}

// TSK-09: a comment needs a non-empty author and body, each within its bound.
func TestNewComment_Validation(t *testing.T) {
	taskID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name    string
		author  string
		body    string
		wantErr error
	}{
		{"empty author", "", "hello", domain.ErrCommentAuthorRequired},
		{"whitespace author", "   ", "hello", domain.ErrCommentAuthorRequired},
		{"author too long", strings.Repeat("ї", domain.MaxCommentAuthorLength+1), "hello", domain.ErrCommentAuthorTooLong},
		{"empty body", "Ada", "", domain.ErrCommentBodyRequired},
		{"whitespace body", "Ada", "  \n ", domain.ErrCommentBodyRequired},
		{"body too long", "Ada", strings.Repeat("ї", domain.MaxCommentBodyLength+1), domain.ErrCommentBodyTooLong},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewComment(taskID, tc.author, tc.body)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewComment(%s) error = %v, want errors.Is(err, %v)", tc.name, err, tc.wantErr)
			}
			if got != nil {
				t.Fatalf("NewComment(%s) comment = %+v, want nil on error", tc.name, got)
			}
		})
	}
}

// TSK-08/TSK-09: exactly at both bounds a comment is valid and carries its
// task, a generated id and the given text.
func TestNewComment_Boundary_Succeeds(t *testing.T) {
	taskID := uuid.Must(uuid.NewV7())
	author := strings.Repeat("ї", domain.MaxCommentAuthorLength)
	body := strings.Repeat("ї", domain.MaxCommentBodyLength)

	got, err := domain.NewComment(taskID, author, body)
	if err != nil {
		t.Fatalf("NewComment(at both limits) error = %v, want nil", err)
	}
	if got.ID == uuid.Nil {
		t.Fatal("NewComment ID is the zero value, want a generated UUID")
	}
	if got.TaskID != taskID {
		t.Fatalf("NewComment TaskID = %v, want %v", got.TaskID, taskID)
	}
	if got.Author != author || got.Body != body {
		t.Fatal("NewComment must keep the author and body it was given")
	}
}
