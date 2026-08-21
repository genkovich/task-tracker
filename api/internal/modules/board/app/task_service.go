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

// TaskService implements the task use-cases (create/edit/move/delete).
// Every mutation broadcasts to exactly the mutated task's board (boards
// BRD-05) — the repository reports which board that is.
type TaskService struct {
	repo  ports.Repository
	bcast ports.Broadcaster
}

// NewTaskService wires a TaskService against the given repository and
// broadcaster.
func NewTaskService(repo ports.Repository, bcast ports.Broadcaster) *TaskService {
	return &TaskService{repo: repo, bcast: bcast}
}

// CreateTask creates a task with the given details in the leftmost column of
// boardID (AC-01, boards BRD-08). An empty title is rejected before any
// repository write (AC-02), as is an over-long description or an unknown
// priority (tasks TSK-02/TSK-04); an unknown board surfaces as
// domain.ErrBoardNotFound. On success it broadcasts board.state_changed to
// that board exactly once.
func (s *TaskService) CreateTask(ctx context.Context, boardID uuid.UUID, details domain.TaskDetails) (*domain.Task, error) {
	leftmostID, err := s.repo.LeftmostColumnID(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("resolve leftmost column: %w", err)
	}

	task, err := domain.NewTask(leftmostID, details)
	if err != nil {
		return nil, err
	}

	if err := s.repo.InsertTask(ctx, task); err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	s.broadcast(boardID)
	return task, nil
}

// EditTask replaces an existing task's details (AC-03, tasks TSK-01/TSK-03/
// TSK-05/TSK-07) and returns the complete updated task — the repository fills
// the remaining fields (column_id, created_at, updated_at) from the stored
// row. On success it broadcasts board.state_changed to the task's board
// exactly once.
func (s *TaskService) EditTask(ctx context.Context, taskID uuid.UUID, details domain.TaskDetails) (*domain.Task, error) {
	task := &domain.Task{ID: taskID}
	if err := task.SetDetails(details); err != nil {
		return nil, err
	}

	boardID, err := s.repo.UpdateTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	s.broadcast(boardID)
	return task, nil
}

// MoveTask moves a task to columnID (AC-04) and returns the complete moved
// task. A move to a column that does not exist is rejected, leaving the
// task in its previous column, as if the drop never happened (AC-05), and
// does not broadcast.
func (s *TaskService) MoveTask(ctx context.Context, taskID, columnID uuid.UUID) (*domain.Task, error) {
	task, boardID, err := s.repo.MoveTask(ctx, taskID, columnID)
	if err != nil {
		return nil, fmt.Errorf("move task: %w", err)
	}

	s.broadcast(boardID)
	return task, nil
}

// DeleteTask hard-deletes a task (AC-06). On success it broadcasts
// board.state_changed to the task's former board exactly once.
func (s *TaskService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	boardID, err := s.repo.DeleteTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	s.broadcast(boardID)
	return nil
}

// broadcast notifies boardID's live connections that its state changed
// (contracts/events.md board.state_changed.v1) — the signal a mutation's
// caller relies on to fan out to SSE clients.
func (s *TaskService) broadcast(boardID uuid.UUID) {
	s.bcast.Broadcast(boardID, ports.Event{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		EventType:  eventTypeBoardStateChanged,
		Version:    1,
		OccurredAt: time.Now(),
	})
}
