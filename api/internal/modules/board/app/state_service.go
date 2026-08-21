package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// StateService implements the board-state read use-cases: the team-editor
// view (GetBoardState) and the read-only viewer view resolved by public-link
// token (GetPublicBoardState).
type StateService struct {
	repo ports.Repository
}

// NewStateService wires a StateService against the given repository.
func NewStateService(repo ports.Repository) *StateService {
	return &StateService{repo: repo}
}

// GetBoardState returns the team-editor view of boardID: every column with
// its tasks, plus the board's current public link (or none, SCR-01/SCR-04).
func (s *StateService) GetBoardState(ctx context.Context, boardID uuid.UUID) (*ports.BoardState, error) {
	state, err := s.repo.GetBoardState(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("get board state: %w", err)
	}
	return state, nil
}

// GetPublicBoardState returns the read-only viewer view of the board behind
// token (AC-09): columns and tasks only — deliberately no public_link field,
// matching the PublicBoardState schema (contracts/openapi.yaml). Returns
// domain.ErrLinkNotFound for an unknown or revoked token (AC-11).
func (s *StateService) GetPublicBoardState(ctx context.Context, token string) (*ports.PublicBoardState, error) {
	link, err := s.repo.PublicLinkByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get public board state: %w", err)
	}

	state, err := s.repo.GetBoardState(ctx, link.BoardID)
	if err != nil {
		return nil, fmt.Errorf("get public board state: %w", err)
	}

	return &ports.PublicBoardState{Columns: state.Columns}, nil
}
