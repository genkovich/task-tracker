package app_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

type fakeCardRepo struct {
	cards       map[uuid.UUID]domain.Card
	moveCalls   int
	updateCalls int
}

func newFakeCardRepo() *fakeCardRepo {
	return &fakeCardRepo{cards: map[uuid.UUID]domain.Card{}}
}

func (f *fakeCardRepo) Create(_ context.Context, c *domain.Card) error {
	f.cards[c.ID] = *c
	return nil
}

func (f *fakeCardRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Card, error) {
	c, ok := f.cards[id]
	if !ok {
		return nil, domain.ErrCardNotFound
	}
	return &c, nil
}

func (f *fakeCardRepo) List(_ context.Context) ([]domain.Card, error) {
	out := make([]domain.Card, 0, len(f.cards))
	for _, c := range f.cards {
		out = append(out, c)
	}
	return out, nil
}

// Update mirrors the real repository's partial-update shape: only
// name/assignee change, column_status is untouched.
func (f *fakeCardRepo) Update(_ context.Context, c *domain.Card) error {
	f.updateCalls++
	existing, ok := f.cards[c.ID]
	if !ok {
		return domain.ErrCardNotFound
	}
	existing.Name = c.Name
	existing.Assignee = c.Assignee
	f.cards[c.ID] = existing
	*c = existing
	return nil
}

func (f *fakeCardRepo) Move(_ context.Context, id uuid.UUID, columnStatus string) (*domain.Card, error) {
	f.moveCalls++
	c, ok := f.cards[id]
	if !ok {
		return nil, domain.ErrCardNotFound
	}
	c.ColumnStatus = columnStatus
	f.cards[id] = c
	return &c, nil
}

func (f *fakeCardRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.cards[id]; !ok {
		return domain.ErrCardNotFound
	}
	delete(f.cards, id)
	return nil
}

func TestCardService_CreateCard_Rejects_EmptyName(t *testing.T) {
	svc := app.NewCardService(newFakeCardRepo())
	_, err := svc.CreateCard(context.Background(), "   ", nil)
	require.ErrorIs(t, err, domain.ErrNameRequired)
}

func TestCardService_CreateCard_LandsInTodo(t *testing.T) {
	svc := app.NewCardService(newFakeCardRepo())
	card, err := svc.CreateCard(context.Background(), "Write the deck", nil)
	require.NoError(t, err)
	require.Equal(t, domain.ColumnTodo, card.ColumnStatus)
}

func TestCardService_UpdateCard_Rejects_EmptyName(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	created, err := svc.CreateCard(context.Background(), "original", nil)
	require.NoError(t, err)

	_, err = svc.UpdateCard(context.Background(), created.ID, "", nil)
	require.ErrorIs(t, err, domain.ErrNameRequired)
}

// TestCardService_UpdateCard_NoVersionCheck proves ADR-0002: the service
// performs an unconditional overwrite — no read-current-then-compare step
// that could reject a write.
func TestCardService_UpdateCard_NoVersionCheck(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	created, err := svc.CreateCard(context.Background(), "original", nil)
	require.NoError(t, err)

	_, err = svc.UpdateCard(context.Background(), created.ID, "renamed", nil)
	require.NoError(t, err)
	require.Equal(t, 1, repo.updateCalls)
}

func TestCardService_MoveCard_Rejects_InvalidColumn(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	created, err := svc.CreateCard(context.Background(), "movable", nil)
	require.NoError(t, err)

	_, err = svc.MoveCard(context.Background(), created.ID, "backlog")
	require.ErrorIs(t, err, domain.ErrInvalidColumn)
}

func TestCardService_MoveCard_OK(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	created, err := svc.CreateCard(context.Background(), "movable", nil)
	require.NoError(t, err)

	moved, err := svc.MoveCard(context.Background(), created.ID, domain.ColumnInProgress)
	require.NoError(t, err)
	require.Equal(t, domain.ColumnInProgress, moved.ColumnStatus)
}

// TestCardService_MoveCard_ConcurrentDeleteWins covers AC-15 at the service
// layer: moving an already-deleted card is a no-op signaled by
// ErrCardNotFound, not a special-cased success.
func TestCardService_MoveCard_ConcurrentDeleteWins(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	created, err := svc.CreateCard(context.Background(), "deleted mid-flight", nil)
	require.NoError(t, err)

	require.NoError(t, svc.DeleteCard(context.Background(), created.ID))

	_, err = svc.MoveCard(context.Background(), created.ID, domain.ColumnDone)
	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

func TestCardService_GetBoard(t *testing.T) {
	repo := newFakeCardRepo()
	svc := app.NewCardService(repo)
	_, err := svc.CreateCard(context.Background(), "one", nil)
	require.NoError(t, err)
	_, err = svc.CreateCard(context.Background(), "two", nil)
	require.NoError(t, err)

	cards, err := svc.GetBoard(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 2)
}
