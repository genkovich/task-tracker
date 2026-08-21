//go:build integration

package board_test

// Пінінг рев'ю 2026-08-21, корінь A: board раніше монтувався на root router
// повз server.New і тому обходив rate limiter і metrics middleware, а SSE
// різався per-request timeout'ом. Ці тести йдуть через справжній server.New
// з board.New(db) у opts — точнісінько як cmd/api/main.go — і пінять:
//   - board-роути видимі HTTP-метрикам (http_requests_total по route-патерну)
//   - board-роути під загальним rate limit 60/хв (61-й запит → 429)
//   - SSE-стрім живе довше за per-request timeout і після нього все ще
//     доставляє події (streaming-група без middleware.Timeout)

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"
)

// wiringSeedBoardID is the seeded "first board" (migration 000007).
var wiringSeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// setupBoardServer wires board through server.New exactly the way
// cmd/api/main.go does (no root-router side mount), with opts appended.
func setupBoardServer(t *testing.T, opts ...any) *httptest.Server {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	serverOpts := append([]any{board.New(c.DB)}, opts...)
	s := server.New(c.DB, "*", nil, serverOpts...)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestBoardWiring_MetricsAndRateLimit pins two halves of root A: board
// routes must pass through the server's metrics middleware (the review found
// zero series) and through the general 60/min rate limit (the review proved
// 81×200 in a row).
func TestBoardWiring_MetricsAndRateLimit(t *testing.T) {
	ts := setupBoardServer(t)

	// 60 requests fit the budget; the 61st within the same minute must be
	// throttled — before the fix board bypassed the limiter entirely.
	for i := range 60 {
		resp, err := http.Get(ts.URL + "/api/v1/boards/" + wiringSeedBoardID.String()) //nolint:noctx // test helper
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "GET /api/v1/boards/{boardId} #%d within the 60/min budget", i+1)
	}

	resp, err := http.Get(ts.URL + "/api/v1/boards/" + wiringSeedBoardID.String()) //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
		"the 61st GET /api/v1/boards/{boardId} within a minute must hit the general rate limit")

	// /metrics is mounted outside the limiter, so this scrape always works.
	metricsResp, err := http.Get(ts.URL + "/metrics") //nolint:noctx // test helper
	require.NoError(t, err)
	defer metricsResp.Body.Close()
	require.Equal(t, http.StatusOK, metricsResp.StatusCode)

	raw, err := io.ReadAll(metricsResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(raw), `http_requests_total{method="GET",route="/api/v1/boards/{boardId}"`,
		"board requests must be recorded by the HTTP metrics middleware")
}

// TestBoardWiring_SSEOutlivesRequestTimeout pins the streaming half of root
// A: with a deliberately short per-request timeout the SSE stream must stay
// open past it and still deliver events — proof the SSE routes live outside
// middleware.Timeout while ordinary routes keep it.
func TestBoardWiring_SSEOutlivesRequestTimeout(t *testing.T) {
	const requestTimeout = 500 * time.Millisecond
	ts := setupBoardServer(t, server.WithRequestTimeout(requestTimeout))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/v1/boards/"+wiringSeedBoardID.String()+"/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /api/v1/boards/{boardId}/events status")

	lines := make(chan string, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	// Outlive the request timeout threefold before provoking any event: a
	// stream still under middleware.Timeout would be cancelled here.
	select {
	case <-done:
		t.Fatalf("SSE stream closed within %v — the per-request timeout must not apply to streaming routes", 3*requestTimeout)
	case <-time.After(3 * requestTimeout):
	}

	// A mutation broadcasts board.state_changed; the long-lived stream must
	// still be subscribed and deliver it.
	body, err := json.Marshal(map[string]any{"board_id": wiringSeedBoardID.String(), "title": "outlive the timeout"})
	require.NoError(t, err)
	createResp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body)) //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, createResp.Body)
	createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-lines:
			if strings.HasPrefix(line, "data:") && strings.Contains(line, "board.state_changed") {
				return // stream alive well past the timeout and still delivering
			}
		case <-done:
			t.Fatal("SSE stream closed before delivering the broadcast event")
		case <-deadline:
			t.Fatal("no board.state_changed arrived on a stream older than the request timeout")
		}
	}
}

