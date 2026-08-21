//go:build integration

package ports_test

import (
	"context"
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

type publicLinkResp struct {
	ID         string  `json:"id"`
	Token      *string `json:"token"`
	DisabledAt *string `json:"disabled_at"`
}

type activeLinkResp struct {
	Link *publicLinkResp `json:"link"`
}

func setupLinkServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	require.NoError(t, database.RunMigrations(migrations.FS, c.DSN))

	repo := infra.NewPostgresPublicLinkRepository(c.DB)
	svc := app.NewPublicLinkService(repo, app.NewBroadcaster())
	handler := ports.NewPublicLinkHandler(svc)

	s := server.New(c.DB, "http://localhost:5173", nil, handler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestGeneratePublicLink covers AC-09.
func TestGeneratePublicLink(t *testing.T) {
	ts := setupLinkServer(t)

	resp := postJSON(ts, "/api/v1/public-links", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var link publicLinkResp
	decode(t, resp, &link)
	require.NotNil(t, link.Token)
	require.NotEmpty(t, *link.Token)
	require.Nil(t, link.DisabledAt)
}

// TestGeneratePublicLink_ReplacesActive covers the "never more than one
// active link" invariant end to end through HTTP.
func TestGeneratePublicLink_ReplacesActive(t *testing.T) {
	ts := setupLinkServer(t)

	postJSON(ts, "/api/v1/public-links", nil).Body.Close()
	postJSON(ts, "/api/v1/public-links", nil).Body.Close()

	resp, err := http.Get(ts.URL + "/api/v1/public-links/active")
	require.NoError(t, err)
	var active activeLinkResp
	decode(t, resp, &active)
	require.NotNil(t, active.Link)
}

// TestGetActivePublicLink_None covers the "no active link" board state.
func TestGetActivePublicLink_None(t *testing.T) {
	ts := setupLinkServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/public-links/active")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var active activeLinkResp
	decode(t, resp, &active)
	require.Nil(t, active.Link)
}

// TestDisablePublicLink covers AC-04.
func TestDisablePublicLink(t *testing.T) {
	ts := setupLinkServer(t)

	genResp := postJSON(ts, "/api/v1/public-links", nil)
	var link publicLinkResp
	decode(t, genResp, &link)

	disResp := postJSON(ts, "/api/v1/public-links/"+link.ID+"/disable", nil)
	require.Equal(t, http.StatusOK, disResp.StatusCode)
	var disabled publicLinkResp
	decode(t, disResp, &disabled)
	require.NotNil(t, disabled.DisabledAt)

	activeResp, err := http.Get(ts.URL + "/api/v1/public-links/active")
	require.NoError(t, err)
	var active activeLinkResp
	decode(t, activeResp, &active)
	require.Nil(t, active.Link)
}

// TestDisablePublicLink_Idempotent covers the idempotent-disable behavior.
func TestDisablePublicLink_Idempotent(t *testing.T) {
	ts := setupLinkServer(t)

	genResp := postJSON(ts, "/api/v1/public-links", nil)
	var link publicLinkResp
	decode(t, genResp, &link)

	postJSON(ts, "/api/v1/public-links/"+link.ID+"/disable", nil).Body.Close()
	resp2 := postJSON(ts, "/api/v1/public-links/"+link.ID+"/disable", nil)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
}
