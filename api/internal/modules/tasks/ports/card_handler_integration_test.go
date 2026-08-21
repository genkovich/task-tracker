//go:build integration

package ports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"
)

type cardResp struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Assignee     *string `json:"assignee"`
	ColumnStatus string  `json:"column_status"`
}

type cardPageResp struct {
	Items []cardResp `json:"items"`
}

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func setupCardServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	require.NoError(t, database.RunMigrations(migrations.FS, c.DSN))

	repo := infra.NewPostgresCardRepository(c.DB)
	svc := app.NewCardService(repo, app.NewBroadcaster())
	handler := ports.NewCardHandler(svc)

	s := server.New(c.DB, "http://localhost:5173", nil, handler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func postJSON(ts *httptest.Server, path string, body any) *http.Response {
	b, _ := json.Marshal(body)
	resp, _ := http.Post(ts.URL+path, "application/json", bytes.NewReader(b))
	return resp
}

func patchJSON(ts *httptest.Server, path string, body any) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func deleteReq(ts *httptest.Server, path string) *http.Response {
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

// TestCreateCard_NoAuthRequired covers AC-02 and confirms the route carries
// no auth middleware (spec §3 Non-goals) — a plain unauthenticated POST works.
func TestCreateCard_NoAuthRequired(t *testing.T) {
	ts := setupCardServer(t)

	resp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "Write the deck"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created cardResp
	decode(t, resp, &created)
	require.Equal(t, "Write the deck", created.Name)
	require.Equal(t, "todo", created.ColumnStatus)
}

// TestCreateCard_EmptyName covers AC-03.
func TestCreateCard_EmptyName(t *testing.T) {
	ts := setupCardServer(t)

	resp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "   "})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var e errResp
	decode(t, resp, &e)
	require.Equal(t, "tasks.name_required", e.Error.Code)
}

// TestListCards_VisibleAfterCreate covers the "everyone who opens or
// refreshes" clause of AC-02.
func TestListCards_VisibleAfterCreate(t *testing.T) {
	ts := setupCardServer(t)

	postJSON(ts, "/api/v1/cards", map[string]any{"name": "one"}).Body.Close()
	postJSON(ts, "/api/v1/cards", map[string]any{"name": "two"}).Body.Close()

	resp, err := http.Get(ts.URL + "/api/v1/cards")
	require.NoError(t, err)
	var page cardPageResp
	decode(t, resp, &page)
	require.Len(t, page.Items, 2)
}

// TestMoveCard covers AC-01.
func TestMoveCard(t *testing.T) {
	ts := setupCardServer(t)

	createResp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "movable"})
	var created cardResp
	decode(t, createResp, &created)

	moveResp := patchJSON(ts, "/api/v1/cards/"+created.ID+"/column", map[string]any{"column_status": "in_progress"})
	require.Equal(t, http.StatusOK, moveResp.StatusCode)

	var moved cardResp
	decode(t, moveResp, &moved)
	require.Equal(t, "in_progress", moved.ColumnStatus)
}

// TestMoveCard_NotFound covers AC-15's HTTP shape: a move on an
// already-deleted card returns 404, which the client treats as a no-op.
func TestMoveCard_NotFound(t *testing.T) {
	ts := setupCardServer(t)

	resp := patchJSON(ts, "/api/v1/cards/01961234-5678-7000-8000-000000000000/column",
		map[string]any{"column_status": "done"})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var e errResp
	decode(t, resp, &e)
	require.Equal(t, "tasks.card_not_found", e.Error.Code)
}

// TestUpdateCard covers AC-13 and AC-14.
func TestUpdateCard(t *testing.T) {
	ts := setupCardServer(t)

	createResp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "original"})
	var created cardResp
	decode(t, createResp, &created)

	updateResp := patchJSON(ts, "/api/v1/cards/"+created.ID, map[string]any{"name": "renamed"})
	require.Equal(t, http.StatusOK, updateResp.StatusCode)
	var updated cardResp
	decode(t, updateResp, &updated)
	require.Equal(t, "renamed", updated.Name)

	badResp := patchJSON(ts, "/api/v1/cards/"+created.ID, map[string]any{"name": ""})
	require.Equal(t, http.StatusBadRequest, badResp.StatusCode)
}

// TestDeleteCard covers AC-10.
func TestDeleteCard(t *testing.T) {
	ts := setupCardServer(t)

	createResp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "to delete"})
	var created cardResp
	decode(t, createResp, &created)

	delResp := deleteReq(ts, "/api/v1/cards/"+created.ID)
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	resp, _ := http.Get(ts.URL + "/api/v1/cards")
	var page cardPageResp
	decode(t, resp, &page)
	require.Empty(t, page.Items)
}
