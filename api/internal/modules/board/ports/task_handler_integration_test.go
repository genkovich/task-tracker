//go:build integration

package ports_test

// RED for T7 (docs/features/board/tasks/T7-ports-task-routes.md):
// api/internal/modules/board/ports/task_handler.go does not exist yet —
// ports.NewTaskHandler is expected to land there, wrapping app.TaskService
// (T5) behind POST /api/v1/tasks, PATCH/DELETE /api/v1/tasks/{taskId}, and
// POST /api/v1/tasks/{taskId}/move (contracts/openapi.yaml operationIds
// createTask/editTask/deleteTask/moveTask).
//
// Covers:
//   - AC-01: POST /api/v1/tasks with a non-empty title returns 201 with the
//     created Task (leftmost column).
//   - AC-02: POST/PATCH with an empty title returns 422 task.title_required.
//   - AC-03: PATCH /api/v1/tasks/{id} with a new title/assignee returns 200
//     with the updated Task.
//   - AC-04: POST /api/v1/tasks/{id}/move to a valid column returns 200 with
//     the task recorded in the new column.
//   - AC-05: POST /api/v1/tasks/{id}/move to an unknown column returns 422
//     board.column_not_found, task unchanged.
//   - AC-06: DELETE /api/v1/tasks/{id} returns 204 and the task is gone.
//   - unknown-id 404 task.not_found on PATCH/DELETE/move.
//   - spec §6.1 abuse case: the 31st create request within a minute from the
//     same client returns 429 task.rate_limited.
//
// Deliberately self-contained: wires the real app.TaskService (T5, already
// implemented) directly against a fresh ephemeral Postgres, independent of
// sibling ports_test files (T8/T9/T10) also under active development.
// Type/helper names below are prefixed t7TaskHandler* to avoid clashing with
// them.
//
// No production code exists yet for ports.NewTaskHandler — this is the RED
// step; the package will not compile until T7 adds it.

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

