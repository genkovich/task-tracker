//go:build integration

package infra_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/infra"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

func setupCardRepo(t *testing.T) *infra.PostgresCardRepository {
	t.Helper()
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	require.NoError(t, database.RunMigrations(migrations.FS, c.DSN))
	return infra.NewPostgresCardRepository(c.DB)
}

func newTestCard(name, columnStatus string) *domain.Card {
	id, _ := uuid.NewV7()
	return &domain.Card{ID: id, Name: name, ColumnStatus: columnStatus}
}

func TestCardRepository_CreateAndList(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c1 := newTestCard("first", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c1))
	require.False(t, c1.CreatedAt.IsZero())
	require.False(t, c1.UpdatedAt.IsZero())

	c2 := newTestCard("second", domain.ColumnInProgress)
	require.NoError(t, repo.Create(ctx, c2))

	cards, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, cards, 2)
}

func TestCardRepository_Update(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c := newTestCard("original", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c))

	assignee := "Test User"
	c.Name = "renamed"
	c.Assignee = &assignee
	require.NoError(t, repo.Update(ctx, c))

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Name)
	require.Equal(t, "Test User", *got.Assignee)
}

func TestCardRepository_Update_NotFound(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	missing := newTestCard("ghost", domain.ColumnTodo)
	err := repo.Update(ctx, missing)
	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

func TestCardRepository_Move(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c := newTestCard("movable", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c))
	firstUpdatedAt := c.UpdatedAt

	moved, err := repo.Move(ctx, c.ID, domain.ColumnDone)
	require.NoError(t, err)
	require.Equal(t, domain.ColumnDone, moved.ColumnStatus)
	require.True(t, moved.UpdatedAt.After(firstUpdatedAt) || moved.UpdatedAt.Equal(firstUpdatedAt))
}

func TestCardRepository_Move_NotFound(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	missingID, _ := uuid.NewV7()
	_, err := repo.Move(ctx, missingID, domain.ColumnDone)
	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

func TestCardRepository_Move_LastWriteWins(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c := newTestCard("race target", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c))

	// Simulate two near-simultaneous moves: the second call's write is the
	// last one the server processes, so it must be the state that persists —
	// no version check, no rejection (ADR-0002).
	_, err := repo.Move(ctx, c.ID, domain.ColumnInProgress)
	require.NoError(t, err)
	final, err := repo.Move(ctx, c.ID, domain.ColumnDone)
	require.NoError(t, err)
	require.Equal(t, domain.ColumnDone, final.ColumnStatus)

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ColumnDone, got.ColumnStatus)
}

func TestCardRepository_Delete(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c := newTestCard("to delete", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c))

	require.NoError(t, repo.Delete(ctx, c.ID))

	_, err := repo.GetByID(ctx, c.ID)
	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

func TestCardRepository_Delete_NotFound(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	missingID, _ := uuid.NewV7()
	err := repo.Delete(ctx, missingID)
	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

// TestCardRepository_Move_ConcurrentDeleteWins covers AC-15: a move that
// arrives after a delete on the same card finds nothing to update and
// changes nothing on the board.
func TestCardRepository_Move_ConcurrentDeleteWins(t *testing.T) {
	repo := setupCardRepo(t)
	ctx := context.Background()

	c := newTestCard("deleted mid-flight", domain.ColumnTodo)
	require.NoError(t, repo.Create(ctx, c))

	require.NoError(t, repo.Delete(ctx, c.ID))

	_, err := repo.Move(ctx, c.ID, domain.ColumnDone)
	require.ErrorIs(t, err, domain.ErrCardNotFound)

	cards, err := repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, cards)
}
