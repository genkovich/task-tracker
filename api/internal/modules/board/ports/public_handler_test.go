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
	state  *PublicBoardState
	detail *TaskDetail
}

func (s *stubPublicStateService) GetPublicBoardState(context.Context, string) (*PublicBoardState, error) {
	return s.state, nil
}

func (s *stubPublicStateService) GetPublicTaskDetail(context.Context, string, uuid.UUID) (*TaskDetail, error) {
	return s.detail, nil
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

// TestHandleGetPublicTask_NoStoreAndNoIndexHeaders pins the same two headers
// on the task-detail read (tasks TSK-12): the token in the path is a
// capability URL, so the response must be neither cached nor indexed — the
// board fetch already had this, and a second public route that forgot it
// would leak exactly as much.
func TestHandleGetPublicTask_NoStoreAndNoIndexHeaders(t *testing.T) {
	svc := &stubPublicStateService{detail: &TaskDetail{}}
	h := NewPublicHandler(svc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/public/some-token/tasks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"),
		"public task detail must never be cached by an intermediary")
	require.Equal(t, "noindex, nofollow", rec.Header().Get("X-Robots-Tag"),
		"a capability URL must never end up in a search index")
}

// A non-UUID task id is a caller error, not a 500 from a driver further down.
func TestHandleGetPublicTask_InvalidTaskID_Returns400(t *testing.T) {
	svc := &stubPublicStateService{detail: &TaskDetail{}}
	h := NewPublicHandler(svc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/public/some-token/tasks/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
