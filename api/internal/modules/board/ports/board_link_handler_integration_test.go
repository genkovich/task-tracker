//go:build integration

package ports_test

// RED for T8 (docs/features/board/tasks/T8-ports-board-link-routes.md):
// api/internal/modules/board/ports/board_handler.go and link_handler.go do
// not exist yet — ports.NewBoardHandler and ports.NewLinkHandler are
// expected to land there, wrapping app.StateService.GetBoardState
// (GET /api/v1/board) and app.LinkService.IssuePublicLink/RevokePublicLink
// (POST/DELETE /api/v1/board/public-link, contracts/openapi.yaml
// operationIds issuePublicLink/revokePublicLink).
//
// Covers:
//   - AC-07: issuing a public link when none is active returns 201 with a
//     token, and a subsequent GET /api/v1/board reflects it; issuing a
//     second one while active returns 409 board.link_already_active.
//   - AC-08: revoking an active public link returns 204 and clears it from
//     board state; revoking with none active returns 404
//     board.link_not_found.
//
// Deliberately self-contained: no shared symbols with sibling ports_test
// files (T9/T10) also under active development. Type/helper names below are
// prefixed t8* to avoid clashing with them.
//
// No production code exists yet for ports.NewBoardHandler /
// ports.NewLinkHandler — this is the RED step; the package will not compile
// until T8 adds them.

import (
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

// seeded board from migration 000007 (data-model.md, CONTEXT.md: exactly one
// board row) — matches infra's postgres_repo_integration_test.go.
var t8SeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// t8PublicLinkResp mirrors the PublicLink schema (contracts/openapi.yaml).
type t8PublicLinkResp struct {
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

// t8BoardStateResp mirrors the BoardState schema (contracts/openapi.yaml) —
// decoded independently of any internal DTO type so this test pins the wire
// contract, not an implementation detail.
type t8BoardStateResp struct {
	Columns    []json.RawMessage `json:"columns"`
	PublicLink *t8PublicLinkResp `json:"public_link"`
}

// t8ErrorResponse mirrors the Error envelope (contracts/openapi.yaml).
type t8ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// t8BoardLinkFixture bundles the board handler (T8, GET /api/v1/board) and
// the link handler under test (T8, POST/DELETE /api/v1/board/public-link)
// behind one router, the way T11's wiring will eventually combine every
// ports/*.go handler.
type t8BoardLinkFixture struct {
	ts *httptest.Server
}

func setupT8BoardLinkServer(t *testing.T) t8BoardLinkFixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()

	stateSvc := app.NewStateService(repo)
	linkSvc := app.NewLinkService(repo, hub)

	// ports.NewBoardHandler / ports.NewLinkHandler do not exist yet (T8) —
	// these lines are why the package fails to compile before T8 lands.
	boardHandler := ports.NewBoardHandler(stateSvc, t8SeedBoardID)
	linkHandler := ports.NewLinkHandler(linkSvc, t8SeedBoardID)

	r := chi.NewRouter()
	boardHandler.RegisterRoutes(r)
	linkHandler.RegisterRoutes(r)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return t8BoardLinkFixture{ts: ts}
}

func (f t8BoardLinkFixture) getBoardState(t *testing.T) (*http.Response, t8BoardStateResp) {
	t.Helper()
	resp, err := http.Get(f.ts.URL + "/api/v1/board") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	var state t8BoardStateResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&state))
	return resp, state
}

// doMethod issues method against path on the fixture's server with no
// request body, returning the *http.Response with its Body left open for
// the caller to decode/close.
func (f t8BoardLinkFixture) doMethod(t *testing.T, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, f.ts.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// TestBoardHandler_GetBoard_ReflectsPublicLinkPresence covers the BoardState
// shape's public_link field in both the absent and present states around
// AC-07's happy path.
func TestBoardHandler_GetBoard_ReflectsPublicLinkPresence(t *testing.T) {
	f := setupT8BoardLinkServer(t)

	getResp, before := f.getBoardState(t)
	require.Equalf(t, http.StatusOK, getResp.StatusCode, "GET /api/v1/board status")
	require.Nil(t, before.PublicLink, "board must have no active public link before AC-07")

	issueResp := f.doMethod(t, http.MethodPost, "/api/v1/board/public-link")
	defer issueResp.Body.Close()
	require.Equalf(t, http.StatusCreated, issueResp.StatusCode, "POST /api/v1/board/public-link (AC-07 happy path) status")

	var issued t8PublicLinkResp
	require.NoError(t, json.NewDecoder(issueResp.Body).Decode(&issued))
	require.NotEmpty(t, issued.Token, "PublicLink.token must be present per openapi.yaml PublicLink schema")

	_, after := f.getBoardState(t)
	require.NotNil(t, after.PublicLink, "GET /api/v1/board must reflect the newly issued public link")
	require.Equal(t, issued.Token, after.PublicLink.Token)
}

// AC-07 (US-05) conflict: issuing a second public link while one is already
// active returns 409 board.link_already_active (openapi.yaml).
func TestLinkHandler_IssuePublicLink_Conflict(t *testing.T) {
	f := setupT8BoardLinkServer(t)

	first := f.doMethod(t, http.MethodPost, "/api/v1/board/public-link")
	require.Equal(t, http.StatusCreated, first.StatusCode)
	first.Body.Close()

	second := f.doMethod(t, http.MethodPost, "/api/v1/board/public-link")
	defer second.Body.Close()
	require.Equal(t, http.StatusConflict, second.StatusCode, "issuing a second public link while one is active must return 409 (AC-07)")

	var body t8ErrorResponse
	require.NoError(t, json.NewDecoder(second.Body).Decode(&body))
	require.Equal(t, "board.link_already_active", body.Error.Code)
}

// AC-08 (US-06) happy path: revoking an active public link returns 204 and
// the board state no longer carries it.
func TestLinkHandler_RevokePublicLink_HappyPath(t *testing.T) {
	f := setupT8BoardLinkServer(t)

	issue := f.doMethod(t, http.MethodPost, "/api/v1/board/public-link")
	require.Equal(t, http.StatusCreated, issue.StatusCode)
	issue.Body.Close()

	revoke := f.doMethod(t, http.MethodDelete, "/api/v1/board/public-link")
	defer revoke.Body.Close()
	require.Equal(t, http.StatusNoContent, revoke.StatusCode, "revoking an active public link must return 204 (AC-08)")

	_, after := f.getBoardState(t)
	require.Nil(t, after.PublicLink, "board state must no longer carry the revoked public link (AC-08)")
}

// AC-08 (US-06) error path: revoking when there is no active public link
// returns 404 board.link_not_found (openapi.yaml).
func TestLinkHandler_RevokePublicLink_NotFound(t *testing.T) {
	f := setupT8BoardLinkServer(t)

	revoke := f.doMethod(t, http.MethodDelete, "/api/v1/board/public-link")
	defer revoke.Body.Close()
	require.Equal(t, http.StatusNotFound, revoke.StatusCode, "revoking with no active public link must return 404 (AC-08)")

	var body t8ErrorResponse
	require.NoError(t, json.NewDecoder(revoke.Body).Decode(&body))
	require.Equal(t, "board.link_not_found", body.Error.Code)
}
