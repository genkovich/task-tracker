//go:build integration

package ports_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConcurrency_MoveMove_LastArrivedWins covers AC-07 under genuine
// concurrent load (real goroutines racing over HTTP against a real
// Postgres instance) — not just two sequential calls. The server accepts
// both writes; whichever one Postgres processes last is the state every
// subsequent reader must see.
func TestConcurrency_MoveMove_LastArrivedWins(t *testing.T) {
	ts := setupCardServer(t)

	createResp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "raced"})
	var created cardResp
	decode(t, createResp, &created)

	var wg sync.WaitGroup
	statuses := []string{"in_progress", "done"}
	results := make([]int, len(statuses))
	wg.Add(len(statuses))
	for i, status := range statuses {
		go func(i int, status string) {
			defer wg.Done()
			resp := patchJSON(ts, "/api/v1/cards/"+created.ID+"/column", map[string]any{"column_status": status})
			results[i] = resp.StatusCode
			resp.Body.Close()
		}(i, status)
	}
	wg.Wait()

	for _, code := range results {
		require.Equal(t, http.StatusOK, code, "every concurrent move must be accepted, never rejected (ADR-0002 — no conflict errors)")
	}

	// The final state must be ONE of the two attempted columns — never
	// something else, never a torn write.
	getResp, err := http.Get(ts.URL + "/api/v1/cards")
	require.NoError(t, err)
	var page cardPageResp
	decode(t, getResp, &page)
	require.Len(t, page.Items, 1)
	require.Contains(t, statuses, page.Items[0].ColumnStatus)
}

// TestConcurrency_DeleteMove_DeleteWins covers AC-15 under genuine
// concurrent load: a delete racing a move on the same card. Whichever the
// server actually committed first decides the outcome — the move either
// succeeds (if it landed first) or 404s (if the delete won), but the board
// never ends up with a "resurrected" card after a delete has been accepted.
func TestConcurrency_DeleteMove_DeleteWins(t *testing.T) {
	ts := setupCardServer(t)

	createResp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "raced-for-deletion"})
	var created cardResp
	decode(t, createResp, &created)

	var wg sync.WaitGroup
	var deleteCode, moveCode int
	wg.Add(2)
	go func() {
		defer wg.Done()
		resp := deleteReq(ts, "/api/v1/cards/"+created.ID)
		deleteCode = resp.StatusCode
		resp.Body.Close()
	}()
	go func() {
		defer wg.Done()
		resp := patchJSON(ts, "/api/v1/cards/"+created.ID+"/column", map[string]any{"column_status": "done"})
		moveCode = resp.StatusCode
		resp.Body.Close()
	}()
	wg.Wait()

	require.Equal(t, http.StatusNoContent, deleteCode, "delete must always succeed regardless of race outcome")
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, moveCode,
		"the racing move either lands before the delete (200) or after it (404) — never any other outcome")

	// Whatever the race outcome, the card must be gone by the time both
	// requests have completed — a delete is never silently undone by a
	// racing move (AC-15's core guarantee).
	getResp, err := http.Get(ts.URL + "/api/v1/cards")
	require.NoError(t, err)
	var page cardPageResp
	decode(t, getResp, &page)
	require.Empty(t, page.Items, "the card must not be resurrected by a racing move")
}

// TestConcurrency_ManyEditorsCreatingSimultaneously is a broader smoke test:
// N concurrent create requests all succeed and all end up visible.
func TestConcurrency_ManyEditorsCreatingSimultaneously(t *testing.T) {
	ts := setupCardServer(t)
	const n = 10

	var wg sync.WaitGroup
	codes := make([]int, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			resp := postJSON(ts, "/api/v1/cards", map[string]any{"name": "concurrent card"})
			codes[i] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	for _, c := range codes {
		require.Equal(t, http.StatusCreated, c)
	}

	getResp, err := http.Get(ts.URL + "/api/v1/cards")
	require.NoError(t, err)
	var page cardPageResp
	decode(t, getResp, &page)
	require.Len(t, page.Items, n)
}
