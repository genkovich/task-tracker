package app_test

// Unit tests for BoardService (boards BRD-01/BRD-02/BRD-03): the dashboard
// list and the create use-case. The repository is faked; the real
// transactionality of CreateBoard lives in the repo and is pinned by the
// infra integration suite — here the pin is that board and columns arrive in
// ONE repo call, so no code path can persist one without the other.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// boardsFakeRepo reuses task_service_test.go's fakeRepo for the untouched
// surface and overrides the two board methods this suite exercises.
type boardsFakeRepo struct {
	*fakeRepo

	summaries   []ports.BoardSummary
	createCalls int
	created     *domain.Board
	createdCols []domain.Column
}

func (r *boardsFakeRepo) ListBoards(_ context.Context) ([]ports.BoardSummary, error) {
	return r.summaries, nil
}

func (r *boardsFakeRepo) CreateBoard(_ context.Context, board *domain.Board, columns []domain.Column) error {
	r.createCalls++
	board.CreatedAt = time.Now()
	for i := range columns {
		columns[i].CreatedAt = board.CreatedAt
	}
	r.created = board
	r.createdCols = columns
	return nil
}

func newBoardsFakeRepo() *boardsFakeRepo {
	return &boardsFakeRepo{fakeRepo: newFakeRepo(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))}
}

// TestCreateBoard_ValidName covers BRD-02: a non-empty name creates the
// board together with its three fixed columns in a single repository call,
// and returns the full new board state — empty columns, no public link.
func TestCreateBoard_ValidName(t *testing.T) {
	repo := newBoardsFakeRepo()
	svc := app.NewBoardService(repo)

	state, err := svc.CreateBoard(context.Background(), "Воркшоп")

	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, "Воркшоп", state.Name)
	require.NotEqual(t, uuid.Nil, state.ID)
	require.Nil(t, state.PublicLink, "a new board has no public link")

	require.Equal(t, 1, repo.createCalls,
		"board and columns must be persisted in one repository call (transactional by construction)")
	require.NotNil(t, repo.created)
	require.Len(t, repo.createdCols, 3, "every board carries exactly three fixed columns")

	require.Len(t, state.Columns, 3)
	wantNames := []string{"To Do", "In Progress", "Done"}
	for i, col := range state.Columns {
		require.Equal(t, wantNames[i], col.Name)
		require.Equal(t, int16(i), col.Position) //nolint:gosec // i is 0..2
		require.Equal(t, state.ID, col.BoardID, "each column must belong to the new board")
		require.Empty(t, col.Tasks, "a new board starts with no tasks")
	}
}

// TestCreateBoard_EmptyName_RejectedNoWrite covers BRD-03: an empty name is
// rejected before any repository write.
func TestCreateBoard_EmptyName_RejectedNoWrite(t *testing.T) {
	repo := newBoardsFakeRepo()
	svc := app.NewBoardService(repo)

	state, err := svc.CreateBoard(context.Background(), "   ")

	require.ErrorIs(t, err, domain.ErrBoardNameRequired)
	require.Nil(t, state)
	require.Equal(t, 0, repo.createCalls, "no board may be persisted when the name is empty")
}

// TestCreateBoard_TooLongName_RejectedNoWrite covers BRD-03's length bound:
// a name over 200 characters is rejected before any repository write.
func TestCreateBoard_TooLongName_RejectedNoWrite(t *testing.T) {
	repo := newBoardsFakeRepo()
	svc := app.NewBoardService(repo)

	state, err := svc.CreateBoard(context.Background(), strings.Repeat("б", 201))

	require.ErrorIs(t, err, domain.ErrBoardNameTooLong)
	require.Nil(t, state)
	require.Equal(t, 0, repo.createCalls, "no board may be persisted when the name is too long")
}

// TestListBoards covers BRD-01: the dashboard rows come straight from the
// repository — name and task count included.
func TestListBoards(t *testing.T) {
	repo := newBoardsFakeRepo()
	repo.summaries = []ports.BoardSummary{
		{ID: uuid.Must(uuid.NewV7()), Name: "Дошка команди", CreatedAt: time.Now(), TaskCount: 4},
		{ID: uuid.Must(uuid.NewV7()), Name: "Воркшоп", CreatedAt: time.Now(), TaskCount: 0},
	}
	svc := app.NewBoardService(repo)

	boards, err := svc.ListBoards(context.Background())

	require.NoError(t, err)
	require.Len(t, boards, 2)
	require.Equal(t, "Дошка команди", boards[0].Name)
	require.Equal(t, 4, boards[0].TaskCount)
	require.Equal(t, "Воркшоп", boards[1].Name)
	require.Equal(t, 0, boards[1].TaskCount)
}
