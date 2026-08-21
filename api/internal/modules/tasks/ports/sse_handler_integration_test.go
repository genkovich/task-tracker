//go:build integration

package ports_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"
)

type taskServer struct {
	ts          *httptest.Server
	cardSvc     *app.CardService
	linkSvc     *app.PublicLinkService
	broadcaster *app.Broadcaster
}

func setupFullServer(t *testing.T) *taskServer {
	t.Helper()
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	require.NoError(t, database.RunMigrations(migrations.FS, c.DSN))

	broadcaster := app.NewBroadcaster()
	cardRepo := infra.NewPostgresCardRepository(c.DB)
	linkRepo := infra.NewPostgresPublicLinkRepository(c.DB)
	cardSvc := app.NewCardService(cardRepo, broadcaster)
	linkSvc := app.NewPublicLinkService(linkRepo, broadcaster)

	cardHandler := ports.NewCardHandler(cardSvc)
	linkHandler := ports.NewPublicLinkHandler(linkSvc)
	sseHandler := ports.NewSSEHandler(broadcaster)
	boardHandler := ports.NewPublicBoardHandler(cardSvc, linkSvc, broadcaster)

	s := server.New(c.DB, "http://localhost:5173", nil, cardHandler, linkHandler, sseHandler, boardHandler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return &taskServer{ts: ts, cardSvc: cardSvc, linkSvc: linkSvc, broadcaster: broadcaster}
}

// readOneSSELine reads until the next non-empty "data:" line or times out.
func readOneSSELine(t *testing.T, r *bufio.Reader, timeout time.Duration) string {
	t.Helper()
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ch <- result{"", err}
				return
			}
			if strings.HasPrefix(line, "data:") {
				ch <- result{line, nil}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		require.NoError(t, res.err)
		return res.line
	case <-time.After(timeout):
		t.Fatal("timed out waiting for SSE data line")
		return ""
	}
}

// TestPublicBoard_DisabledLink_ByteIdenticalNotFound covers AC-05: the
// response for a never-valid token must be byte-identical to the platform's
// generic not-found for any unmatched route.
func TestPublicBoard_NeverValidToken_ByteIdenticalNotFound(t *testing.T) {
	env := setupFullServer(t)

	genericResp, err := http.Get(env.ts.URL + "/this/route/does/not/exist")
	require.NoError(t, err)
	genericBody := readAll(t, genericResp)

	tokenResp, err := http.Get(env.ts.URL + "/api/v1/public/boards/never-existed")
	require.NoError(t, err)
	tokenBody := readAll(t, tokenResp)

	require.Equal(t, genericResp.StatusCode, tokenResp.StatusCode)
	require.Equal(t, genericResp.Header.Get("Content-Type"), tokenResp.Header.Get("Content-Type"))
	require.Equal(t, genericBody, tokenBody)
}

// TestPublicBoard_DisabledToken_ByteIdenticalNotFound covers the other half
// of AC-05: a token that WAS valid, then got disabled, must resolve exactly
// like the never-valid case above — never a signal that it once worked.
func TestPublicBoard_DisabledToken_ByteIdenticalNotFound(t *testing.T) {
	env := setupFullServer(t)

	link, err := env.linkSvc.GenerateLink(context.Background())
	require.NoError(t, err)
	require.NoError(t, env.linkSvc.DisableLink(context.Background(), link.ID))

	genericResp, err := http.Get(env.ts.URL + "/this/route/does/not/exist")
	require.NoError(t, err)
	genericBody := readAll(t, genericResp)

	tokenResp, err := http.Get(env.ts.URL + "/api/v1/public/boards/" + link.Token)
	require.NoError(t, err)
	tokenBody := readAll(t, tokenResp)

	require.Equal(t, genericResp.StatusCode, tokenResp.StatusCode)
	require.Equal(t, genericResp.Header.Get("Content-Type"), tokenResp.Header.Get("Content-Type"))
	require.Equal(t, genericBody, tokenBody)
}

// TestPublicBoard_ActiveLink covers AC-06/AC-08: an active link resolves the
// board's current state, read-only.
func TestPublicBoard_ActiveLink(t *testing.T) {
	env := setupFullServer(t)

	link, err := env.linkSvc.GenerateLink(context.Background())
	require.NoError(t, err)
	_, err = env.cardSvc.CreateCard(context.Background(), "on the board", nil)
	require.NoError(t, err)

	resp, err := http.Get(env.ts.URL + "/api/v1/public/boards/" + link.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var page cardPageResp
	decode(t, resp, &page)
	require.Len(t, page.Items, 1)
}

// TestCardsEvents_ReceivesCardMoved covers the editor-side SSE stream.
func TestCardsEvents_ReceivesCardMoved(t *testing.T) {
	env := setupFullServer(t)

	req, _ := http.NewRequest(http.MethodGet, env.ts.URL+"/api/v1/cards/events", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	created, err := env.cardSvc.CreateCard(context.Background(), "watch me", nil)
	require.NoError(t, err)

	reader := bufio.NewReader(resp.Body)
	line := readOneSSELine(t, reader, 5*time.Second)
	require.Contains(t, line, "card.created")

	_, err = env.cardSvc.MoveCard(context.Background(), created.ID, "in_progress")
	require.NoError(t, err)
	line2 := readOneSSELine(t, reader, 5*time.Second)
	require.Contains(t, line2, "card.moved")
}

// TestPublicBoardEvents_ClosesOnOwnLinkDisabled covers AC-12: an open
// viewer's SSE stream ends when its own link is disabled.
func TestPublicBoardEvents_ClosesOnOwnLinkDisabled(t *testing.T) {
	env := setupFullServer(t)

	link, err := env.linkSvc.GenerateLink(context.Background())
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, env.ts.URL+"/api/v1/public/boards/"+link.Token+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.NoError(t, env.linkSvc.DisableLink(context.Background(), link.ID))

	reader := bufio.NewReader(resp.Body)
	// Read the final link.disabled frame, then confirm the stream closes.
	line := readOneSSELine(t, reader, 5*time.Second)
	require.Contains(t, line, "link.disabled")

	done := make(chan struct{})
	go func() {
		_, _ = reader.ReadString('\n') // blocks until EOF once the handler returns
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not close after the viewer's own link was disabled")
	}
}

func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n])
}
