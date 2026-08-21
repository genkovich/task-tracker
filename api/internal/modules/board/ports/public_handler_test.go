//go:build integration

package ports_test

// RED for T10 (docs/features/board/tasks/T10-ports-public-viewer-routes.md):
// api/internal/modules/board/ports/public_handler.go does not exist yet —
// ports.NewPublicHandler is expected to land there, wrapping
// app.StateService.GetPublicBoardState behind
// GET /api/v1/public/{token}/board (contracts/openapi.yaml operationId
// getPublicBoard).
//
// Covers:
//   - AC-09: a viewer holding a valid public-link token sees the board's
//     current columns and tasks — the PublicBoardState shape, which
//     deliberately has no "public_link" field on the wire (unlike
//     BoardState).
//   - AC-11: an unknown/revoked token returns 404 board.link_invalid, not a
//     stale or partial board state; a token that was valid but has since
//     been revoked behaves identically.
//
// AC-10 (no mutating route under this token-scoped path) is enforced
// structurally by T11's router registration, not by a runtime assertion
// here (T10's own DoD note) — this file only asserts the one GET route.
//
// Deliberately self-contained: the public link under test is issued/revoked
// directly through the already-implemented app.LinkService (T6) rather than
// through T8's HTTP handler, so this RED does not depend on sibling
// in-flight tasks (T8/T9) for its own compile/run status. Type/helper names
// below are prefixed t10PublicHandler* to avoid clashing with sibling
// ports_test files also under active development.
//
// No production code exists yet for ports.NewPublicHandler — this is the
// RED step; the package will not compile until T10 adds it.

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
	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

// seeded board/columns from migrations 000007/000009 (data-model.md,
// CONTEXT.md: exactly one board row).
var t10SeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// t10WireTask/t10WireColumn mirror the Task/Column schemas in
// contracts/openapi.yaml — decoded independently of any internal DTO type
// so this test pins the wire contract, not an implementation detail.
type t10WireTask struct {
	ID       string  `json:"id"`
	ColumnID string  `json:"column_id"`
	Title    string  `json:"title"`
	Assignee *string `json:"assignee"`
}

type t10WireColumn struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Position int           `json:"position"`
	Tasks    []t10WireTask `json:"tasks"`
}

type t10WireErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// t10PublicHandlerFixture bundles the public handler under test (T10) with
// the real repo/link-service, so a valid token can be issued/revoked
// directly at the app layer without depending on T8's HTTP handler.
type t10PublicHandlerFixture struct {
	ts      *httptest.Server
	repo    *infra.PostgresRepository
	linkSvc *app.LinkService
}

func setupT10PublicHandlerServer(t *testing.T) t10PublicHandlerFixture {
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

	// ports.NewPublicHandler does not exist yet (T10) — this line is why
	// the package fails to compile before T10 lands.
	publicHandler := ports.NewPublicHandler(stateSvc)

	r := chi.NewRouter()
	r.Route("/api/v1", publicHandler.RegisterRoutes)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return t10PublicHandlerFixture{ts: ts, repo: repo, linkSvc: linkSvc}
}

// TestGetPublicBoard_ValidToken_ReturnsPublicBoardState covers AC-09: a
// viewer opening a valid public link sees the current columns/tasks, and
// the response never carries a "public_link" field (PublicBoardState,
// openapi.yaml — deliberately distinct from BoardState's shape).
func TestGetPublicBoard_ValidToken_ReturnsPublicBoardState(t *testing.T) {
	f := setupT10PublicHandlerServer(t)
	ctx := context.Background()

	// Seed a task directly through the repo so this test does not depend on
	// POST /api/v1/tasks (T7) existing yet — matches
	// infra/postgres_repo_integration_test.go's own seeding pattern.
	leftmostID, err := f.repo.LeftmostColumnID(ctx, t10SeedBoardID)
	require.NoError(t, err)
	task, err := domain.NewTask(leftmostID, "Write the report", nil)
	require.NoError(t, err)
	require.NoError(t, f.repo.InsertTask(ctx, task))

	link, err := f.linkSvc.IssuePublicLink(ctx, t10SeedBoardID)
	require.NoError(t, err)
	require.NotEmpty(t, link.Token)

	resp, err := http.Get(f.ts.URL + "/api/v1/public/" + link.Token + "/board") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equalf(t, http.StatusOK, resp.StatusCode, "GET /api/v1/public/{token}/board (valid token) status")

	var raw map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))
	_, hasPublicLink := raw["public_link"]
	require.False(t, hasPublicLink, "PublicBoardState must not expose a public_link field (AC-09, openapi.yaml)")

	var columns []t10WireColumn
	require.NoError(t, json.Unmarshal(raw["columns"], &columns))
	require.NotEmpty(t, columns, "expected the seeded columns")

	var foundTask bool
	for _, col := range columns {
		for _, tk := range col.Tasks {
			if tk.Title == "Write the report" {
				foundTask = true
			}
		}
	}
	require.True(t, foundTask, "expected the seeded task to appear in the public board state (AC-09)")
}

// TestGetPublicBoard_UnknownToken_ReturnsLinkInvalid covers AC-11: an
// unknown or revoked token must not show any board state — 404
// board.link_invalid, not a stale/partial view.
func TestGetPublicBoard_UnknownToken_ReturnsLinkInvalid(t *testing.T) {
	f := setupT10PublicHandlerServer(t)

	resp, err := http.Get(f.ts.URL + "/api/v1/public/revoked-or-unknown-token/board") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "GET /api/v1/public/{token}/board (unknown token) status")

	var body t10WireErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.link_invalid", body.Error.Code, "AC-11: unknown/revoked token must map to board.link_invalid")
}

// TestGetPublicBoard_RevokedToken_ReturnsLinkInvalid covers AC-11's
// cross-context case: a token that was valid but has since been revoked
// must behave exactly like an unknown one — 404 board.link_invalid, never a
// stale board state.
func TestGetPublicBoard_RevokedToken_ReturnsLinkInvalid(t *testing.T) {
	f := setupT10PublicHandlerServer(t)
	ctx := context.Background()

	link, err := f.linkSvc.IssuePublicLink(ctx, t10SeedBoardID)
	require.NoError(t, err)

	require.NoError(t, f.linkSvc.RevokePublicLink(ctx, t10SeedBoardID))

	resp, err := http.Get(f.ts.URL + "/api/v1/public/" + link.Token + "/board") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "GET /api/v1/public/{token}/board (revoked token) status")

	var body t10WireErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.link_invalid", body.Error.Code, "AC-11: revoked token must map to board.link_invalid")
}