// TestPublicRoutes_MutatingMethodsRefused covers AC-10 against the fully
// wired server (review 2026-08-21 re2, #10): the earlier version ran on a
// fixture that registered only the public handler, so its 404s were
// guaranteed by the empty router — a tautology. Here server.New carries
// every board route (task CRUD included), the GET sanity check proves the
// public path itself is live on this router, and every mutating method
// under /api/v1/public/{token}/... is still refused by routing (405 on the
// read-only board path, 404 on paths that are not registered), never
// executed — even with a VALID token.
func TestPublicRoutes_MutatingMethodsRefused(t *testing.T) {
	ts := setupBoardServer(t)
	ctx := context.Background()

	// Issue a real public link over the team-editor HTTP route.
	linkResp, err := http.Post(ts.URL+"/api/v1/boards/"+wiringSeedBoardID.String()+"/public-link", "application/json", nil) //nolint:noctx // test helper
	require.NoError(t, err)
	var link struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(linkResp.Body).Decode(&link))
	linkResp.Body.Close()
	require.Equal(t, http.StatusCreated, linkResp.StatusCode)
	require.NotEmpty(t, link.Token)

	base := ts.URL + "/api/v1/public/" + link.Token

	// Sanity: the read-only route answers on this very server, so the
	// refusals below prove routing policy, not an unregistered path.
	getResp, err := http.Get(base + "/board") //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, getResp.Body)
	getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode, "GET public board must work with a valid token")

	cases := []struct {
		method string
		url    string
	}{
		{http.MethodPost, base + "/board"},
		{http.MethodPatch, base + "/board"},
		{http.MethodDelete, base + "/board"},
		{http.MethodPost, base + "/tasks"},
		{http.MethodDelete, base + "/tasks/" + uuid.NewString()},
	}

	for _, tc := range cases {
		req, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		require.Containsf(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode,
			"AC-10: %s %s must be refused by routing, got %d", tc.method, tc.url, resp.StatusCode)
	}
}

// TestPublicBoardWiring_HighTrafficRateLimit pins A3: the public-viewer board
// fetch sits on its own 300/min tier (server.HighTrafficRouteRegistrar), not
// the general 60/min one — a room of phones behind one venue Wi-Fi opening a
// public link in the same minute must not 429 past the old 60/min ceiling,
// only past the new 300/min one.
func TestPublicBoardWiring_HighTrafficRateLimit(t *testing.T) {
	ts := setupBoardServer(t)

	linkResp, err := http.Post(ts.URL+"/api/v1/boards/"+wiringSeedBoardID.String()+"/public-link", "application/json", nil) //nolint:noctx // test helper
	require.NoError(t, err)
	var link struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(linkResp.Body).Decode(&link))
	linkResp.Body.Close()
	require.Equal(t, http.StatusCreated, linkResp.StatusCode)

	url := ts.URL + "/api/v1/public/" + link.Token + "/board"

	// 1..300: the 61st onward is well past the old 60/min ceiling, still
	// inside the new 300/min one — none of these 300 requests may be
	// throttled.
	for i := 1; i <= 300; i++ {
		resp, err := http.Get(url) //nolint:noctx // test helper
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode,
			"GET public board #%d must fit the 300/min high-traffic budget, not the old 60/min one", i)
	}

	// The 301st within the same minute must hit the high-traffic ceiling —
	// proof this is a real (finite) limit, not an accidental bypass.
	resp, err := http.Get(url) //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
		"the 301st GET public board within a minute must hit the high-traffic limit")
}

// issuePublicLink issues a link over the team-editor route and returns its
// token.
func issuePublicLink(t *testing.T, ts *httptest.Server, boardID uuid.UUID) string {
	t.Helper()

	resp, err := http.Post(ts.URL+"/api/v1/boards/"+boardID.String()+"/public-link", "application/json", nil) //nolint:noctx // test helper
	require.NoError(t, err)
	var link struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&link))
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, link.Token)
	return link.Token
}

// createTaskOnBoard creates a task through the real HTTP route and returns
// its id.
func createTaskOnBoard(t *testing.T, ts *httptest.Server, boardID uuid.UUID, payload map[string]any) string {
	t.Helper()

	payload["board_id"] = boardID.String()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(raw)) //nolint:noctx // test helper
	require.NoError(t, err)
	var task struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&task))
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return task.ID
}

// createBoard creates a second board through the dashboard route.
func createBoard(t *testing.T, ts *httptest.Server, name string) uuid.UUID {
	t.Helper()

	raw, err := json.Marshal(map[string]any{"name": name})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/boards", "application/json", bytes.NewReader(raw)) //nolint:noctx // test helper
	require.NoError(t, err)
	var board struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&board))
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return board.ID
}

// TestPublicTaskDetail_ViewerSeesDetailsWithoutLeaks covers tasks TSK-12: the
// viewer's detail read serves the same content as the editor's, and nothing
// beyond it — no board id, no public link.
func TestPublicTaskDetail_ViewerSeesDetailsWithoutLeaks(t *testing.T) {
	ts := setupBoardServer(t)
	token := issuePublicLink(t, ts, wiringSeedBoardID)
	taskID := createTaskOnBoard(t, ts, wiringSeedBoardID, map[string]any{
		"title":       "Public detail",
		"description": "visible to the viewer",
		"priority":    "high",
		"due_date":    "2026-09-01",
	})

	resp, err := http.Get(ts.URL + "/api/v1/public/" + token + "/tasks/" + taskID) //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	require.Equal(t, "noindex, nofollow", resp.Header.Get("X-Robots-Tag"))

	var raw map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))
	require.Contains(t, raw, "task")
	require.Contains(t, raw, "comments")
	require.NotContains(t, raw, "board_id", "the viewer must not learn the board id")
	require.NotContains(t, raw, "public_link")

	var task struct {
		Description string  `json:"description"`
		Priority    string  `json:"priority"`
		DueDate     *string `json:"due_date"`
	}
	require.NoError(t, json.Unmarshal(raw["task"], &task))
	require.Equal(t, "visible to the viewer", task.Description)
	require.Equal(t, "high", task.Priority)
	require.NotNil(t, task.DueDate)
	require.Equal(t, "2026-09-01", *task.DueDate)
}

