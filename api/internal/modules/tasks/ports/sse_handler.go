package ports

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
)

// sseEventPayload is what a subscriber actually receives — event.Payload is
// an internal id, not meant for the wire; the type name is enough for the
// client to know "refetch" (ADR-0001 keeps the payload minimal by design).
type sseEventPayload struct {
	Type string `json:"type"`
}

// writeSSEEvent writes one `data: ...\n\n` frame and flushes immediately —
// SSE delivery is only useful if it isn't buffered.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, ev app.BoardEvent) {
	data, _ := json.Marshal(sseEventPayload{Type: ev.Type})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// SSEHandler exposes the editor's board-wide event stream — every card and
// link change, unscoped (ADR-0001).
type SSEHandler struct {
	broadcaster *app.Broadcaster
}

func NewSSEHandler(broadcaster *app.Broadcaster) *SSEHandler {
	return &SSEHandler{broadcaster: broadcaster}
}

// RegisterRoutes is a no-op — this handler's only route is long-lived.
func (h *SSEHandler) RegisterRoutes(_ chi.Router) {}

func (h *SSEHandler) RegisterLongLivedRoutes(r chi.Router) {
	r.Get("/cards/events", h.handleSubscribe)
}

func (h *SSEHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	setSSEHeaders(w)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := h.broadcaster.Subscribe()
	defer unsubscribe()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			writeSSEEvent(w, flusher, ev)
		}
	}
}

func httpError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	fmt.Fprint(w, msg)
}
