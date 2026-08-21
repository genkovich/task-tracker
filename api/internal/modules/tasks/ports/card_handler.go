package ports

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

type CardResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Assignee     *string `json:"assignee"`
	ColumnStatus string  `json:"column_status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type CardPageResponse struct {
	Items      []CardResponse `json:"items"`
	HasNext    bool           `json:"has_next"`
	HasPrev    bool           `json:"has_prev"`
	NextCursor *string        `json:"next_cursor"`
}

type CreateCardRequest struct {
	Name     string  `json:"name"`
	Assignee *string `json:"assignee"`
}

type UpdateCardRequest struct {
	Name     string  `json:"name"`
	Assignee *string `json:"assignee"`
}

type MoveCardRequest struct {
	ColumnStatus string `json:"column_status"`
}

func toCardResponse(c *domain.Card) CardResponse {
	return CardResponse{
		ID:           c.ID.String(),
		Name:         c.Name,
		Assignee:     c.Assignee,
		ColumnStatus: c.ColumnStatus,
		CreatedAt:    c.CreatedAt.Format(timeFormat),
		UpdatedAt:    c.UpdatedAt.Format(timeFormat),
	}
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

// CardHandler exposes card CRUD + drag. No route it registers is auth-gated
// — editing has no login by design (spec §3 Non-goals).
type CardHandler struct {
	service *app.CardService
}

func NewCardHandler(service *app.CardService) *CardHandler {
	return &CardHandler{service: service}
}

func (h *CardHandler) RegisterRoutes(r chi.Router) {
	r.Route("/cards", func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Patch("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
		r.Patch("/{id}/column", h.handleMove)
	})
}

func (h *CardHandler) handleList(w http.ResponseWriter, r *http.Request) {
	cards, err := h.service.GetBoard(r.Context())
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	items := make([]CardResponse, 0, len(cards))
	for i := range cards {
		items = append(items, toCardResponse(&cards[i]))
	}

	httputil.WriteJSON(w, CardPageResponse{Items: items, HasNext: false, HasPrev: false}, http.StatusOK)
}

func (h *CardHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	card, err := h.service.CreateCard(r.Context(), req.Name, req.Assignee)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toCardResponse(card), http.StatusCreated)
}

func (h *CardHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid card id")
		return
	}

	var req UpdateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	card, err := h.service.UpdateCard(r.Context(), id, req.Name, req.Assignee)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toCardResponse(card), http.StatusOK)
}

func (h *CardHandler) handleMove(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid card id")
		return
	}

	var req MoveCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	card, err := h.service.MoveCard(r.Context(), id, req.ColumnStatus)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toCardResponse(card), http.StatusOK)
}

func (h *CardHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid card id")
		return
	}

	if err := h.service.DeleteCard(r.Context(), id); err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
