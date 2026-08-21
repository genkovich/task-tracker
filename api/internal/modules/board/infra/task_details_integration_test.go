//go:build integration

package infra_test

// Integration tests for the tasks feature's persistence (T3): task details
// round-tripping, the card-level fields the board state carries instead of a
// description body, and the comment thread — including the cascade that
// TSK-11 rests on, which lives in the schema rather than in Go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// insertDetailedTask stores a task with the given details in the seeded
// leftmost column and returns it.
func insertDetailedTask(t *testing.T, f repoFixture, details domain.TaskDetails) *domain.Task {
	t.Helper()

	task, err := domain.NewTask(seedColumnToDoID, details)
	require.NoError(t, err)
	require.NoError(t, f.repo.InsertTask(context.Background(), task))
	return task
}

// findListItem locates a task in a board state's columns.
func findListItem(state *ports.BoardState, taskID uuid.UUID) (ports.TaskListItem, bool) {
	for _, col := range state.Columns {
		for _, item := range col.Tasks {
			if item.ID == taskID {
				return item, true
			}
		}
	}
	return ports.TaskListItem{}, false
}

// TSK-01/TSK-03/TSK-05: description, priority and due date survive a write
// and come back through TaskByID — which also reports the task's board, the
// fact TSK-13's authorization check is built on.
func TestPostgresRepository_TaskByID_RoundTripsDetails(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	due := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	assignee := "Ada"
	task := insertDetailedTask(t, f, domain.TaskDetails{
		Title:       "Підняти стенд",
		Assignee:    &assignee,
		Description: "Розгорнути на VPS, перевірити TLS",
		Priority:    string(domain.PriorityHigh),
		DueDate:     &due,
	})

	got, boardID, err := f.repo.TaskByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "Розгорнути на VPS, перевірити TLS", got.Description)
	require.Equal(t, domain.PriorityHigh, got.Priority)
	require.NotNil(t, got.DueDate)
	require.Equal(t, due.Format("2006-01-02"), got.DueDate.Format("2006-01-02"))
	require.Equal(t, seedBoardID, boardID, "TaskByID must report which board the task sits on (TSK-13)")
}

// A task created without details takes the schema defaults, not NULLs — the
// card always has a priority to draw and a description that is simply empty.
func TestPostgresRepository_InsertTask_AppliesDetailDefaults(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	task := insertDetailedTask(t, f, domain.TaskDetails{Title: "Bare task"})

	got, _, err := f.repo.TaskByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "", got.Description)
	require.Equal(t, domain.PriorityMedium, got.Priority)
	require.Nil(t, got.DueDate)
}

