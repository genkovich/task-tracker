package app_test

// Unit tests for BoardService's task use-cases (T5): CreateTask, EditTask,
// MoveTask, DeleteTask. The repository (ports.Repository) and broadcaster
// (ports.Broadcaster) are faked — no Postgres, no SSE — matching the
// "unit tests (repo faked via the ports.Repository interface)" DoD line in
// docs/features/board/tasks/T5-app-task-usecases.md.
//
// No production code exists yet for app.NewTaskService / app.TaskService —
// this is the RED step; the test intentionally references a symbol that
// does not exist so it fails to compile until T5 is implemented.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// ---------------------------------------------------------------------------
// fakeRepo — test double for ports.Repository
// ---------------------------------------------------------------------------

type fakeRepo struct {
	mu sync.Mutex

	boardID   uuid.UUID
	leftmost  uuid.UUID
	columns   map[uuid.UUID]bool
	tasks     map[uuid.UUID]domain.Task
	insertErr error
}

func newFakeRepo(boardID, leftmostColumnID uuid.UUID, otherColumnIDs ...uuid.UUID) *fakeRepo {
	columns := map[uuid.UUID]bool{leftmostColumnID: true}
	for _, id := range otherColumnIDs {
		columns[id] = true
	}
	return &fakeRepo{
		boardID:  boardID,
		leftmost: leftmostColumnID,
		columns:  columns,
		tasks:    make(map[uuid.UUID]domain.Task),
	}
}

func (r *fakeRepo) GetBoardState(_ context.Context, _ uuid.UUID) (*ports.BoardState, error) {
	return &ports.BoardState{}, nil
}

func (r *fakeRepo) LeftmostColumnID(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return r.leftmost, nil
}

func (r *fakeRepo) InsertTask(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.insertErr != nil {
		return r.insertErr
	}
	if !r.columns[task.ColumnID] {
		return domain.ErrColumnNotFound
	}
	r.tasks[task.ID] = *task
	return nil
}

// UpdateTask mirrors the real repository's contract honestly (review
// 2026-08-21, root G): only title/assignee change on the stored row — never
// a full replacement, which would mask a service handing over a Task with
// zero column_id/created_at — and, like the SQL RETURNING, the caller's
// task is back-filled with the stored row's remaining fields.
func (r *fakeRepo) UpdateTask(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.tasks[task.ID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	stored.Title = task.Title
	stored.Assignee = task.Assignee
	stored.UpdatedAt = time.Now()
	r.tasks[task.ID] = stored
	*task = stored
	return nil
}

func (r *fakeRepo) MoveTask(_ context.Context, taskID, columnID uuid.UUID) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[taskID]
	if !ok {
		return nil, domain.ErrTaskNotFound
	}
	if !r.columns[columnID] {
		return nil, domain.ErrColumnNotFound
	}
	task.ColumnID = columnID
	task.UpdatedAt = time.Now()
	r.tasks[taskID] = task
	return &task, nil
}

func (r *fakeRepo) DeleteTask(_ context.Context, taskID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[taskID]; !ok {
		return domain.ErrTaskNotFound
	}
	delete(r.tasks, taskID)
	return nil
}

func (r *fakeRepo) ColumnExists(_ context.Context, columnID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.columns[columnID], nil
}

func (r *fakeRepo) IssuePublicLink(_ context.Context, _ *domain.PublicLink) error {
	return errors.New("not used by task_service tests")
}

func (r *fakeRepo) RevokePublicLink(_ context.Context, _ uuid.UUID) error {
	return errors.New("not used by task_service tests")
}

func (r *fakeRepo) PublicLinkByToken(_ context.Context, _ string) (*domain.PublicLink, error) {
	return nil, errors.New("not used by task_service tests")
}

func (r *fakeRepo) PublicLinkByBoard(_ context.Context, _ uuid.UUID) (*domain.PublicLink, error) {
	return nil, errors.New("not used by task_service tests")
}

func (r *fakeRepo) seedTask(task domain.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
}

func (r *fakeRepo) getTask(id uuid.UUID) (domain.Task, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[id]
	return task, ok
}

func (r *fakeRepo) taskCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.tasks)
}

// ---------------------------------------------------------------------------
// fakeBroadcaster — test double for ports.Broadcaster
// ---------------------------------------------------------------------------

type fakeBroadcaster struct {
	mu           sync.Mutex
	broadcasts   int
	closedTokens []string
}

func (b *fakeBroadcaster) Broadcast(_ ports.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.broadcasts++
}

func (b *fakeBroadcaster) CloseToken(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closedTokens = append(b.closedTokens, token)
}

func (b *fakeBroadcaster) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.broadcasts
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestCreateTask_NonEmptyTitle_LandsInLeftmostColumn covers AC-01: creating a
// task with a non-empty title inserts it into the board's leftmost column,
// and the mutation broadcasts exactly once.
func TestCreateTask_NonEmptyTitle_LandsInLeftmostColumn(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	task, err := svc.CreateTask(context.Background(), "Write the report", nil)

	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, leftmostID, task.ColumnID, "new task must land in the leftmost column (AC-01)")

	stored, ok := repo.getTask(task.ID)
	require.True(t, ok, "task must be persisted via the repository")
	require.Equal(t, leftmostID, stored.ColumnID)

	require.Equal(t, 1, bcast.count(), "a successful mutation must broadcast exactly once")
}

