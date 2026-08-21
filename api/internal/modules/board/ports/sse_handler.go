package ports

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// teamEditorToken is the registry bucket key for team-editor connections —
// mirrors infra.Hub's own convention (empty string = no public-link token).
const teamEditorToken = ""

// SSEBoardStateService validates a board id before its team-editor SSE
// connection is registered (an unknown board must 404, not stream forever) —
// satisfied by app.StateService.
type SSEBoardStateService interface {
	GetBoardState(ctx context.Context, boardID uuid.UUID) (*BoardState, error)
}

// SSERegistry is the connection-registry port streamBoardEvents/
// streamPublicBoardEvents depend on — satisfied by infra.Hub's Subscribe.
// Deliberately its own small interface (not infra.Hub's Register/Unregister
// directly): Hub's connection id is an unexported type, so it cannot be
// named in a port declared outside the infra package.
type SSERegistry interface {
	// Subscribe registers a new connection under boardID's token bucket
	// (empty string for the team-editor bucket) and returns the event
	// channel — closed by the hub on CloseToken (AC-11) — plus a func that
	// unregisters this connection on normal client disconnect.
	Subscribe(boardID uuid.UUID, token string) (events <-chan Event, unregister func())
}

// SSEHandler serves the live-update routes (boards contract
// streamBoardEvents, base contract streamPublicBoardEvents): per-board
// team-editor and public-viewer SSE streams of board.state_changed events
// (contracts/events.md), scoped so board A's events never reach board B's
// connections (boards BRD-05).
type SSEHandler struct {
	registry      SSERegistry
	stateService  PublicStateService
	boardStateSvc SSEBoardStateService
}

// NewSSEHandler wires an SSEHandler against the given connection registry
// and the state services used to validate a public-viewer token / a board id
// before registering a connection.
func NewSSEHandler(registry SSERegistry, stateService PublicStateService, boardStateSvc SSEBoardStateService) *SSEHandler {
	return &SSEHandler{registry: registry, stateService: stateService, boardStateSvc: boardStateSvc}
}

// RegisterRoutes mounts the team-editor SSE stream (boards contract
// streamBoardEvents), relative to the caller's mount point. The caller must
// keep this off any per-request timeout middleware — a timeout would cancel
// the stream's context mid-flight (board.Handler exposes it via
// RegisterStreamingRoutes for exactly that). The public-viewer stream is
// registered separately — see RegisterPublicRoutes.
func (h *SSEHandler) RegisterRoutes(r chi.Router) {
	r.Get("/boards/{boardId}/events", h.handleStreamBoardEvents)
}

// RegisterPublicRoutes mounts the public-viewer SSE stream (base contract
// streamPublicBoardEvents). Kept separate from RegisterRoutes so board.Handler
// can mount it on the server's high-traffic rate-limit subtree
// (server.HighTrafficRouteRegistrar) instead of the standard one — many
// public-link viewers behind a handful of shared IPs may open it at once.
// Like the team-editor stream, it must stay off any per-request timeout.
func (h *SSEHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/public/{token}/events", h.handleStreamPublicBoardEvents)
}

// @Summary  Stream board events (team-editor)
// @Tags     board
// @Produce  text/event-stream
// @Param    boardId path string true "Board id"
// @Success  200 {string} string
// @Failure  404 {object} httputil.ErrorResponse
// @Router   /boards/{boardId}/events [get]
func (h *SSEHandler) handleStreamBoardEvents(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseBoardID(w, r)
	if !ok {
		return
	}

	// An unknown board must 404 before any connection is registered —
	// otherwise the stream would sit alive forever on a board that doesn't
	// exist (boards contract streamBoardEvents).
	if _, err := h.boardStateSvc.GetBoardState(r.Context(), boardID); err != nil {
		httputil.WriteError(w, mapBoardError(err))
		return
	}

	events, unregister := h.registry.Subscribe(boardID, teamEditorToken)
	defer unregister()

	stream(w, r, events)
}

// @Summary  Stream public board events (viewer)
// @Tags     public
// @Produce  text/event-stream
// @Param    token path string true "Public link token"
// @Success  200   {string} string
// @Failure  404   {object} httputil.ErrorResponse
// @Router   /public/{token}/events [get]
func (h *SSEHandler) handleStreamPublicBoardEvents(w http.ResponseWriter, r *http.Request) {
	setNoIndexHeader(w)
	token := chi.URLParam(r, "token")

	// Delegate token validity to app.StateService (T6) — AC-11 requires an
	// unknown/revoked token to be rejected with 404 board.link_invalid
	// before any connection is registered with the hub. The state also
	// carries the link's board id — the hub bucket this viewer joins
	// (boards BRD-05).
	state, err := h.stateService.GetPublicBoardState(r.Context(), token)
	if err != nil {
		httputil.WriteError(w, mapPublicError(err))
		return
	}

	events, unregister := h.registry.Subscribe(state.BoardID, token)
	defer unregister()

	// Re-check after registering: a revoke landing between the validation
	// above and Subscribe finds no bucket to close (CloseToken is a no-op on
	// an unknown token), which would leave this fresh connection immortal —
	// the heartbeat keeps it alive and no later CloseToken ever sees it.
	if _, err := h.stateService.GetPublicBoardState(r.Context(), token); err != nil {
		httputil.WriteError(w, mapPublicError(err))
		return
	}

	stream(w, r, events)
}

// heartbeatInterval paces the SSE keepalive comments: often enough to stop
// idle-connection reapers (proxies, LBs) from cutting a quiet stream, rare
// enough to cost nothing.
const heartbeatInterval = 20 * time.Second

// stream writes SSE headers, then relays every event off events onto w until
// events closes (hub.CloseToken — AC-11) or the client disconnects
// (r.Context().Done()). Between events it emits an SSE comment line as a
// heartbeat so intermediaries never see the connection as idle.
func stream(w http.ResponseWriter, r *http.Request, events <-chan Event) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteError(w, &apperr.Error{
			Code:       "board.streaming_unsupported",
			Message:    "this server does not support streaming responses",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case evt, ok := <-events:
			if !ok {
				// The hub closed this connection's channel — a public-link
				// revoke (AC-11). Ending the handler ends the response
				// stream synchronously.
				return
			}
			writeSSEEvent(w, evt)
			flusher.Flush()
		case <-heartbeat.C:
			// SSE comment line — ignored by EventSource, keeps the
			// connection visibly alive for proxies.
			_, _ = w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// writeSSEEvent writes evt as one named SSE message: an "event:" line
// carrying evt.EventType — clients subscribe by name via
// EventSource.addEventListener, and an unnamed message would never fire
// their listener — then the "data:" line (contracts/events.md
// board.state_changed.v1 shape) and the blank line that terminates an SSE
// message.
func writeSSEEvent(w http.ResponseWriter, evt Event) {
	payload, err := json.Marshal(evt)
	if err != nil {
		// evt is a fixed, always-marshalable struct — unreachable in
		// practice; skip this one event rather than break the stream.
		return
	}
	_, _ = w.Write([]byte("event: " + evt.EventType + "\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(payload)
	_, _ = w.Write([]byte("\n\n"))
}
