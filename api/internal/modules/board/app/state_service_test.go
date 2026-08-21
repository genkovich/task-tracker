package app

// RED for T6 (docs/features/board/tasks/T6-app-link-state-usecases.md):
// api/internal/modules/board/app/state_service.go does not exist yet.
// Covers AC-09 (viewer fetches board state by a valid public-link token) and
// AC-11 (an unknown/revoked token is rejected, not shown a stale state).
//
// GetPublicBoardState returns ports.PublicBoardState — the viewer shape from
// contracts/openapi.yaml (columns only, no public_link field at all, unlike
// ports.BoardState) — per T6's own note: "must not expose team-editor-only
// fields (e.g. no second link)". ports.PublicBoardState does not exist yet
// either; that is expected to land alongside state_service.go.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// TestStateService_GetPublicBoardState_ValidToken_ReturnsBoardState covers
// AC-09: a viewer holding a valid public-link token sees the board's current
// columns and tasks.
func TestStateService_GetPublicBoardState_ValidToken_ReturnsBoardState(t *testing.T) {
	repo := newFakeRepo()
	svc := NewStateService(repo)

	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	link := &domain.PublicLink{ID: uuid.Must(uuid.NewV7()), BoardID: boardID, Token: "valid-token"}
	repo.linksByBoard[boardID] = link
	repo.linksByToken[link.Token] = link

	task := ports.TaskListItem{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Write the report"}
	column := domain.Column{ID: columnID, BoardID: boardID, Name: "To Do", Position: 0}
	repo.states[boardID] = &ports.BoardState{
		Columns:    []ports.ColumnState{{Column: column, Tasks: []ports.TaskListItem{task}}},
		PublicLink: link,
	}

	got, err := svc.GetPublicBoardState(context.Background(), "valid-token")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Columns, 1)
	require.Equal(t, column.ID, got.Columns[0].ID)
	require.Len(t, got.Columns[0].Tasks, 1)
	require.Equal(t, task.Title, got.Columns[0].Tasks[0].Title)
}

// TestStateService_GetPublicBoardState_UnknownToken_ReturnsErrLinkNotFound
// covers AC-11: a revoked or never-issued token must not show any board
// state — the not-found sentinel, not a stale/cached view.
func TestStateService_GetPublicBoardState_UnknownToken_ReturnsErrLinkNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewStateService(repo)

	got, err := svc.GetPublicBoardState(context.Background(), "revoked-or-unknown-token")

	require.Nil(t, got, "GetPublicBoardState(unknown token) returned a state, want nil")
	require.ErrorIsf(t, err, domain.ErrLinkNotFound,
		"GetPublicBoardState(unknown token) error = %v, want errors.Is(err, domain.ErrLinkNotFound)", err)
}
