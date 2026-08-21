package ports

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

type PublicLinkResponse struct {
	ID         string  `json:"id"`
	Token      *string `json:"token,omitempty"`
	DisabledAt *string `json:"disabled_at"`
	CreatedAt  string  `json:"created_at"`
}

type ActiveLinkResponse struct {
	Link *PublicLinkResponse `json:"link"`
}

func toPublicLinkResponse(l *domain.PublicLink, includeToken bool) PublicLinkResponse {
	resp := PublicLinkResponse{
		ID:        l.ID.String(),
		CreatedAt: l.CreatedAt.Format(timeFormat),
	}
	if includeToken {
		resp.Token = &l.Token
	}
	if l.DisabledAt != nil {
		s := l.DisabledAt.Format(timeFormat)
		resp.DisabledAt = &s
	}
	return resp
}

// PublicLinkHandler exposes generate/disable/get-active. No route it
// registers is auth-gated (spec §3 Non-goals).
type PublicLinkHandler struct {
	service *app.PublicLinkService
}

func NewPublicLinkHandler(service *app.PublicLinkService) *PublicLinkHandler {
	return &PublicLinkHandler{service: service}
}

func (h *PublicLinkHandler) RegisterRoutes(r chi.Router) {
	r.Route("/public-links", func(r chi.Router) {
		r.Post("/", h.handleGenerate)
		r.Get("/active", h.handleGetActive)
		r.Post("/{id}/disable", h.handleDisable)
	})
}

func (h *PublicLinkHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	link, err := h.service.GenerateLink(r.Context())
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}
	httputil.WriteJSON(w, toPublicLinkResponse(link, true), http.StatusCreated)
}

func (h *PublicLinkHandler) handleGetActive(w http.ResponseWriter, r *http.Request) {
	link, err := h.service.GetActiveLink(r.Context())
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}
	if link == nil {
		httputil.WriteJSON(w, ActiveLinkResponse{Link: nil}, http.StatusOK)
		return
	}
	resp := toPublicLinkResponse(link, false)
	httputil.WriteJSON(w, ActiveLinkResponse{Link: &resp}, http.StatusOK)
}

func (h *PublicLinkHandler) handleDisable(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid link id")
		return
	}

	if err := h.service.DisableLink(r.Context(), id); err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	link, err := h.service.GetLink(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}
	httputil.WriteJSON(w, toPublicLinkResponse(link, false), http.StatusOK)
}
