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

func setupLinkRepo(t *testing.T) *infra.PostgresPublicLinkRepository {
	t.Helper()
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	require.NoError(t, database.RunMigrations(migrations.FS, c.DSN))
	return infra.NewPostgresPublicLinkRepository(c.DB)
}

func TestPublicLinkRepository_Generate(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	link, err := repo.Generate(ctx, "tok-1")
	require.NoError(t, err)
	require.Equal(t, "tok-1", link.Token)
	require.True(t, link.Active())
}

func TestPublicLinkRepository_Generate_DisablesPriorActive(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	first, err := repo.Generate(ctx, "tok-first")
	require.NoError(t, err)

	second, err := repo.Generate(ctx, "tok-second")
	require.NoError(t, err)
	require.True(t, second.Active())

	// The prior link must now be disabled — never more than one active link.
	stale, err := repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.False(t, stale.Active())
}

func TestPublicLinkRepository_ResolveByToken_Active(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	link, err := repo.Generate(ctx, "tok-resolve")
	require.NoError(t, err)

	got, err := repo.ResolveByToken(ctx, "tok-resolve")
	require.NoError(t, err)
	require.Equal(t, link.ID, got.ID)
}

func TestPublicLinkRepository_ResolveByToken_DisabledOrUnknown(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	link, err := repo.Generate(ctx, "tok-disable-me")
	require.NoError(t, err)
	require.NoError(t, repo.Disable(ctx, link.ID))

	_, err = repo.ResolveByToken(ctx, "tok-disable-me")
	require.ErrorIs(t, err, domain.ErrLinkNotFound)

	_, err = repo.ResolveByToken(ctx, "never-existed")
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}

func TestPublicLinkRepository_GetActive_None(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	got, err := repo.GetActive(ctx)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestPublicLinkRepository_GetActive_Present(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	link, err := repo.Generate(ctx, "tok-active")
	require.NoError(t, err)

	got, err := repo.GetActive(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, link.ID, got.ID)
}

func TestPublicLinkRepository_Disable_Idempotent(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	link, err := repo.Generate(ctx, "tok-idem")
	require.NoError(t, err)

	require.NoError(t, repo.Disable(ctx, link.ID))
	require.NoError(t, repo.Disable(ctx, link.ID)) // second call: no error, stays disabled

	got, err := repo.GetByID(ctx, link.ID)
	require.NoError(t, err)
	require.False(t, got.Active())
}

func TestPublicLinkRepository_Disable_NotFound(t *testing.T) {
	repo := setupLinkRepo(t)
	ctx := context.Background()

	missingID, _ := uuid.NewV7()
	err := repo.Disable(ctx, missingID)
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}