// TestCreateTask_EmptyTitle_RejectedNoWriteNoBroadcast covers AC-02: an empty
// title is rejected before any repository write, and no broadcast fires.
func TestCreateTask_EmptyTitle_RejectedNoWriteNoBroadcast(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	task, err := svc.CreateTask(context.Background(), "", nil)

	require.ErrorIs(t, err, domain.ErrTitleRequired)
	require.Nil(t, task)
	require.Equal(t, 0, repo.taskCount(), "no task should be persisted when the title is empty")
	require.Equal(t, 0, bcast.count(), "a rejected creation must not broadcast")
}

// TestEditTask_UpdatesTitleAndAssignee covers AC-03: editing an existing
// task's title/assignee persists the new values and broadcasts once.
func TestEditTask_UpdatesTitleAndAssignee(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	createdAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	existing := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: leftmostID, Title: "Original title", CreatedAt: createdAt}
	repo.seedTask(existing)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	newAssignee := "Alex"
	updated, err := svc.EditTask(context.Background(), existing.ID, "New title", &newAssignee)

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "New title", updated.Title)
	require.NotNil(t, updated.Assignee)
	require.Equal(t, "Alex", *updated.Assignee)
	// Root G pin: the returned Task must be complete, not just the edited
	// fields — column_id/created_at come from the stored row.
	require.Equal(t, leftmostID, updated.ColumnID, "EditTask must return the task's real column_id")
	require.Equal(t, createdAt, updated.CreatedAt, "EditTask must return the task's real created_at")
	require.False(t, updated.UpdatedAt.IsZero(), "EditTask must return a fresh updated_at")

	stored, ok := repo.getTask(existing.ID)
	require.True(t, ok)
	require.Equal(t, "New title", stored.Title)
	require.NotNil(t, stored.Assignee)
	require.Equal(t, "Alex", *stored.Assignee)

	require.Equal(t, 1, bcast.count(), "a successful edit must broadcast exactly once")
}

// TestMoveTask_ValidColumn_UpdatesColumnID covers AC-04: moving a task to a
// valid column persists the new column and broadcasts once.
func TestMoveTask_ValidColumn_UpdatesColumnID(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	targetID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID, targetID)
	existing := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: leftmostID, Title: "Move me"}
	repo.seedTask(existing)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	moved, err := svc.MoveTask(context.Background(), existing.ID, targetID)

	require.NoError(t, err)
	stored, ok := repo.getTask(existing.ID)
	require.True(t, ok)
	require.Equal(t, targetID, stored.ColumnID, "task must be recorded in the new column (AC-04)")

	// Root G pin: MoveTask returns the complete moved task, not a stub.
	require.NotNil(t, moved)
	require.Equal(t, existing.ID, moved.ID)
	require.Equal(t, targetID, moved.ColumnID)
	require.Equal(t, "Move me", moved.Title, "MoveTask must return the task's title")
	require.False(t, moved.UpdatedAt.IsZero(), "MoveTask must refresh and return updated_at")

	require.Equal(t, 1, bcast.count(), "a successful move must broadcast exactly once")
}

// TestMoveTask_InvalidColumn_RejectedNoWriteNoBroadcast covers AC-05: moving
// a task to a column that does not exist leaves the task in its previous
// column, returns the domain not-found error, and does not broadcast.
func TestMoveTask_InvalidColumn_RejectedNoWriteNoBroadcast(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	existing := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: leftmostID, Title: "Stay put"}
	repo.seedTask(existing)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	invalidColumnID := uuid.Must(uuid.NewV7())
	moved, err := svc.MoveTask(context.Background(), existing.ID, invalidColumnID)

	require.ErrorIs(t, err, domain.ErrColumnNotFound)
	require.Nil(t, moved)
	stored, ok := repo.getTask(existing.ID)
	require.True(t, ok)
	require.Equal(t, leftmostID, stored.ColumnID, "task must stay in its previous column, as if the drop never happened (AC-05)")

	require.Equal(t, 0, bcast.count(), "a rejected move must not broadcast")
}

// TestDeleteTask_RemovesTask covers AC-06: deleting a task removes it from
// the repository and broadcasts once.
func TestDeleteTask_RemovesTask(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	existing := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: leftmostID, Title: "Delete me"}
	repo.seedTask(existing)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast, boardID)

	err := svc.DeleteTask(context.Background(), existing.ID)

	require.NoError(t, err)
	_, ok := repo.getTask(existing.ID)
	require.False(t, ok, "task must be gone from the board after delete (AC-06)")

	require.Equal(t, 1, bcast.count(), "a successful delete must broadcast exactly once")
}