// seeded board/columns from migrations 000007/000009 (data-model.md,
// CONTEXT.md: exactly one board row) — matches sibling ports_test files.
var t7TaskHandlerSeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// t7TaskWire mirrors the Task schema in contracts/openapi.yaml — decoded
// independently of any internal DTO type so this test pins the wire
// contract, not an implementation detail.
type t7TaskWire struct {
	ID        string  `json:"id"`
	ColumnID  string  `json:"column_id"`
	Title     string  `json:"title"`
	Assignee  *string `json:"assignee"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type t7ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// t7TaskHandlerFixture bundles the task handler under test (T7) with the
// real repo/task-service, backed by a fresh ephemeral Postgres per test.
type t7TaskHandlerFixture struct {
	ts   *httptest.Server
	repo *infra.PostgresRepository
}

func setupT7TaskHandlerServer(t *testing.T) t7TaskHandlerFixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()

	taskSvc := app.NewTaskService(repo, hub)

	// ports.NewTaskHandler does not exist yet (T7) — this line is why the
	// package fails to compile before T7 lands.
	taskHandler := ports.NewTaskHandler(taskSvc, app.NewStateService(repo))

	r := chi.NewRouter()
	r.Route("/api/v1", taskHandler.RegisterRoutes)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return t7TaskHandlerFixture{ts: ts, repo: repo}
}

func (f t7TaskHandlerFixture) do(t *testing.T, method, path string, body any, headers map[string]string) *http.Response {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, f.ts.URL+path, reader) //nolint:noctx // test helper
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func (f t7TaskHandlerFixture) createTask(t *testing.T, title string) t7TaskWire {
	t.Helper()
	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": title}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "seed task creation must succeed")

	var task t7TaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&task))
	return task
}

// TestTaskHandler_CreateTask_HappyPath covers AC-01: a task with a non-empty
// title is created in the board's leftmost column.
func TestTaskHandler_CreateTask_HappyPath(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": "Write the report", "assignee": "Ada"}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "POST /api/v1/tasks (AC-01) status")

	var task t7TaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&task))
	require.NotEmpty(t, task.ID)
	require.Equal(t, "Write the report", task.Title)
	require.NotEmpty(t, task.ColumnID, "task must land in a column (AC-01)")

	leftmostID, err := f.repo.LeftmostColumnID(context.Background(), t7TaskHandlerSeedBoardID)
	require.NoError(t, err)
	require.Equal(t, leftmostID.String(), task.ColumnID, "AC-01: task must be created in the leftmost column")
}

// TestTaskHandler_CreateTask_EmptyTitle covers AC-02: an empty title is
// rejected with 422 task.title_required, no task created.
func TestTaskHandler_CreateTask_EmptyTitle(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": ""}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusUnprocessableEntity, resp.StatusCode, "POST /api/v1/tasks with empty title (AC-02) status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.title_required", body.Error.Code)
}

// TestTaskHandler_EditTask_HappyPath covers AC-03: editing a task's title
// and assignee persists and is reflected in the response.
func TestTaskHandler_EditTask_HappyPath(t *testing.T) {
	f := setupT7TaskHandlerServer(t)
	task := f.createTask(t, "Original title")

	resp := f.do(t, http.MethodPatch, "/api/v1/tasks/"+task.ID, map[string]any{"title": "Updated title", "assignee": "Grace"}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusOK, resp.StatusCode, "PATCH /api/v1/tasks/{id} (AC-03) status")

	var updated t7TaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	require.Equal(t, "Updated title", updated.Title)
	require.NotNil(t, updated.Assignee)
	require.Equal(t, "Grace", *updated.Assignee)
	// Root G pin: the edit response is a complete Task (openapi.yaml
	// requires column_id/created_at), not just the edited fields.
	require.Equal(t, task.ColumnID, updated.ColumnID, "PATCH response must carry the task's real column_id")
	require.Equal(t, task.CreatedAt, updated.CreatedAt, "PATCH response must carry the task's real created_at")
	require.NotEmpty(t, updated.UpdatedAt)
}

// AC-03/AC-02 edge: editing an unknown task returns 404 task.not_found.
func TestTaskHandler_EditTask_UnknownID(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	resp := f.do(t, http.MethodPatch, "/api/v1/tasks/"+uuid.NewString(), map[string]any{"title": "Whatever"}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "PATCH /api/v1/tasks/{unknown-id} status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}

// AC-02 applies to edits too: an empty title on PATCH is rejected.
func TestTaskHandler_EditTask_EmptyTitle(t *testing.T) {
	f := setupT7TaskHandlerServer(t)
	task := f.createTask(t, "Original title")

	resp := f.do(t, http.MethodPatch, "/api/v1/tasks/"+task.ID, map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": ""}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusUnprocessableEntity, resp.StatusCode, "PATCH /api/v1/tasks/{id} with empty title (AC-02) status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.title_required", body.Error.Code)
}

// TestTaskHandler_DeleteTask_HappyPath covers AC-06: deleting a task returns
// 204 and it is gone.
func TestTaskHandler_DeleteTask_HappyPath(t *testing.T) {
	f := setupT7TaskHandlerServer(t)
	task := f.createTask(t, "Delete me")

	resp := f.do(t, http.MethodDelete, "/api/v1/tasks/"+task.ID, nil, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusNoContent, resp.StatusCode, "DELETE /api/v1/tasks/{id} (AC-06) status")

	// Editing the now-deleted task must 404.
	editResp := f.do(t, http.MethodPatch, "/api/v1/tasks/"+task.ID, map[string]any{"title": "Should not work"}, nil)
	defer editResp.Body.Close()
	require.Equal(t, http.StatusNotFound, editResp.StatusCode, "AC-06: deleted task must be gone from the board")
}

// AC-06 edge: deleting an unknown task returns 404 task.not_found.
func TestTaskHandler_DeleteTask_UnknownID(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	resp := f.do(t, http.MethodDelete, "/api/v1/tasks/"+uuid.NewString(), nil, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "DELETE /api/v1/tasks/{unknown-id} status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}

// TestTaskHandler_MoveTask_HappyPath covers AC-04: moving a task to a valid
// column returns 200 with the task recorded there.
func TestTaskHandler_MoveTask_HappyPath(t *testing.T) {
	f := setupT7TaskHandlerServer(t)
	task := f.createTask(t, "Move me")

	state, err := f.repo.GetBoardState(context.Background(), t7TaskHandlerSeedBoardID)
	require.NoError(t, err)
	require.GreaterOrEqualf(t, len(state.Columns), 2, "seed must provide at least two columns to move between")

	var targetColumnID uuid.UUID
	for _, col := range state.Columns {
		if col.ID.String() != task.ColumnID {
			targetColumnID = col.ID
			break
		}
	}
	require.NotEqual(t, uuid.Nil, targetColumnID)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/move", map[string]any{"column_id": targetColumnID.String()}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusOK, resp.StatusCode, "POST /api/v1/tasks/{id}/move (AC-04) status")

	var moved t7TaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&moved))
	require.Equal(t, targetColumnID.String(), moved.ColumnID, "AC-04: task must be recorded in the new column")
	// Root G pin: the move response is a complete Task, not an id/column stub.
	require.Equal(t, task.ID, moved.ID)
	require.Equal(t, "Move me", moved.Title, "move response must carry the task's title")
	require.Equal(t, task.CreatedAt, moved.CreatedAt, "move response must carry the task's real created_at")
	require.NotEmpty(t, moved.UpdatedAt)
}

// TestTaskHandler_MoveTask_UnknownColumn covers AC-05: moving to a column
// that does not exist returns 422 board.column_not_found, task unchanged.
func TestTaskHandler_MoveTask_UnknownColumn(t *testing.T) {
	f := setupT7TaskHandlerServer(t)
	task := f.createTask(t, "Stay put")

	resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/move", map[string]any{"column_id": uuid.NewString()}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusUnprocessableEntity, resp.StatusCode, "POST /api/v1/tasks/{id}/move to unknown column (AC-05) status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.column_not_found", body.Error.Code)

	// AC-05: the rejected move leaves the task in its source column, as if
	// the drop never happened.
	state, err := f.repo.GetBoardState(context.Background(), t7TaskHandlerSeedBoardID)
	require.NoError(t, err)
	var foundColumnID string
	for _, col := range state.Columns {
		for _, tk := range col.Tasks {
			if tk.ID.String() == task.ID {
				foundColumnID = col.ID.String()
			}
		}
	}
	require.Equal(t, task.ColumnID, foundColumnID, "AC-05: task must stay in its source column after a rejected move")
}

// AC-04/AC-05 edge: moving an unknown task returns 404 task.not_found.
func TestTaskHandler_MoveTask_UnknownTask(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/move", map[string]any{"column_id": uuid.NewString()}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "POST /api/v1/tasks/{unknown-id}/move status")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}

// TestTaskHandler_CreateTask_RateLimited covers spec §6.1's abuse case: the
// 31st create request within a minute from the same client is rejected with
// 429 task.rate_limited, not persisted.
func TestTaskHandler_CreateTask_RateLimited(t *testing.T) {
	f := setupT7TaskHandlerServer(t)

	for i := range 30 {
		resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": "Bulk task"}, nil)
		resp.Body.Close()
		require.Equalf(t, http.StatusCreated, resp.StatusCode, "creation #%d within the 30/min budget must succeed", i+1)
	}

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{"board_id": t7TaskHandlerSeedBoardID.String(), "title": "One too many"}, nil)
	defer resp.Body.Close()
	require.Equalf(t, http.StatusTooManyRequests, resp.StatusCode, "the 31st creation within a minute must return 429 (spec §6.1)")

	var body t7ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.rate_limited", body.Error.Code)
}
