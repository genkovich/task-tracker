//go:build integration

package infra_test

// Integration tests for T3 — the Postgres repository backing the board
// module (ports.Repository / infra.PostgresRepository, not yet written).
// Per T3's DoD, this covers:
//   - AC-01: inserting a task without an explicit column lands in the
//     leftmost column by position.
//   - AC-04: MoveTask updates column_id for a valid target column.
//   - AC-05: moving to a non-existent column_id returns the domain
//     not-found error, no row changed.
//   - AC-06: DeleteTask hard-deletes the row.
//
// No production code exists yet for infra.NewPostgresRepository /
// infra.PostgresRepository — this is the RED step; the package will not
// compile until T3 adds them.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

// seeded board/columns from migrations 000007/000009 (data-model.md,
// docs/features/board/spec.md AC-01 "найлівіша column").
var (
	seedBoardID        = uuid.MustParse("019a0000-0000-7000-8000-000000000101")
	seedColumnToDoID   = uuid.MustParse("019a0000-0000-7000-8000-000000000201") // position 0, leftmost
	seedColumnInProgID = uuid.MustParse("019a0000-0000-7000-8000-000000000202") // position 1
)

// repoFixture bundles the repo under test with the raw *database.DB the
// container gives us, so assertions can verify durable DB state directly
// rather than through the repo's own read path.
type repoFixture struct {
	repo *infra.PostgresRepository
	db   *database.DB
}

func setupRepo(t *testing.T) repoFixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return repoFixture{
		repo: infra.NewPostgresRepository(c.DB),
		db:   c.DB,
	}
}

// taskColumnID reads back a task's current column_id directly, independent
// of the repo under test, so the assertions verify durable DB state rather
// than the repo's own read path.
func taskColumnID(t *testing.T, f repoFixture, taskID uuid.UUID) (uuid.UUID, bool) {
	t.Helper()

	var columnID uuid.UUID
	err := f.db.QueryRow(context.Background(),
		`SELECT column_id FROM tasks WHERE id = $1`, taskID,
	).Scan(&columnID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false
	}
	require.NoError(t, err)
	return columnID, true
}

// AC-01 (US-01) happy path: a task created against the leftmost column
// (resolved via LeftmostColumnID, position 0) is durably stored there.
func TestPostgresRepository_InsertTask_LandsInLeftmostColumn(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	leftmostID, err := f.repo.LeftmostColumnID(ctx, seedBoardID)
	require.NoError(t, err)
	require.Equal(t, seedColumnToDoID, leftmostID, "LeftmostColumnID should resolve to the position=0 seeded column (\"To Do\")")

	task, err := domain.NewTask(leftmostID, "Write the report", nil)
	require.NoError(t, err)

	err = f.repo.InsertTask(ctx, task)
	require.NoError(t, err)

	gotColumnID, found := taskColumnID(t, f, task.ID)
	require.True(t, found, "inserted task should be readable back from tasks")
	require.Equal(t, seedColumnToDoID, gotColumnID, "task should land in the leftmost column (AC-01)")
}

// AC-04 (US-03) happy path: MoveTask updates column_id to a valid target
// column and the change is durable.
func TestPostgresRepository_MoveTask_UpdatesColumnID(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	task, err := domain.NewTask(seedColumnToDoID, "Move me", nil)
	require.NoError(t, err)
	require.NoError(t, f.repo.InsertTask(ctx, task))

	moved, movedBoardID, err := f.repo.MoveTask(ctx, task.ID, seedColumnInProgID)
	require.NoError(t, err)

	gotColumnID, found := taskColumnID(t, f, task.ID)
	require.True(t, found)
	require.Equal(t, seedColumnInProgID, gotColumnID, "MoveTask should update column_id to the new column (AC-04)")

	// Root G pin: the returned row is complete and updated_at moved forward.
	// The reported board id feeds the board-scoped broadcast (boards BRD-05).
	require.Equal(t, seedBoardID, movedBoardID, "MoveTask must report the task's board id")
	require.NotNil(t, moved)
	require.Equal(t, task.ID, moved.ID)
	require.Equal(t, seedColumnInProgID, moved.ColumnID)
	require.Equal(t, "Move me", moved.Title)
	require.False(t, moved.CreatedAt.IsZero())
	require.False(t, moved.UpdatedAt.Before(task.UpdatedAt), "MoveTask must refresh updated_at")
}

// AC-05 (US-03) domain invariant: moving a task to a non-existent column_id
// is rejected — the domain not-found error is returned and the task's
// column is left unchanged.
func TestPostgresRepository_MoveTask_NonexistentColumn_ReturnsErrColumnNotFound(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	task, err := domain.NewTask(seedColumnToDoID, "Stay put", nil)
	require.NoError(t, err)
	require.NoError(t, f.repo.InsertTask(ctx, task))

	bogusColumnID := uuid.Must(uuid.NewV7())
	_, _, err = f.repo.MoveTask(ctx, task.ID, bogusColumnID)

	require.Error(t, err)
	require.Truef(t, errors.Is(err, domain.ErrColumnNotFound),
		"MoveTask(nonexistent column) error = %v, want errors.Is(err, domain.ErrColumnNotFound)", err)

	gotColumnID, found := taskColumnID(t, f, task.ID)
	require.True(t, found)
	require.Equal(t, seedColumnToDoID, gotColumnID, "task's column must stay unchanged when the move target does not exist (AC-05)")
}

// AC-06 (US-04) happy path: DeleteTask hard-deletes the row — matches the
// user module's existing hard-delete pattern (data-model.md).
func TestPostgresRepository_DeleteTask_HardDeletesRow(t *testing.T) {
	f := setupRepo(t)
	ctx := context.Background()

	task, err := domain.NewTask(seedColumnToDoID, "Delete me", nil)
	require.NoError(t, err)
	require.NoError(t, f.repo.InsertTask(ctx, task))

	deletedBoardID, err := f.repo.DeleteTask(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, seedBoardID, deletedBoardID, "DeleteTask must report the board the task belonged to (BRD-05)")

	_, found := taskColumnID(t, f, task.ID)
	require.False(t, found, "task row must be gone after DeleteTask (AC-06 hard delete)")
}