// TestPublicTaskDetail_TaskOfAnotherBoard_Refused covers tasks TSK-13 — the
// feature's central authorization property. Without the board check, a single
// public link would read every task in the product, since task ids are global
// and the route is unauthenticated.
func TestPublicTaskDetail_TaskOfAnotherBoard_Refused(t *testing.T) {
	ts := setupBoardServer(t)
	token := issuePublicLink(t, ts, wiringSeedBoardID)

	otherBoardID := createBoard(t, ts, "Board B")
	foreignTaskID := createTaskOnBoard(t, ts, otherBoardID, map[string]any{
		"title":       "Board B secret",
		"description": "must not leak through board A's link",
	})

	resp, err := http.Get(ts.URL + "/api/v1/public/" + token + "/tasks/" + foreignTaskID) //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"a task of another board must be refused through this token (TSK-13)")

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "Board B secret", "the refusal must not echo the task")
	require.Contains(t, string(raw), "board.link_invalid",
		"the refusal must be indistinguishable from an unknown token")
}

// TestPublicTaskDetail_HighTrafficRateLimit covers spec §6.1: the detail read
// sits in the 300/min tier next to the board fetch — a room of phones opening
// cards would exhaust the general 60/min budget within seconds otherwise.
func TestPublicTaskDetail_HighTrafficRateLimit(t *testing.T) {
	ts := setupBoardServer(t)
	token := issuePublicLink(t, ts, wiringSeedBoardID)
	taskID := createTaskOnBoard(t, ts, wiringSeedBoardID, map[string]any{"title": "Opened by everyone"})

	url := ts.URL + "/api/v1/public/" + token + "/tasks/" + taskID

	// Past the old 60/min ceiling and still inside the new one: none of these
	// may be throttled.
	for i := 1; i <= 200; i++ {
		resp, err := http.Get(url) //nolint:noctx // test helper
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode,
			"GET public task detail #%d must fit the 300/min high-traffic budget, not the old 60/min one", i)
	}
}

// TSK-12: there is no comment route under a public token — refused by
// routing, not by a check inside a handler that could be forgotten.
func TestPublicTaskDetail_NoCommentRoutesUnderToken(t *testing.T) {
	ts := setupBoardServer(t)
	token := issuePublicLink(t, ts, wiringSeedBoardID)
	taskID := createTaskOnBoard(t, ts, wiringSeedBoardID, map[string]any{"title": "Read only"})
	ctx := context.Background()

	base := ts.URL + "/api/v1/public/" + token + "/tasks/" + taskID

	cases := []struct {
		method string
		url    string
	}{
		{http.MethodPost, base + "/comments"},
		{http.MethodGet, base + "/comments"},
		{http.MethodDelete, base + "/comments/" + uuid.NewString()},
		{http.MethodPatch, base},
		{http.MethodDelete, base},
	}

	for _, tc := range cases {
		req, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		require.Containsf(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode,
			"TSK-12: %s %s must be refused by routing, got %d", tc.method, tc.url, resp.StatusCode)
	}
}

// TSK-01 + spec §6: the board state carries the card fields and never a
// description body. Asserted on the raw JSON — a typed decode would silently
// ignore a field that crept back in, which is exactly the regression this
// guards against.
func TestBoardState_CarriesCardFieldsNotDescriptionBody(t *testing.T) {
	ts := setupBoardServer(t)
	taskID := createTaskOnBoard(t, ts, wiringSeedBoardID, map[string]any{
		"title":       "Described",
		"description": "a body nobody wants on every refetch",
		"priority":    "low",
	})

	resp, err := http.Get(ts.URL + "/api/v1/boards/" + wiringSeedBoardID.String()) //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(body), "a body nobody wants on every refetch",
		"the board state must not carry description bodies (spec §6)")

	var state struct {
		Columns []struct {
			Tasks []map[string]json.RawMessage `json:"tasks"`
		} `json:"columns"`
	}
	require.NoError(t, json.Unmarshal(body, &state))

	var card map[string]json.RawMessage
	for _, col := range state.Columns {
		for _, task := range col.Tasks {
			var id string
			require.NoError(t, json.Unmarshal(task["id"], &id))
			if id == taskID {
				card = task
			}
		}
	}
	require.NotNil(t, card, "the created task must appear on the board")
	require.NotContains(t, card, "description")
	require.Contains(t, card, "has_description")
	require.Contains(t, card, "comment_count")
	require.Contains(t, card, "priority")
	require.Contains(t, card, "due_date")
}
