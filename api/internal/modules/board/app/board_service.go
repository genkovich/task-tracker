package app

import (
	"context"
	"fmt"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// BoardService implements the dashboard use-cases: list all boards and
// create a new one (boards BRD-01/BRD-02).
type BoardService struct {
	repo ports.Repository
}

// NewBoardService wires a BoardService against the given repository.
func NewBoardService(repo ports.Repository) *BoardService {
	return &BoardService{repo: repo}
}

// ListBoards returns every board (oldest first) with its total task count —
// the dashboard rows (BRD-01).
func (s *BoardService) ListBoards(ctx context.Context) ([]ports.BoardSummary, error) {
	boards, err := s.repo.ListBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list boards: %w", err)
	}
	return boards, nil
}

// CreateBoard creates a board with the given name and its three fixed
// columns in one transaction (BRD-02) and returns the full new board state
// (empty columns, no public link). An empty or oversized name is rejected
// before any write (BRD-03).
func (s *BoardService) CreateBoard(ctx context.Context, name string) (*ports.BoardState, error) {
	board, err := domain.NewBoard(name)
	if err != nil {
		return nil, err
	}

	columns := domain.DefaultColumns(board.ID)
	if err := s.repo.CreateBoard(ctx, board, columns); err != nil {
		return nil, fmt.Errorf("create board: %w", err)
	}

	state := &ports.BoardState{
		ID:        board.ID,
		Name:      board.Name,
		CreatedAt: board.CreatedAt,
		Columns:   make([]ports.ColumnState, 0, len(columns)),
	}
	for _, col := range columns {
		state.Columns = append(state.Columns, ports.ColumnState{Column: col, Tasks: []domain.Task{}})
	}
	return state, nil
}
