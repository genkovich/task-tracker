//go:build integration

package ports_test

// Integration tests for the tasks feature's team-editor HTTP surface (T6/T7):
// the task-detail GET, the detail fields on create/edit, and the comment
// routes — each against a real ephemeral Postgres, asserting the wire shape
// documented in docs/features/tasks/contracts/openapi.yaml rather than an
// internal DTO type.
//
// Names are prefixed tsk* to stay clear of the sibling ports_test files.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

var tskSeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// tskTaskWire mirrors the Task schema (tasks contract) — decoded
// independently of any internal DTO so this pins the wire contract.
type tskTaskWire struct {
	ID          string  `json:"id"`
	ColumnID    string  `json:"column_id"`
	Title       string  `json:"title"`
	Assignee    *string `json:"assignee"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type tskCommentWire struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type tskDetailWire struct {
	Task     tskTaskWire      `json:"task"`
	Comments []tskCommentWire `json:"comments"`
}

type tskErrorWire struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type tskFixture struct {
	ts *httptest.Server
}

func setupTskServer(t *testing.T) tskFixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()
	stateSvc := app.NewStateService(repo)

	taskHandler := ports.NewTaskHandler(app.NewTaskService(repo, hub), stateSvc)
	commentHandler := ports.NewCommentHandler(app.NewCommentService(repo, hub))

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		taskHandler.RegisterRoutes(r)
		commentHandler.RegisterRoutes(r)
	})

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return tskFixture{ts: ts}
}

func (f tskFixture) do(t *testing.T, method, path string, body any) *http.Response {
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

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func (f tskFixture) createTask(t *testing.T, body map[string]any) tskTaskWire {
	t.Helper()
	body["board_id"] = tskSeedBoardID.String()

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", body)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "seed task creation must succeed")

	var task tskTaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&task))
	return task
}

// TSK-01/TSK-03/TSK-05: create accepts the detail fields and echoes them, with
// the due date as a calendar day rather than an RFC3339 timestamp.
func TestTaskDetails_CreateTask_AcceptsAndEchoesDetails(t *testing.T) {
	f := setupTskServer(t)

	task := f.createTask(t, map[string]any{
		"title":       "Підняти стенд",
		"description": "Розгорнути на VPS",
		"priority":    "high",
		"due_date":    "2026-09-01",
	})

	require.Equal(t, "Розгорнути на VPS", task.Description)
	require.Equal(t, "high", task.Priority)
	require.NotNil(t, task.DueDate)
	require.Equal(t, "2026-09-01", *task.DueDate, "due_date travels as a calendar day")
}

// TSK-03: a task created without a priority comes back as medium, never
// empty — the card always has a marker to draw.
func TestTaskDetails_CreateTask_DefaultsToMediumPriority(t *testing.T) {
	f := setupTskServer(t)

	task := f.createTask(t, map[string]any{"title": "Bare task"})

	require.Equal(t, "medium", task.Priority)
	require.Nil(t, task.DueDate)
	require.Equal(t, "", task.Description)
}

// TSK-04: an unknown priority is refused with the documented 422 and nothing
// is stored.
func TestTaskDetails_CreateTask_UnknownPriority_Returns422(t *testing.T) {
	f := setupTskServer(t)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{
		"board_id": tskSeedBoardID.String(),
		"title":    "ok",
		"priority": "urgent",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var body tskErrorWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.priority_invalid", body.Error.Code)
}

// TSK-02: an over-long description is a 422, not a 500 from the driver.
func TestTaskDetails_CreateTask_OversizedDescription_Returns422(t *testing.T) {
	f := setupTskServer(t)

	oversized := make([]rune, 4001)
	for i := range oversized {
		oversized[i] = 'ї'
	}

	resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{
		"board_id":    tskSeedBoardID.String(),
		"title":       "ok",
		"description": string(oversized),
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var body tskErrorWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.description_too_long", body.Error.Code)
}

// A due date that is not a calendar day is the caller's mistake — 400, not a
// 500 further down (test-plan.md edge case).
func TestTaskDetails_CreateTask_MalformedDueDate_Returns400(t *testing.T) {
	f := setupTskServer(t)

	for _, raw := range []string{"tomorrow", "2026-13-01", "2026-09-01T10:00:00Z"} {
		resp := f.do(t, http.MethodPost, "/api/v1/tasks", map[string]any{
			"board_id": tskSeedBoardID.String(),
			"title":    "ok",
			"due_date": raw,
		})
		var body tskErrorWire
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		resp.Body.Close()

		require.Equalf(t, http.StatusBadRequest, resp.StatusCode, "due_date %q must be refused as a caller error", raw)
		require.Equal(t, "validation.invalid_due_date", body.Error.Code)
	}
}

// TSK-01/TSK-07: an edit rewrites the details, and leaving the due date out
// clears it — the same clearing semantics assignee always had.
func TestTaskDetails_EditTask_RewritesDetailsAndClearsDueDate(t *testing.T) {
	f := setupTskServer(t)
	task := f.createTask(t, map[string]any{
		"title":       "Before",
		"description": "old body",
		"priority":    "low",
		"due_date":    "2026-09-01",
	})

	resp := f.do(t, http.MethodPatch, "/api/v1/tasks/"+task.ID, map[string]any{
		"title":       "After",
		"description": "new body",
		"priority":    "high",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updated tskTaskWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	require.Equal(t, "After", updated.Title)
	require.Equal(t, "new body", updated.Description)
	require.Equal(t, "high", updated.Priority)
	require.Nil(t, updated.DueDate, "an edit without a due date clears it (TSK-07)")
}

// TSK-01/TSK-08: the detail GET returns the full task plus its comments.
func TestTaskDetails_GetTask_ReturnsTaskAndComments(t *testing.T) {
	f := setupTskServer(t)
	task := f.createTask(t, map[string]any{"title": "Detailed", "description": "the body"})

	addResp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/comments",
		map[string]any{"author": "Ada", "body": "перше слово"})
	require.Equal(t, http.StatusCreated, addResp.StatusCode)
	addResp.Body.Close()

	resp := f.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var detail tskDetailWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	require.Equal(t, "the body", detail.Task.Description)
	require.Len(t, detail.Comments, 1)
	require.Equal(t, "Ada", detail.Comments[0].Author)
	require.Equal(t, "перше слово", detail.Comments[0].Body)
	require.Equal(t, task.ID, detail.Comments[0].TaskID)
}

func TestTaskDetails_GetTask_UnknownID_Returns404(t *testing.T) {
	f := setupTskServer(t)

	resp := f.do(t, http.MethodGet, "/api/v1/tasks/"+uuid.NewString(), nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body tskErrorWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}

func TestTaskDetails_GetTask_InvalidID_Returns400(t *testing.T) {
	f := setupTskServer(t)

	resp := f.do(t, http.MethodGet, "/api/v1/tasks/not-a-uuid", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TSK-08: comments post, list oldest first, and carry a real created_at.
func TestTaskDetails_Comments_AddAndListOldestFirst(t *testing.T) {
	f := setupTskServer(t)
	task := f.createTask(t, map[string]any{"title": "Discuss me"})

	for _, body := range []string{"first", "second"} {
		resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/comments",
			map[string]any{"author": "Ada", "body": body})
		var created tskCommentWire
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, created.ID)
		createdAt, err := time.Parse(time.RFC3339Nano, created.CreatedAt)
		require.NoError(t, err, "a comment must come back with a parseable created_at")
		require.False(t, createdAt.IsZero())
	}

	resp := f.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID+"/comments", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var comments []tskCommentWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&comments))
	require.Len(t, comments, 2)
	require.Equal(t, "first", comments[0].Body, "comments read oldest first (TSK-08)")
	require.Equal(t, "second", comments[1].Body)
}

// TSK-09: each rejected shape carries its own documented code, so the UI can
// tell the user which field to fix.
func TestTaskDetails_AddComment_Validation(t *testing.T) {
	f := setupTskServer(t)
	task := f.createTask(t, map[string]any{"title": "Discuss me"})

	longAuthor := make([]rune, 201)
	for i := range longAuthor {
		longAuthor[i] = 'ї'
	}
	longBody := make([]rune, 2001)
	for i := range longBody {
		longBody[i] = 'ї'
	}

	cases := []struct {
		name     string
		payload  map[string]any
		wantCode string
	}{
		{"empty author", map[string]any{"author": "", "body": "text"}, "comment.author_required"},
		{"empty body", map[string]any{"author": "Ada", "body": "   "}, "comment.body_required"},
		{"author too long", map[string]any{"author": string(longAuthor), "body": "text"}, "comment.author_too_long"},
		{"body too long", map[string]any{"author": "Ada", "body": string(longBody)}, "comment.body_too_long"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/comments", tc.payload)
			var body tskErrorWire
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			resp.Body.Close()

			require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
			require.Equal(t, tc.wantCode, body.Error.Code)
		})
	}

	listResp := f.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID+"/comments", nil)
	defer listResp.Body.Close()
	var comments []tskCommentWire
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&comments))
	require.Empty(t, comments, "no rejected comment may have been stored (TSK-09)")
}

// A comment on a task that does not exist is a 404, not a 500 from the FK.
func TestTaskDetails_AddComment_UnknownTask_Returns404(t *testing.T) {
	f := setupTskServer(t)

	resp := f.do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/comments",
		map[string]any{"author": "Ada", "body": "orphan"})
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body tskErrorWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}

// TSK-10: deleting a comment returns 204 and it is gone; deleting it twice
// is a 404.
func TestTaskDetails_DeleteComment(t *testing.T) {
	f := setupTskServer(t)
	task := f.createTask(t, map[string]any{"title": "Discuss me"})

	addResp := f.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/comments",
		map[string]any{"author": "Ada", "body": "bye"})
	var created tskCommentWire
	require.NoError(t, json.NewDecoder(addResp.Body).Decode(&created))
	addResp.Body.Close()

	delResp := f.do(t, http.MethodDelete, "/api/v1/tasks/"+task.ID+"/comments/"+created.ID, nil)
	delResp.Body.Close()
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	listResp := f.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID+"/comments", nil)
	var comments []tskCommentWire
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&comments))
	listResp.Body.Close()
	require.Empty(t, comments)

	againResp := f.do(t, http.MethodDelete, "/api/v1/tasks/"+task.ID+"/comments/"+created.ID, nil)
	var body tskErrorWire
	require.NoError(t, json.NewDecoder(againResp.Body).Decode(&body))
	againResp.Body.Close()
	require.Equal(t, http.StatusNotFound, againResp.StatusCode)
	require.Equal(t, "comment.not_found", body.Error.Code)
}

// The delete route names both ids; a comment must not come off through
// another task's path, or the URL is lying about what it scopes.
func TestTaskDetails_DeleteComment_ThroughAnotherTask_Refused(t *testing.T) {
	f := setupTskServer(t)
	owner := f.createTask(t, map[string]any{"title": "Owns the comment"})
	other := f.createTask(t, map[string]any{"title": "Unrelated"})

	addResp := f.do(t, http.MethodPost, "/api/v1/tasks/"+owner.ID+"/comments",
		map[string]any{"author": "Ada", "body": "mine"})
	var created tskCommentWire
	require.NoError(t, json.NewDecoder(addResp.Body).Decode(&created))
	addResp.Body.Close()

	delResp := f.do(t, http.MethodDelete, "/api/v1/tasks/"+other.ID+"/comments/"+created.ID, nil)
	delResp.Body.Close()
	require.Equal(t, http.StatusNotFound, delResp.StatusCode)

	listResp := f.do(t, http.MethodGet, "/api/v1/tasks/"+owner.ID+"/comments", nil)
	defer listResp.Body.Close()
	var comments []tskCommentWire
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&comments))
	require.Len(t, comments, 1, "the comment must survive a delete through the wrong task")
}

// Listing comments of a task that does not exist is a 404, not an empty list:
// "no comments" and "no such task" are different answers (tasks contract).
func TestTaskDetails_ListComments_UnknownTask_Returns404(t *testing.T) {
	f := setupTskServer(t)

	resp := f.do(t, http.MethodGet, "/api/v1/tasks/"+uuid.NewString()+"/comments", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body tskErrorWire
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "task.not_found", body.Error.Code)
}
