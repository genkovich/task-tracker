package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
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

	return &ports.PublicBoardState{BoardID: link.BoardID, Columns: state.Columns}, nil
}

// GetTaskDetail returns one task in full with its comments (tasks TSK-01/
// TSK-08) — the team-editor detail view. Returns domain.ErrTaskNotFound for
// an unknown task.
func (s *StateService) GetTaskDetail(ctx context.Context, taskID uuid.UUID) (*ports.TaskDetail, error) {
	task, _, err := s.repo.TaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task detail: %w", err)
	}

	comments, err := s.repo.ListComments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task detail: %w", err)
	}

	return &ports.TaskDetail{Task: *task, Comments: comments}, nil
}

// GetPublicTaskDetail returns the same detail to a viewer holding token
// (TSK-12) — but only for a task that actually sits on the board behind that
// token. A task belonging to another board is refused with the very same
// domain.ErrLinkNotFound an unknown token gets (TSK-13): a distinct error
// would itself confirm that the task exists somewhere.
func (s *StateService) GetPublicTaskDetail(ctx context.Context, token string, taskID uuid.UUID) (*ports.TaskDetail, error) {
	link, err := s.repo.PublicLinkByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get public task detail: %w", err)
	}

	task, boardID, err := s.repo.TaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, fmt.Errorf("get public task detail: %w", domain.ErrLinkNotFound)
		}
		return nil, fmt.Errorf("get public task detail: %w", err)
	}
	if boardID != link.BoardID {
		return nil, fmt.Errorf("get public task detail: %w", domain.ErrLinkNotFound)
	}

	comments, err := s.repo.ListComments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get public task detail: %w", err)
	}

	return &ports.TaskDetail{Task: *task, Comments: comments}, nil
}