func TestPostgresRepository_TaskByID_UnknownTask_ReturnsTaskNotFound(t *testing.T) {
	f := setupRepo(t)

	_, _, err := f.repo.TaskByID(context.Background(), uuid.Must(uuid.NewV7()))
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// TSK-01/TSK-08 + spec §6: the board state carries the two derived card
// fields and never the description body — the single most load-bearing
// property of this feature's read path, since a board refetch runs on every
// SSE event.
func TestPostgresRepository_GetBoardState_CarriesCardFieldsNotDescription(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	due := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	described := insertDetailedTask(t, f, domain.TaskDetails{
		Title:       "Described",
		Description: "a long body nobody wants on every refetch",
		Priority:    string(domain.PriorityLow),
		DueDate:     &due,
	})
	bare := insertDetailedTask(t, f, domain.TaskDetails{Title: "Bare"})

	for range 2 {
		comment, err := domain.NewComment(described.ID, "Ada", "note")
		require.NoError(t, err)
		_, err = f.repo.InsertComment(ctx, comment)
		require.NoError(t, err)
	}

	state, err := f.repo.GetBoardState(ctx, seedBoardID)
	require.NoError(t, err)

	describedItem, ok := findListItem(state, described.ID)
	require.True(t, ok)
	require.True(t, describedItem.HasDescription, "a task with a description must be flagged (TSK-01)")
	require.Equal(t, 2, describedItem.CommentCount, "the card counts its comments (TSK-08)")
	require.Equal(t, domain.PriorityLow, describedItem.Priority)
	require.NotNil(t, describedItem.DueDate)

	bareItem, ok := findListItem(state, bare.ID)
	require.True(t, ok)
	require.False(t, bareItem.HasDescription)
	require.Equal(t, 0, bareItem.CommentCount, "a task without comments counts zero, not null")
	require.Nil(t, bareItem.DueDate)
}

// TSK-08: comments insert and read back oldest first.
func TestPostgresRepository_Comments_InsertAndListOldestFirst(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()
	task := insertDetailedTask(t, f, domain.TaskDetails{Title: "Discuss me"})

	for _, body := range []string{"first", "second", "third"} {
		comment, err := domain.NewComment(task.ID, "Ada", body)
		require.NoError(t, err)
		boardID, err := f.repo.InsertComment(ctx, comment)
		require.NoError(t, err)
		require.Equal(t, seedBoardID, boardID, "InsertComment must report the task's board for the broadcast")
		require.False(t, comment.CreatedAt.IsZero(), "the stored comment must carry created_at")
	}

	comments, err := f.repo.ListComments(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, comments, 3)
	require.Equal(t, []string{"first", "second", "third"},
		[]string{comments[0].Body, comments[1].Body, comments[2].Body},
		"comments must read oldest first (TSK-08)")
}

// A comment on a task that does not exist is the caller's mistake, mapped to
// the task not-found sentinel — never a raw FK violation surfacing as a 500.
func TestPostgresRepository_InsertComment_UnknownTask_ReturnsTaskNotFound(t *testing.T) {
	f := setupRepo(t)

	comment, err := domain.NewComment(uuid.Must(uuid.NewV7()), "Ada", "orphan")
	require.NoError(t, err)

	_, err = f.repo.InsertComment(context.Background(), comment)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// TSK-10: deleting a comment removes it and reports its board.
func TestPostgresRepository_DeleteComment(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()
	task := insertDetailedTask(t, f, domain.TaskDetails{Title: "Discuss me"})

	comment, err := domain.NewComment(task.ID, "Ada", "bye")
	require.NoError(t, err)
	_, err = f.repo.InsertComment(ctx, comment)
	require.NoError(t, err)

	// A comment is only reachable through its own task's path: the route
	// names both ids, so the pair is the key.
	other := insertDetailedTask(t, f, domain.TaskDetails{Title: "Unrelated"})
	_, err = f.repo.DeleteComment(ctx, other.ID, comment.ID)
	require.ErrorIs(t, err, domain.ErrCommentNotFound, "a comment must not be deletable through another task")

	boardID, err := f.repo.DeleteComment(ctx, task.ID, comment.ID)
	require.NoError(t, err)
	require.Equal(t, seedBoardID, boardID)

	comments, err := f.repo.ListComments(ctx, task.ID)
	require.NoError(t, err)
	require.Empty(t, comments)

	_, err = f.repo.DeleteComment(ctx, task.ID, comment.ID)
	require.ErrorIs(t, err, domain.ErrCommentNotFound, "deleting an already-deleted comment is a 404")
}

// TSK-11: deleting a task takes its comments with it. The cascade lives in
// the schema, so this is the only place it can be proven — no Go code runs
// between the delete and the orphans.
func TestPostgresRepository_DeleteTask_CascadesComments(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()
	task := insertDetailedTask(t, f, domain.TaskDetails{Title: "Doomed"})

	comment, err := domain.NewComment(task.ID, "Ada", "will vanish with the task")
	require.NoError(t, err)
	_, err = f.repo.InsertComment(ctx, comment)
	require.NoError(t, err)

	_, err = f.repo.DeleteTask(ctx, task.ID)
	require.NoError(t, err)

	var orphans int
	require.NoError(t, f.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM task_comments WHERE task_id = $1`, task.ID,
	).Scan(&orphans))
	require.Zero(t, orphans, "a deleted task must not leave orphaned comments (TSK-11)")
}

// TSK-01/TSK-03/TSK-05/TSK-07: an edit rewrites every detail field, and
// leaving the due date out clears it.
func TestPostgresRepository_UpdateTask_PersistsDetailsAndClearsDueDate(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	due := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	task := insertDetailedTask(t, f, domain.TaskDetails{
		Title:       "Before",
		Description: "old body",
		Priority:    string(domain.PriorityLow),
		DueDate:     &due,
	})

	edited := &domain.Task{ID: task.ID}
	require.NoError(t, edited.SetDetails(domain.TaskDetails{
		Title:       "After",
		Description: "new body",
		Priority:    string(domain.PriorityHigh),
	}))
	boardID, err := f.repo.UpdateTask(ctx, edited)
	require.NoError(t, err)
	require.Equal(t, seedBoardID, boardID)

	got, _, err := f.repo.TaskByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "After", got.Title)
	require.Equal(t, "new body", got.Description)
	require.Equal(t, domain.PriorityHigh, got.Priority)
	require.Nil(t, got.DueDate, "an edit without a due date clears it (TSK-07)")
}

// The schema's CHECK is the second line behind the domain (data-model.md):
// a priority that never passed through ParsePriority must still be refused
// by the database.
func TestPostgresRepository_PriorityCheckConstraint_RefusesUnknownValue(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()
	task := insertDetailedTask(t, f, domain.TaskDetails{Title: "Guarded"})

	_, err := f.db.Exec(ctx, `UPDATE tasks SET priority = 'urgent' WHERE id = $1`, task.ID)
	require.Error(t, err, "the priority CHECK must refuse a value outside low/medium/high")
}
