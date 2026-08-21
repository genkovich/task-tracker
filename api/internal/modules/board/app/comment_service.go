package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// CommentService implements the comment use-cases (tasks TSK-08/TSK-10). It
// lives beside TaskService rather than inside it: a comment never touches the
// tasks row, and task_handler.go already carries CRUD plus move plus a rate
// limit. Like every other service here it depends only on the consumer-side
// ports, never on infra.
type CommentService struct {
	repo  ports.Repository
	bcast ports.Broadcaster
}

// NewCommentService wires a CommentService against the given repository and
// broadcaster.
func NewCommentService(repo ports.Repository, bcast ports.Broadcaster) *CommentService {
	return &CommentService{repo: repo, bcast: bcast}
}

// AddComment adds a comment under taskID (TSK-08). An empty or over-long
// author/body is rejected before any repository write (TSK-09); an unknown
// task surfaces as domain.ErrTaskNotFound. On success it broadcasts
// board.state_changed to the task's board exactly once — the card's comment
// counter is part of the board's visible state (TSK-14).
func (s *CommentService) AddComment(ctx context.Context, taskID uuid.UUID, author, body string) (*domain.Comment, error) {
	comment, err := domain.NewComment(taskID, author, body)
	if err != nil {
		return nil, err
	}

	boardID, err := s.repo.InsertComment(ctx, comment)
	if err != nil {
		return nil, fmt.Errorf("insert comment: %w", err)
	}

	s.broadcast(boardID)
	return comment, nil
}

// ListComments returns a task's comments, oldest first (TSK-08). An unknown
// task is domain.ErrTaskNotFound rather than an empty list — "no comments"
// and "no such task" are different answers, and the contract documents a 404
// for the second.
func (s *CommentService) ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	if _, _, err := s.repo.TaskByID(ctx, taskID); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	comments, err := s.repo.ListComments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return comments, nil
}

// DeleteComment hard-deletes a comment under taskID (TSK-10) and broadcasts
// to the board its task belongs to, the same way AddComment does.
func (s *CommentService) DeleteComment(ctx context.Context, taskID, commentID uuid.UUID) error {
	boardID, err := s.repo.DeleteComment(ctx, taskID, commentID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	s.broadcast(boardID)
	return nil
}

// broadcast notifies boardID's live connections that its state changed —
// identical to TaskService's, because to a viewer a new comment and a moved
// card are the same thing: the board looks different now.
func (s *CommentService) broadcast(boardID uuid.UUID) {
	s.bcast.Broadcast(boardID, ports.Event{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		EventType:  eventTypeBoardStateChanged,
		Version:    1,
		OccurredAt: time.Now(),
	})
}
