package ports

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// teamEditorToken is the registry bucket key for team-editor connections —
// mirrors infra.Hub's own convention (empty string = no public-link token).
const teamEditorToken = ""

// SSERegistry is the connection-registry port streamBoardEvents/
// streamPublicBoardEvents depend on — satisfied by infra.Hub's Subscribe.
// Deliberately its own small interface (not infra.Hub's Register/Unregister
// directly): Hub's connection id is an unexported type, so it cannot be
// named in a port declared outside the infra package.
type SSERegistry interface {
	// Subscribe registers a new connection under token (empty string for
	// the team-editor bucket) and returns the event channel — closed by the
	// hub on CloseToken(token) (AC-11) — plus a func that unregisters this
	// connection on normal client disconnect.
	Subscribe(token string) (events <-chan Event, unregister func())
}

// SSEHandler serves the live-update routes (contracts/openapi.yaml
// streamBoardEvents, streamPublicBoardEvents): team-editor and public-viewer
// SSE streams of board.state_changed events (contracts/events.md).
type SSEHandler struct {
	registry     SSERegistry
	stateService PublicStateService
}

// NewSSEHandler wires an SSEHandler against the given connection registry
// and the read-only, token-scoped state service used to validate a
// public-viewer token before registering its connection.
func NewSSEHandler(registry SSERegistry, stateService PublicStateService) *SSEHandler {
	return &SSEHandler{registry: registry, stateService: stateService}
}

// RegisterRoutes mounts the SSE routes (contracts/openapi.yaml
// streamBoardEvents, streamPublicBoardEvents), relative to the caller's
// mount point. The caller must keep these off any per-request timeout
// middleware — a timeout would cancel the stream's context mid-flight
// (board.Handler exposes them via RegisterStreamingRoutes for exactly that).
func (h *SSEHandler) RegisterRoutes(r chi.Router) {
	r.Get("/board/events", h.handleStreamBoardEvents)
	r.Get("/public/{token}/events", h.handleStreamPublicBoardEvents)
}

// @Summary  Stream board events (team-editor)
// @Tags     board
// @Produce  text/event-stream
// @Success  200 {string} string
// @Router   /board/events [get]
func (h *SSEHandler) handleStreamBoardEvents(w http.ResponseWriter, r *http.Request) {
	events, unregister := h.registry.Subscribe(teamEditorToken)
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
	// before any connection is registered with the hub.
	if _, err := h.stateService.GetPublicBoardState(r.Context(), token); err != nil {
		httputil.WriteError(w, mapPublicError(err))
		return
	}

	events, unregister := h.registry.Subscribe(token)
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
