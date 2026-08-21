package app_test

// Unit tests for CommentService (tasks T5): AddComment, ListComments,
// DeleteComment against the faked ports.Repository / ports.Broadcaster shared
// with task_service_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// TestAddComment_Persists_AndBroadcastsToItsBoard covers TSK-08 and TSK-14: a
// valid comment is stored and the board it belongs to is notified exactly
// once — the card's comment counter is part of what the board looks like.
func TestAddComment_Persists_AndBroadcastsToItsBoard(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Discuss me"}
	repo.seedTask(task)
	bcast := &fakeBroadcaster{}
	svc := app.NewCommentService(repo, bcast)

	comment, err := svc.AddComment(context.Background(), task.ID, "Ada", "Стенд піднято")

	require.NoError(t, err)
	require.NotNil(t, comment)
	require.Equal(t, task.ID, comment.TaskID)
	require.Equal(t, "Ada", comment.Author)
	require.Equal(t, "Стенд піднято", comment.Body)
	require.False(t, comment.CreatedAt.IsZero(), "a stored comment must carry its created_at")

	require.Equal(t, 1, repo.commentCount())
	require.Equal(t, 1, bcast.count(), "adding a comment must broadcast exactly once (TSK-14)")
	require.Equal(t, boardID, bcast.lastBoard(), "the broadcast must target the task's board (BRD-05)")
}

// TestAddComment_Invalid_RejectedBeforeAnyWrite covers TSK-09: the domain
// bounds are enforced before the repository is touched, and nothing is
// broadcast.
func TestAddComment_Invalid_RejectedBeforeAnyWrite(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Discuss me"}

	cases := []struct {
		name    string
		author  string
		body    string
		wantErr error
	}{
		{"empty author", "", "text", domain.ErrCommentAuthorRequired},
		{"empty body", "Ada", "   ", domain.ErrCommentBodyRequired},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo(boardID, columnID)
			repo.seedTask(task)
			bcast := &fakeBroadcaster{}
			svc := app.NewCommentService(repo, bcast)

			comment, err := svc.AddComment(context.Background(), task.ID, tc.author, tc.body)

			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, comment)
			require.Equal(t, 0, repo.commentCount(), "an invalid comment must not be persisted")
			require.Equal(t, 0, bcast.count(), "a rejected comment must not broadcast")
		})
	}
}

// An unknown task surfaces as the task not-found sentinel, so ports can map
// it to a 404 rather than letting an FK violation become a 500.
func TestAddComment_UnknownTask_ReturnsTaskNotFound(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	bcast := &fakeBroadcaster{}
	svc := app.NewCommentService(repo, bcast)

	comment, err := svc.AddComment(context.Background(), uuid.Must(uuid.NewV7()), "Ada", "hello")

	require.ErrorIs(t, err, domain.ErrTaskNotFound)
	require.Nil(t, comment)
	require.Equal(t, 0, bcast.count())
}

// TestListComments_OldestFirst covers TSK-08's ordering: a thread reads top
// to bottom in the order it was written.
func TestListComments_OldestFirst(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Discuss me"}
	repo.seedTask(task)

	base := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	repo.seedComment(domain.Comment{ID: uuid.Must(uuid.NewV7()), TaskID: task.ID, Author: "Ada", Body: "second", CreatedAt: base.Add(time.Hour)})
	repo.seedComment(domain.Comment{ID: uuid.Must(uuid.NewV7()), TaskID: task.ID, Author: "Grace", Body: "first", CreatedAt: base})

	svc := app.NewCommentService(repo, &fakeBroadcaster{})
	comments, err := svc.ListComments(context.Background(), task.ID)

	require.NoError(t, err)
	require.Len(t, comments, 2)
	require.Equal(t, "first", comments[0].Body, "oldest comment must come first (TSK-08)")
	require.Equal(t, "second", comments[1].Body)
}

// TestDeleteComment_RemovesAndBroadcasts covers TSK-10.
func TestDeleteComment_RemovesAndBroadcasts(t *testing.T) {
	boardID := uuid.Must(uuid.NewV7())
	columnID := uuid.Must(uuid.NewV7())
	repo := newFakeRepo(boardID, columnID)
	task := domain.Task{ID: uuid.Must(uuid.NewV7()), ColumnID: columnID, Title: "Discuss me"}
	repo.seedTask(task)
	comment := domain.Comment{ID: uuid.Must(uuid.NewV7()), TaskID: task.ID, Author: "Ada", Body: "bye"}
	repo.seedComment(comment)
	bcast := &fakeBroadcaster{}
	svc := app.NewCommentService(repo, bcast)

	require.NoError(t, svc.DeleteComment(context.Background(), comment.ID))

	require.Equal(t, 0, repo.commentCount(), "the comment must be gone (TSK-10)")
	require.Equal(t, 1, bcast.count(), "a successful delete must broadcast exactly once")
	require.Equal(t, boardID, bcast.lastBoard())
}

// An unknown comment id is a 404, and nothing is broadcast for a delete that
// did not happen.
func TestDeleteComment_Unknown_ReturnsCommentNotFound(t *testing.T) {
	repo := newFakeRepo(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	bcast := &fakeBroadcaster{}
	svc := app.NewCommentService(repo, bcast)

	err := svc.DeleteComment(context.Background(), uuid.Must(uuid.NewV7()))

	require.ErrorIs(t, err, domain.ErrCommentNotFound)
	require.Equal(t, 0, bcast.count())
}
