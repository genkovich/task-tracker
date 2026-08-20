//go:build integration

package app_test

// RED for T12 (docs/features/board/tasks/T12-integration-tests-invariants.md):
// integration tests for the two invariants left uncovered by T5/T6/T9's own
// integration suites:
//
//   - AC-05b: two concurrent MoveTask calls for the same task, to different
//     columns, must converge on exactly one column — whichever write landed
//     last at the database — with no lost update and no torn/ambiguous
//     state; every subsequent read must agree on that single column.
//   - AC-11: a viewer holding a live SSE connection (represented here at the
//     Hub/LinkService seam, one layer below T9's HTTP-level SSE test) must
//     have that connection closed synchronously by RevokePublicLink — by the
//     time RevokePublicLink returns, not merely "eventually".
//
// Both app.TaskService/app.LinkService and their infra dependencies
// (infra.PostgresRepository, infra.Hub) already exist (T1-T6, T8-T10) — this
// suite exercises the already-wired seam directly rather than waiting on new
// production code, per T12's DoD. It is still expected to prove its own
// worth: if either assertion below is too weak to catch a regression (e.g.
// a lost update, or an async/eventual close), that is the signal to
// strengthen it before treating this as a real RED/GREEN gate.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

// seeded board/columns from migrations 000007/000009 (data-model.md) —
// matches infra's postgres_repo_integration_test.go and ports'
// sse_handler_integration_test.go.
var (
	t12SeedBoardID        = uuid.MustParse("019a0000-0000-7000-8000-000000000101")
	t12SeedColumnToDoID   = uuid.MustParse("019a0000-0000-7000-8000-000000000201") // position 0
	t12SeedColumnInProgID = uuid.MustParse("019a0000-0000-7000-8000-000000000202") // position 1
	t12SeedColumnDoneID   = uuid.MustParse("019a0000-0000-7000-8000-000000000203") // position 2
)

// t12Fixture bundles the real Postgres-backed repo, the real in-process Hub,
// and the two app services under test.
type t12Fixture struct {
	taskSvc *app.TaskService
	linkSvc *app.LinkService
	hub     *infra.Hub
	db      *database.DB
}

func setupT12(t *testing.T) t12Fixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()

	return t12Fixture{
		taskSvc: app.NewTaskService(repo, hub, t12SeedBoardID),
		linkSvc: app.NewLinkService(repo, hub),
		hub:     hub,
		db:      c.DB,
	}
}

// t12TaskColumnID reads a task's current column_id directly off the DB,
// independent of any app-layer read path, so the assertion is against
// durable state rather than an in-process cache.
func t12TaskColumnID(t *testing.T, f t12Fixture, taskID uuid.UUID) uuid.UUID {
	t.Helper()

	var columnID uuid.UUID
	err := f.db.QueryRow(context.Background(),
		`SELECT column_id FROM tasks WHERE id = $1`, taskID,
	).Scan(&columnID)
	require.NoError(t, err)
	return columnID
}

// TestConcurrentMoveTask_ConvergesOnSingleLastWrittenColumn covers AC-05b:
// two team members dragging the same task to different columns at the same
// time must not leave the task split/ambiguous — it lands in exactly one of
// the two target columns, and every subsequent read (simulated here by two
// independent DB reads after both writes complete) agrees on that same
// column.
func TestConcurrentMoveTask_ConvergesOnSingleLastWrittenColumn(t *testing.T) {
	f := setupT12(t)
	ctx := context.Background()

	task, err := f.taskSvc.CreateTask(ctx, "Concurrent move target", nil)
	require.NoError(t, err)
	require.Equal(t, t12SeedColumnToDoID, task.ColumnID, "sanity: task starts in the leftmost column")

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- f.taskSvc.MoveTask(ctx, task.ID, t12SeedColumnInProgID)
	}()
	go func() {
		defer wg.Done()
		errCh <- f.taskSvc.MoveTask(ctx, task.ID, t12SeedColumnDoneID)
	}()
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err, "AC-05b: neither concurrent move should error")
	}

	firstRead := t12TaskColumnID(t, f, task.ID)
	require.Containsf(t, []uuid.UUID{t12SeedColumnInProgID, t12SeedColumnDoneID}, firstRead,
		"AC-05b: task must land in exactly one of the two racing target columns, got %s", firstRead)

	secondRead := t12TaskColumnID(t, f, task.ID)
	require.Equalf(t, firstRead, secondRead,
		"AC-05b: every subsequent read must agree on the same single converged column (no lost update / torn state)")
}

// TestRevokePublicLink_ClosesLiveConnectionSynchronously covers AC-11's
// "already-open connection" edge case one layer below T9's HTTP-level test:
// a viewer's Hub connection registered under a token must already be closed
// by the time LinkService.RevokePublicLink returns — not merely eventually,
// and not requiring the test to poll/wait.
func TestRevokePublicLink_ClosesLiveConnectionSynchronously(t *testing.T) {
	f := setupT12(t)
	ctx := context.Background()

	link, err := f.linkSvc.IssuePublicLink(ctx, t12SeedBoardID)
	require.NoError(t, err)

	ch, unregister := f.hub.Subscribe(link.Token)
	defer unregister()

	require.NoError(t, f.linkSvc.RevokePublicLink(ctx, t12SeedBoardID))

	// No select/timeout: AC-11 requires the close to already have happened
	// by the time RevokePublicLink returns, so a non-blocking receive must
	// immediately observe the closed channel.
	select {
	case _, open := <-ch:
		require.Falsef(t, open,
			"AC-11: the viewer's connection channel must be closed (not merely drained) once RevokePublicLink returns")
	default:
		t.Fatal("AC-11: RevokePublicLink returned but the viewer's connection channel was not yet closed — the close is not synchronous")
	}

	// Give any wrongly-async close a moment, then re-check, so a flaky
	// goroutine-based close doesn't make this test pass by luck on a slow
	// machine while still being non-synchronous in practice.
	time.Sleep(50 * time.Millisecond)
	_, open := <-ch
	require.False(t, open, "AC-11: connection channel must remain closed")
}
