package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

type CardService struct {
	repo domain.CardRepository
}

func NewCardService(repo domain.CardRepository) *CardService {
	return &CardService{repo: repo}
}

func (s *CardService) CreateCard(ctx context.Context, name string, assignee *string) (*domain.Card, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	card := domain.Card{
		ID:           id,
		Name:         name,
		Assignee:     assignee,
		ColumnStatus: domain.ColumnTodo,
	}
	if err := card.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateCard unconditionally overwrites name/assignee — no read-then-compare
// step, per ADR-0002.
func (s *CardService) UpdateCard(ctx context.Context, id uuid.UUID, name string, assignee *string) (*domain.Card, error) {
	card := domain.Card{ID: id, Name: name, Assignee: assignee}
	if err := card.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

func (s *CardService) MoveCard(ctx context.Context, id uuid.UUID, columnStatus string) (*domain.Card, error) {
	if !domain.IsValidColumnStatus(columnStatus) {
		return nil, domain.ErrInvalidColumn
	}
	return s.repo.Move(ctx, id, columnStatus)
}

func (s *CardService) DeleteCard(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *CardService) GetBoard(ctx context.Context) ([]domain.Card, error) {
	return s.repo.List(ctx)
}
