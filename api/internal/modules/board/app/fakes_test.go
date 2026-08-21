package app

// Hand-written fakes for the app layer's consumer-side ports (ports.Repository,
// ports.Broadcaster) — repo convention (go-tests.md "hand-written fakes over a
// mock framework", mirroring internal/modules/courses/app/service_test.go).
// No production code exists yet for app.LinkService / app.StateService — this
// is the RED step for T6.

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// fakeRepo implements ports.Repository in-memory, exercising only the
// public-link/board-state behavior T6 owns. Every other method returns an
// explicit "not implemented" error so an accidental call in these tests fails
// loudly instead of silently succeeding.
type fakeRepo struct {
	states       map[uuid.UUID]*ports.BoardState
	linksByBoard map[uuid.UUID]*domain.PublicLink
	linksByToken map[string]*domain.PublicLink
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		states:       make(map[uuid.UUID]*ports.BoardState),
		linksByBoard: make(map[uuid.UUID]*domain.PublicLink),
		linksByToken: make(map[string]*domain.PublicLink),
	}
}

func (r *fakeRepo) GetBoardState(_ context.Context, boardID uuid.UUID) (*ports.BoardState, error) {
	st, ok := r.states[boardID]
	if !ok {
		return nil, errors.New("fakeRepo: no board state seeded for boardID")
	}
	return st, nil
}

func (r *fakeRepo) LeftmostColumnID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, errors.New("fakeRepo: LeftmostColumnID not implemented")
}

func (r *fakeRepo) InsertTask(context.Context, *domain.Task) error {
	return errors.New("fakeRepo: InsertTask not implemented")
}

func (r *fakeRepo) UpdateTask(context.Context, *domain.Task) error {
	return errors.New("fakeRepo: UpdateTask not implemented")
}

func (r *fakeRepo) MoveTask(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error) {
	return nil, errors.New("fakeRepo: MoveTask not implemented")
}

func (r *fakeRepo) DeleteTask(context.Context, uuid.UUID) error {
	return errors.New("fakeRepo: DeleteTask not implemented")
}

func (r *fakeRepo) ColumnExists(context.Context, uuid.UUID) (bool, error) {
	return false, errors.New("fakeRepo: ColumnExists not implemented")
}

// IssuePublicLink mirrors the real repo's contract (data-model.md UNIQUE
// (board_id)): domain.ErrLinkAlreadyActive when the board already has a link,
// no write on that path.
func (r *fakeRepo) IssuePublicLink(_ context.Context, link *domain.PublicLink) error {
	if _, active := r.linksByBoard[link.BoardID]; active {
		return domain.ErrLinkAlreadyActive
	}
	r.linksByBoard[link.BoardID] = link
	r.linksByToken[link.Token] = link
	return nil
}

// RevokePublicLink hard-deletes the board's active link, domain.ErrLinkNotFound
// if there is none.
func (r *fakeRepo) RevokePublicLink(_ context.Context, boardID uuid.UUID) error {
	link, ok := r.linksByBoard[boardID]
	if !ok {
		return domain.ErrLinkNotFound
	}
	delete(r.linksByBoard, boardID)
	delete(r.linksByToken, link.Token)
	return nil
}

// PublicLinkByToken looks up by opaque token, domain.ErrLinkNotFound for an
// unknown/revoked one (AC-11).
func (r *fakeRepo) PublicLinkByToken(_ context.Context, token string) (*domain.PublicLink, error) {
	link, ok := r.linksByToken[token]
	if !ok {
		return nil, domain.ErrLinkNotFound
	}
	return link, nil
}

// PublicLinkByBoard looks up a board's active public link, domain.ErrLinkNotFound
// if there is none — added alongside link_service.go so RevokePublicLink can
// learn which token to close before deleting the row (AC-08).
func (r *fakeRepo) PublicLinkByBoard(_ context.Context, boardID uuid.UUID) (*domain.PublicLink, error) {
	link, ok := r.linksByBoard[boardID]
	if !ok {
		return nil, domain.ErrLinkNotFound
	}
	return link, nil
}

var _ ports.Repository = (*fakeRepo)(nil)

// fakeBroadcaster records CloseToken/Broadcast calls so tests can assert the
// use-case notified live connections (AC-08, AC-11, sad.md §6 Flow 3).
type fakeBroadcaster struct {
	closedTokens []string
	broadcasts   []ports.Event
}

func (b *fakeBroadcaster) Broadcast(evt ports.Event) {
	b.broadcasts = append(b.broadcasts, evt)
}

func (b *fakeBroadcaster) CloseToken(token string) {
	b.closedTokens = append(b.closedTokens, token)
}

var _ ports.Broadcaster = (*fakeBroadcaster)(nil)
