package app_test

// Unit tests for the tasks feature's app-layer behaviour (T4): the detail
// fields travelling through create/edit, the team-editor detail read, and —
// the one that matters most — the token-scoped read refusing a task that
// belongs to another board (TSK-13).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// TestCreateTask_CarriesDetails covers TSK-01/TSK-03/TSK-05: description,
// priority and due date survive the create path down to the stored row.
func TestCreateTask_CarriesDetails(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	svc := app.NewTaskService(repo, &fakeBroadcaster{})

	due := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	task, err := svc.CreateTask(context.Background(), boardID, domain.TaskDetails{
		Title:       "Підняти стенд",
		Description: "Розгорнути на VPS",
		Priority:    string(domain.PriorityHigh),
		DueDate:     &due,
	})

	require.NoError(t, err)
	stored, ok := repo.getTask(task.ID)
	require.True(t, ok)
	require.Equal(t, "Розгорнути на VPS", stored.Description)
	require.Equal(t, domain.PriorityHigh, stored.Priority)
	require.NotNil(t, stored.DueDate)
	require.Equal(t, due, *stored.DueDate)
}

// TestCreateTask_UnknownPriority_RejectedNoWriteNoBroadcast covers TSK-04.
func TestCreateTask_UnknownPriority_RejectedNoWriteNoBroadcast(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	leftmostID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, leftmostID)
	bcast := &fakeBroadcaster{}
	svc := app.NewTaskService(repo, bcast)

	task, err := svc.CreateTask(context.Background(), boardID, domain.TaskDetails{
		Title:    "ok",
		Priority: "urgent",
	})

	require.ErrorIs(t, err, domain.ErrPriorityInvalid)
	require.Nil(t, task)
	require.Equal(t, 0, repo.taskCount(), "an unknown priority must not persist a task")
	require.Equal(t, 0, bcast.count(), "a rejected creation must not broadcast")
}

// TestEditTask_ClearsDueDate covers TSK-07: leaving the due date out of an
// edit clears it, the same way leaving out the assignee always has.
func TestEditTask_ClearsDueDate(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	due := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	existing := domain.Task{
		ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Deadline task",
		Description: "kept", Priority: domain.PriorityHigh, DueDate: &due,
	}
	repo.seedTask(existing)
	svc := app.NewTaskService(repo, &fakeBroadcaster{})

	updated, err := svc.EditTask(context.Background(), existing.ID, domain.TaskDetails{
		Title:       "Deadline task",
		Description: "kept",
		Priority:    string(domain.PriorityHigh),
	})

	require.NoError(t, err)
	require.Nil(t, updated.DueDate, "an edit without a due date clears it (TSK-07)")
	require.Equal(t, "kept", updated.Description, "clearing the deadline must not touch the rest")
	require.Equal(t, domain.PriorityHigh, updated.Priority)
}

// TestGetTaskDetail_ReturnsTaskAndComments covers the editor detail read.
func TestGetTaskDetail_ReturnsTaskAndComments(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Detailed", Description: "the body"}
	repo.seedTask(task)
	repo.seedComment(domain.Comment{ID: uuid.Must(uuid.NewV7()), TaskID: task.ID, Author: "Ada", Body: "note"})
	svc := app.NewStateService(repo)

	detail, err := svc.GetTaskDetail(context.Background(), task.ID)

	require.NoError(t, err)
	require.Equal(t, "the body", detail.Task.Description)
	require.Len(t, detail.Comments, 1)
	require.Equal(t, "note", detail.Comments[0].Body)
}

func TestGetTaskDetail_UnknownTask_ReturnsTaskNotFound(t *testing.T) {
	repo := newFakeRepo(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	svc := app.NewStateService(repo)

	detail, err := svc.GetTaskDetail(context.Background(), uuid.Must(uuid.NewV7()))

	require.Nil(t, detail)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// TestGetPublicTaskDetail_ValidToken_ReturnsDetail covers TSK-12: a viewer
// holding a live token for the task's own board sees the full detail.
func TestGetPublicTaskDetail_ValidToken_ReturnsDetail(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Public detail", Description: "visible"}
	repo.seedTask(task)
	repo.seedLink("live-token", boardID)
	svc := app.NewStateService(repo)

	detail, err := svc.GetPublicTaskDetail(context.Background(), "live-token", task.ID)

	require.NoError(t, err)
	require.Equal(t, "visible", detail.Task.Description)
}

// TestGetPublicTaskDetail_TaskOfAnotherBoard_ReturnsLinkNotFound is the
// feature's central authorization test (TSK-13): without the board check, one
// public link would read every task in the product. The error is deliberately
// the same one an unknown token gets — a distinct code would itself confirm
// the task exists.
func TestGetPublicTaskDetail_TaskOfAnotherBoard_ReturnsLinkNotFound(t *testing.T) {
	boardA := uuid.Must(uuid.NewV7())
	boardB := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardA, columnID)

	foreign := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Board B secret", Description: "must not leak"}
	repo.seedTask(foreign)
	repo.taskBoards[foreign.ID] = boardB
	repo.seedLink("token-of-board-a", boardA)
	svc := app.NewStateService(repo)

	detail, err := svc.GetPublicTaskDetail(context.Background(), "token-of-board-a", foreign.ID)

	require.Nil(t, detail, "a task of another board must never be returned (TSK-13)")
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}

// An unknown task under a valid token reads as an invalid link too, for the
// same reason: distinguishing "no such task" from "not your task" would leak
// which ids exist.
func TestGetPublicTaskDetail_UnknownTask_ReturnsLinkNotFound(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, uuid.Must(uuid.NewV7()))
	repo.seedLink("live-token", boardID)
	svc := app.NewStateService(repo)

	detail, err := svc.GetPublicTaskDetail(context.Background(), "live-token", uuid.Must(uuid.NewV7()))

	require.Nil(t, detail)
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}

// A revoked (unknown) token never reaches the task lookup at all.
func TestGetPublicTaskDetail_UnknownToken_ReturnsLinkNotFound(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Detail"}
	repo.seedTask(task)
	svc := app.NewStateService(repo)

	detail, err := svc.GetPublicTaskDetail(context.Background(), "revoked-token", task.ID)

	require.Nil(t, detail)
	require.ErrorIs(t, err, domain.ErrLinkNotFound)
}
