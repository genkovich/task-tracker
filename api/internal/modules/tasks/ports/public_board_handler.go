package ports

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// PublicBoardHandler serves the token-gated, read-only public view (ADR-0003)
// and its own SSE stream scoped to that one link. A disabled or never-valid
// token gets the platform's plain generic 404 — byte-identical to any
// unmatched route (AC-05) — never a JSON error body, which would itself leak
// that a board once existed at this address.
type PublicBoardHandler struct {
	cardService *app.CardService
	linkService *app.PublicLinkService
	broadcaster *app.Broadcaster
}

func NewPublicBoardHandler(cardService *app.CardService, linkService *app.PublicLinkService, broadcaster *app.Broadcaster) *PublicBoardHandler {
	return &PublicBoardHandler{cardService: cardService, linkService: linkService, broadcaster: broadcaster}
}

func (h *PublicBoardHandler) RegisterRoutes(r chi.Router) {
	r.Route("/public/boards/{token}", func(r chi.Router) {
		r.Get("/", h.handleGet)
		r.Get("/events", h.handleSubscribe)
	})
}

func (h *PublicBoardHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if _, err := h.linkService.ResolvePublicLink(r.Context(), token); err != nil {
		http.NotFound(w, r)
		return
	}

	cards, err := h.cardService.GetBoard(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]CardResponse, 0, len(cards))
	for i := range cards {
		items = append(items, toCardResponse(&cards[i]))
	}
	httputil.WriteJSON(w, CardPageResponse{Items: items, HasNext: false, HasPrev: false}, http.StatusOK)
}

func (h *PublicBoardHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	link, err := h.linkService.ResolvePublicLink(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}

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
			if ev.Type == app.EventLinkDisabled && payloadMatches(ev.Payload, link.ID) {
				// This viewer's own link just got disabled — deliver one
				// final message, then close the stream (AC-12).
				writeSSEEvent(w, flusher, ev)
				return
			}
			writeSSEEvent(w, flusher, ev)
		}
	}
}

func payloadMatches(payload any, id uuid.UUID) bool {
	got, ok := payload.(uuid.UUID)
	return ok && got == id
}
