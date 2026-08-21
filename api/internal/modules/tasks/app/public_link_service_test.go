package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

var knownTime = time.Now()

type fakeLinkRepo struct {
	links          map[uuid.UUID]domain.PublicLink
	generateCalled int
}

func newFakeLinkRepo() *fakeLinkRepo {
	return &fakeLinkRepo{links: map[uuid.UUID]domain.PublicLink{}}
}

func (f *fakeLinkRepo) Generate(_ context.Context, token string) (*domain.PublicLink, *uuid.UUID, error) {
	f.generateCalled++
	var replacedID *uuid.UUID
	for id, l := range f.links {
		if l.Active() {
			l.DisabledAt = &knownTime
			f.links[id] = l
			replaced := id
			replacedID = &replaced
		}
	}
	id, _ := uuid.NewV7()
	link := domain.PublicLink{ID: id, Token: token}
	f.links[id] = link
	return &link, replacedID, nil
}

func (f *fakeLinkRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.PublicLink, error) {
	l, ok := f.links[id]
	if !ok {
		return nil, domain.ErrLinkNotFound
	}
	return &l, nil
}

func (f *fakeLinkRepo) GetActive(_ context.Context) (*domain.PublicLink, error) {
	for _, l := range f.links {
		if l.Active() {
			return &l, nil
		}
	}
	return nil, nil
}

func (f *fakeLinkRepo) ResolveByToken(_ context.Context, token string) (*domain.PublicLink, error) {
	for _, l := range f.links {
		if l.Token == token && l.Active() {
			return &l, nil
		}
	}
	return nil, domain.ErrLinkNotFound
}

func (f *fakeLinkRepo) Disable(_ context.Context, id uuid.UUID) error {
	l, ok := f.links[id]
	if !ok {
		return domain.ErrLinkNotFound
	}
	l.DisabledAt = &knownTime
	f.links[id] = l
	return nil
}

func TestPublicLinkService_GenerateLink_BroadcastsEvent(t *testing.T) {
	repo := newFakeLinkRepo()
	b := app.NewBroadcaster()
	svc := app.NewPublicLinkService(repo, b)

	ch, unsub := b.Subscribe()
	defer unsub()

	link, err := svc.GenerateLink(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, link.Token)
	require.Equal(t, 1, repo.generateCalled)

	ev := <-ch
	require.Equal(t, app.EventLinkGenerated, ev.Type)
}

func TestPublicLinkService_DisableLink_BroadcastsEvent(t *testing.T) {
	repo := newFakeLinkRepo()
	b := app.NewBroadcaster()
	svc := app.NewPublicLinkService(repo, b)

	link, err := svc.GenerateLink(context.Background())
	require.NoError(t, err)

	ch, unsub := b.Subscribe()
	defer unsub()

	require.NoError(t, svc.DisableLink(context.Background(), link.ID))

	ev := <-ch
	require.Equal(t, app.EventLinkDisabled, ev.Type)
}

// TestPublicLinkService_GenerateLink_RevokesReplacedLink proves a second
// "Get link" click doesn't leave the first link's SSE stream authorized
// indefinitely: a link.disabled event fires for the replaced link too.
func TestPublicLinkService_GenerateLink_RevokesReplacedLink(t *testing.T) {
	repo := newFakeLinkRepo()
	b := app.NewBroadcaster()
	svc := app.NewPublicLinkService(repo, b)

	first, err := svc.GenerateLink(context.Background())
	require.NoError(t, err)

	ch, unsub := b.Subscribe()
	defer unsub()

	_, err = svc.GenerateLink(context.Background())
	require.NoError(t, err)

	disabledEv := <-ch
	require.Equal(t, app.EventLinkDisabled, disabledEv.Type)
	require.Equal(t, first.ID, disabledEv.Payload)

	generatedEv := <-ch
	require.Equal(t, app.EventLinkGenerated, generatedEv.Type)
}

func TestPublicLinkService_GetActiveLink_NoneIsNilNotError(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := app.NewPublicLinkService(repo, app.NewBroadcaster())

	got, err := svc.GetActiveLink(context.Background())
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestPublicLinkService_ResolvePublicLink(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := app.NewPublicLinkService(repo, app.NewBroadcaster())

	link, err := svc.GenerateLink(context.Background())
	require.NoError(t, err)

	got, err := svc.ResolvePublicLink(context.Background(), link.Token)
	require.NoError(t, err)
	require.Equal(t, link.ID, got.ID)

	_, err = svc.ResolvePublicLink(context.Background(), "never-existed")
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}
