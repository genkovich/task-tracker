// Package app holds the board module's use-case services. TaskService (T5)
// implements create/edit/move/delete for tasks: it depends only on the
// consumer-side ports.Repository and ports.Broadcaster interfaces, never on
// infra directly (go-structs-interfaces.md).
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// eventTypeBoardStateChanged is the board.state_changed.v1 event type
// (contracts/events.md) broadcast after every task mutation.
const eventTypeBoardStateChanged = "board.state_changed"

// TaskService implements the task use-cases (create/edit/move/delete) for
// the single board (CONTEXT.md invariant: the product always has exactly
// one board).
type TaskService struct {
	repo    ports.Repository
	bcast   ports.Broadcaster
	boardID uuid.UUID
}

// NewTaskService wires a TaskService against the given repository,
// broadcaster, and the single board's id.
func NewTaskService(repo ports.Repository, bcast ports.Broadcaster, boardID uuid.UUID) *TaskService {
	return &TaskService{repo: repo, bcast: bcast, boardID: boardID}
}

// CreateTask creates a task with the given title/assignee in the board's
// leftmost column (AC-01). An empty title is rejected before any repository
// write (AC-02). On success it broadcasts board.state_changed exactly once.
func (s *TaskService) CreateTask(ctx context.Context, title string, assignee *string) (*domain.Task, error) {
	leftmostID, err := s.repo.LeftmostColumnID(ctx, s.boardID)
	if err != nil {
		return nil, fmt.Errorf("resolve leftmost column: %w", err)
	}

	task, err := domain.NewTask(leftmostID, title, assignee)
	if err != nil {
		return nil, err
	}

	if err := s.repo.InsertTask(ctx, task); err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	s.broadcast()
	return task, nil
}

// EditTask updates an existing task's title and assignee (AC-03). On
// success it broadcasts board.state_changed exactly once.
func (s *TaskService) EditTask(ctx context.Context, taskID uuid.UUID, title string, assignee *string) (*domain.Task, error) {
	task := &domain.Task{ID: taskID, Assignee: assignee}
	if err := task.SetTitle(title); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	s.broadcast()
	return task, nil
}

// MoveTask moves a task to columnID (AC-04). A move to a column that does
// not exist is rejected, leaving the task in its previous column, as if the
// drop never happened (AC-05), and does not broadcast.
func (s *TaskService) MoveTask(ctx context.Context, taskID, columnID uuid.UUID) error {
	if err := s.repo.MoveTask(ctx, taskID, columnID); err != nil {
		return fmt.Errorf("move task: %w", err)
	}

	s.broadcast()
	return nil
}

// DeleteTask hard-deletes a task (AC-06). On success it broadcasts
// board.state_changed exactly once.
func (s *TaskService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	if err := s.repo.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	s.broadcast()
	return nil
}

// broadcast notifies every live connection that board state changed
// (contracts/events.md board.state_changed.v1) — the signal a mutation's
// caller relies on to fan out to SSE clients.
func (s *TaskService) broadcast() {
	s.bcast.Broadcast(ports.Event{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		EventType:  eventTypeBoardStateChanged,
		Version:    1,
		OccurredAt: time.Now(),
	})
}
