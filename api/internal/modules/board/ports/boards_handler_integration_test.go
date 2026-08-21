//go:build integration

package ports_test

// Integration tests for the boards dashboard routes (boards contract
// listBoards/createBoard/getBoard):
//   - BRD-01: GET /api/v1/boards lists every board with its task count.
//   - BRD-02: POST /api/v1/boards creates the board + its three fixed
//     columns transactionally and returns the full board state.
//   - BRD-03: an empty name is rejected with 422 board.name_required and
//     nothing is persisted.
//   - BRD-04: GET /api/v1/boards/{boardId} returns 404 board.not_found for
//     an unknown board.
//
// Naming prefixed t20* — the sibling ports_test files (t7-t10) are
// self-contained the same way.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

// seeded "first board" from migration 000007, named by 000012.
var t20SeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

type t20BoardSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	TaskCount int    `json:"task_count"`
}

type t20Column struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Position int               `json:"position"`
	Tasks    []json.RawMessage `json:"tasks"`
}

type t20BoardState struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Columns    []t20Column     `json:"columns"`
	PublicLink json.RawMessage `json:"public_link"`
}

type t20ErrorResponse struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func setupT20BoardsServer(t *testing.T) (*httptest.Server, *database.DB) {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()

	boardHandler := ports.NewBoardHandler(app.NewBoardService(repo), app.NewStateService(repo))
	taskHandler := ports.NewTaskHandler(app.NewTaskService(repo, hub))

	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		boardHandler.RegisterRoutes(api)
		taskHandler.RegisterRoutes(api)
	})

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts, c.DB
}

func t20PostJSON(t *testing.T, url string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx // test helper
	require.NoError(t, err)
	return resp
}

// TestBoardsHandler_ListBoards covers BRD-01: the dashboard list carries the
// seeded board with its name and live task count.
func TestBoardsHandler_ListBoards(t *testing.T) {
	ts, _ := setupT20BoardsServer(t)

	// One task on the seed board so task_count is non-trivial.
	createResp := t20PostJSON(t, ts.URL+"/api/v1/tasks",
		map[string]any{"board_id": t20SeedBoardID.String(), "title": "Counted task"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	createResp.Body.Close()

	resp, err := http.Get(ts.URL + "/api/v1/boards") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var boards []t20BoardSummary
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&boards))
	require.Len(t, boards, 1, "only the seeded board exists initially")
	require.Equal(t, t20SeedBoardID.String(), boards[0].ID)
	require.Equal(t, "Дошка команди", boards[0].Name, "migration 000012 backfills the seed board's name")
	require.Equal(t, 1, boards[0].TaskCount)
	require.NotEmpty(t, boards[0].CreatedAt)
}

// TestBoardsHandler_CreateBoard_HappyPath covers BRD-02: POST /api/v1/boards
// returns 201 with the full new board — three fixed columns, no tasks, no
// public link — and the board shows up on the dashboard list.
func TestBoardsHandler_CreateBoard_HappyPath(t *testing.T) {
	ts, db := setupT20BoardsServer(t)

	resp := t20PostJSON(t, ts.URL+"/api/v1/boards", map[string]any{"name": "Воркшоп"})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var state t20BoardState
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&state))
	require.Equal(t, "Воркшоп", state.Name)
	require.NotEmpty(t, state.ID)
	require.Len(t, state.Columns, 3, "a new board carries exactly the three fixed columns")
	wantNames := []string{"To Do", "In Progress", "Done"}
	for i, col := range state.Columns {
		require.Equal(t, wantNames[i], col.Name)
		require.Equal(t, i, col.Position)
		require.Empty(t, col.Tasks)
	}

	// Durable: the columns really exist for the new board.
	var columnCount int
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM columns WHERE board_id = $1`, state.ID).Scan(&columnCount))
	require.Equal(t, 3, columnCount)

	// And the dashboard list now carries both boards.
	listResp, err := http.Get(ts.URL + "/api/v1/boards") //nolint:noctx // test helper
	require.NoError(t, err)
	defer listResp.Body.Close()
	var boards []t20BoardSummary
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&boards))
	require.Len(t, boards, 2)
	require.Equal(t, "Воркшоп", boards[1].Name, "boards are listed oldest first")
}

// TestBoardsHandler_CreateBoard_EmptyName covers BRD-03: an empty name is a
// 422 board.name_required and no board row is written.
func TestBoardsHandler_CreateBoard_EmptyName(t *testing.T) {
	ts, db := setupT20BoardsServer(t)

	resp := t20PostJSON(t, ts.URL+"/api/v1/boards", map[string]any{"name": "  "})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var body t20ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.name_required", body.Error.Code)

	var boardCount int
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM boards`).Scan(&boardCount))
	require.Equal(t, 1, boardCount, "only the seeded board may exist after a rejected create")
}

// TestBoardsHandler_GetBoard_UnknownID covers BRD-04's error branch: an
// unknown board id is a 404 board.not_found, a non-UUID id a 400.
func TestBoardsHandler_GetBoard_UnknownID(t *testing.T) {
	ts, _ := setupT20BoardsServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/boards/" + uuid.NewString()) //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body t20ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.not_found", body.Error.Code)

	badResp, err := http.Get(ts.URL + "/api/v1/boards/not-a-uuid") //nolint:noctx // test helper
	require.NoError(t, err)
	defer badResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, badResp.StatusCode)
}

// TestBoardsHandler_CreateTask_UnknownBoard covers BRD-08's error branch:
// creating a task on a non-existent board is a 404 board.not_found.
func TestBoardsHandler_CreateTask_UnknownBoard(t *testing.T) {
	ts, _ := setupT20BoardsServer(t)

	resp := t20PostJSON(t, ts.URL+"/api/v1/tasks",
		map[string]any{"board_id": uuid.NewString(), "title": "Orphan"})
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body t20ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.not_found", body.Error.Code)
}
