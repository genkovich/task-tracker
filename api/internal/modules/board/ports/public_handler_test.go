package ports

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stubPublicStateService is a bare PublicStateService double — this test
// exercises only the handler's own response headers, not state resolution
// (that's public_handler_integration_test.go's job).
type stubPublicStateService struct {
	state *PublicBoardState
}

func (s *stubPublicStateService) GetPublicBoardState(context.Context, string) (*PublicBoardState, error) {
	return s.state, nil
}

// TestHandleGetPublicBoard_NoStoreHeader pins A3's Cache-Control: no-store on
// GET /public/{token}/board: the response must never be cached by an
// intermediary, so a revoked link or a changed board always shows its
// current state to the next viewer, never a stale snapshot.
func TestHandleGetPublicBoard_NoStoreHeader(t *testing.T) {
	svc := &stubPublicStateService{state: &PublicBoardState{BoardID: uuid.New()}}
	h := NewPublicHandler(svc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/public/some-token/board", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"),
		"public board fetch must never be cached by an intermediary")
}
