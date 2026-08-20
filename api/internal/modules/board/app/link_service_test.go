package app

// RED for T6 (docs/features/board/tasks/T6-app-link-state-usecases.md):
// api/internal/modules/board/app/link_service.go does not exist yet.
// Covers AC-07 (issue when none active), the 409 already-active precondition
// (board.link_already_active, contracts/openapi.yaml), and AC-08/AC-11
// (revoke deletes the link and closes its SSE token via Broadcaster).

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// TestLinkService_IssuePublicLink_NoneActive_Succeeds covers AC-07: a board
// with no active public link gets one issued, with a non-empty opaque token
// (ADR-0003) persisted for that board.
func TestLinkService_IssuePublicLink_NoneActive_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	broadcaster := &fakeBroadcaster{}
	svc := NewLinkService(repo, broadcaster)
	boardID := uuid.Must(uuid.NewV7())

	got, err := svc.IssuePublicLink(context.Background(), boardID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, boardID, got.BoardID)
	require.NotEmpty(t, got.Token, "IssuePublicLink() token is empty, want an opaque generated token (ADR-0003)")

	stored, ok := repo.linksByBoard[boardID]
	require.True(t, ok, "IssuePublicLink() did not persist a link for boardID")
	require.Equal(t, got.Token, stored.Token)
}

// TestLinkService_IssuePublicLink_AlreadyActive_RejectsSecondLink covers the
// 409 case: a board that already has an active link rejects a second
// IssuePublicLink call and writes nothing new (board.link_already_active,
// openapi.yaml).
func TestLinkService_IssuePublicLink_AlreadyActive_RejectsSecondLink(t *testing.T) {
	repo := newFakeRepo()
	broadcaster := &fakeBroadcaster{}
	svc := NewLinkService(repo, broadcaster)
	boardID := uuid.Must(uuid.NewV7())

	existing := &domain.PublicLink{ID: uuid.Must(uuid.NewV7()), BoardID: boardID, Token: "existing-token"}
	repo.linksByBoard[boardID] = existing
	repo.linksByToken[existing.Token] = existing

	got, err := svc.IssuePublicLink(context.Background(), boardID)

	require.Nil(t, got, "IssuePublicLink() on an already-active board returned a link, want nil")
	require.Truef(t, errors.Is(err, domain.ErrLinkAlreadyActive),
		"IssuePublicLink() error = %v, want errors.Is(err, domain.ErrLinkAlreadyActive)", err)
	require.Len(t, repo.linksByBoard, 1, "IssuePublicLink() on an already-active board wrote a second link row")
	require.Equal(t, existing.Token, repo.linksByBoard[boardID].Token, "the pre-existing link's token was overwritten")
}

// TestLinkService_RevokePublicLink_Active_DeletesAndClosesToken covers
// AC-08/AC-11: revoking the active link removes it (a subsequent lookup by
// its token fails) and closes any already-open SSE connections for that
// token via Broadcaster.CloseToken (sad.md §6 Flow 3).
func TestLinkService_RevokePublicLink_Active_DeletesAndClosesToken(t *testing.T) {
	repo := newFakeRepo()
	broadcaster := &fakeBroadcaster{}
	svc := NewLinkService(repo, broadcaster)
	boardID := uuid.Must(uuid.NewV7())

	existing := &domain.PublicLink{ID: uuid.Must(uuid.NewV7()), BoardID: boardID, Token: "active-token"}
	repo.linksByBoard[boardID] = existing
	repo.linksByToken[existing.Token] = existing

	err := svc.RevokePublicLink(context.Background(), boardID)

	require.NoError(t, err)
	_, stillFound := repo.linksByBoard[boardID]
	require.False(t, stillFound, "RevokePublicLink() left the link row in place")
	require.Equal(t, []string{"active-token"}, broadcaster.closedTokens,
		"RevokePublicLink() did not close the revoked token's SSE connections")
}
