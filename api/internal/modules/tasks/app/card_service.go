package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

type CardService struct {
	repo        domain.CardRepository
	broadcaster *Broadcaster
}

func NewCardService(repo domain.CardRepository, broadcaster *Broadcaster) *CardService {
	return &CardService{repo: repo, broadcaster: broadcaster}
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
	s.broadcaster.Publish(BoardEvent{Type: EventCardCreated, Payload: card.ID})
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
	s.broadcaster.Publish(BoardEvent{Type: EventCardUpdated, Payload: card.ID})
	return &card, nil
}

func (s *CardService) MoveCard(ctx context.Context, id uuid.UUID, columnStatus string) (*domain.Card, error) {
	if !domain.IsValidColumnStatus(columnStatus) {
		return nil, domain.ErrInvalidColumn
	}
	card, err := s.repo.Move(ctx, id, columnStatus)
	if err != nil {
		return nil, err
	}
	s.broadcaster.Publish(BoardEvent{Type: EventCardMoved, Payload: card.ID})
	return card, nil
}

func (s *CardService) DeleteCard(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcaster.Publish(BoardEvent{Type: EventCardDeleted, Payload: id})
	return nil
}

func (s *CardService) GetBoard(ctx context.Context) ([]domain.Card, error) {
	return s.repo.List(ctx)
}
