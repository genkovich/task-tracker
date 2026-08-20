package ports

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/user/app"
	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
)

type Handler struct {
	service *app.Service
	storage storage.ObjectStorage
}

func NewHandler(service *app.Service, st storage.ObjectStorage) *Handler {
	return &Handler{service: service, storage: st}
}

func (h *Handler) RegisterRoutes(_ chi.Router) {}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Put("/me", h.handleUpdateProfile)
		r.Post("/me/avatar", h.handleUploadAvatar)
		r.Delete("/me/avatar", h.handleDeleteAvatar)
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (*authmw.Claims, bool) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return nil, false
	}
	if !claims.IsAdmin() {
		httputil.WriteError(w, mapError(domain.ErrForbidden))
		return nil, false
	}
	return claims, true
}

// @Summary     Create user
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body     CreateUserRequest true "Create user"
// @Success     201     {object} UserResponse
// @Failure     400     {object} httputil.ErrorResponse
// @Failure     403     {object} httputil.ErrorResponse
// @Failure     409     {object} httputil.ErrorResponse
// @Router      /users [post]
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	if req.Email == "" {
		httputil.WriteValidationError(w, "validation.email_required", "email is required")
		return
	}

	user, err := h.service.Create(r.Context(), req.Email, req.FirstName, req.LastName, req.Role)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toUserResponse(user), http.StatusCreated)
}

// @Summary     List users
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query    int    false "Page size (1-100)" default(20)
// @Param       after  query    string false "Cursor: items after this UUID (next page)"
// @Param       before query    string false "Cursor: items before this UUID (prev page)"
// @Success     200    {object} ListResponse
// @Failure     400    {object} httputil.ErrorResponse
// @Router      /users [get]
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	pg, errCode, errMsg := httputil.ParsePaginationParams(r)
	if errCode != "" {
		httputil.WriteValidationError(w, errCode, errMsg)
		return
	}

	params := domain.ListParams{
		Limit:  pg.Limit,
		After:  pg.After,
		Before: pg.Before,
	}

	result, err := h.service.List(r.Context(), params)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	resp := ListResponse{
		Users:   make([]UserResponse, 0, len(result.Users)),
		HasNext: result.HasNext,
		HasPrev: result.HasPrev,
	}
	for i := range result.Users {
		resp.Users = append(resp.Users, toUserResponse(&result.Users[i]))
	}

	httputil.WriteJSON(w, resp, http.StatusOK)
}

// @Summary     Get user by ID
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id  path     string true "User UUID"
// @Success     200 {object} UserResponse
// @Failure     400 {object} httputil.ErrorResponse
// @Failure     404 {object} httputil.ErrorResponse
// @Router      /users/{id} [get]
func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid user id")
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toUserResponse(user), http.StatusOK)
}

// @Summary     Update user
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id      path     string            true "User UUID"
// @Param       request body     UpdateUserRequest  true "Update user"
// @Success     200     {object} UserResponse
// @Failure     400     {object} httputil.ErrorResponse
// @Failure     403     {object} httputil.ErrorResponse
// @Failure     404     {object} httputil.ErrorResponse
// @Failure     409     {object} httputil.ErrorResponse
// @Router      /users/{id} [put]
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	if req.Email == "" {
		httputil.WriteValidationError(w, "validation.email_required", "email is required")
		return
	}
	if req.Role == "" {
		httputil.WriteValidationError(w, "validation.role_required", "role is required")
		return
	}

	user, err := h.service.Update(r.Context(), id, req.Email, req.FirstName, req.LastName, req.AvatarURL, req.Role)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toUserResponse(user), http.StatusOK)
}

// @Summary     Update own profile
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body     UpdateProfileRequest true "Update profile"
// @Success     200     {object} UserResponse
// @Failure     400     {object} httputil.ErrorResponse
// @Failure     401     {object} httputil.ErrorResponse
// @Failure     404     {object} httputil.ErrorResponse
// @Router      /users/me [put]
func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), claims.UserID, app.UpdateProfileParams{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Position:   req.Position,
		Department: req.Department,
		Bio:        req.Bio,
		Timezone:   req.Timezone,
	})
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toUserResponse(user), http.StatusOK)
}

// @Summary     Delete user
// @Tags        users
// @Security    BearerAuth
// @Param       id  path     string true "User UUID"
// @Success     204 "No Content"
// @Failure     400 {object} httputil.ErrorResponse
// @Failure     403 {object} httputil.ErrorResponse
// @Failure     404 {object} httputil.ErrorResponse
// @Router      /users/{id} [delete]
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteValidationError(w, "validation.invalid_id", "invalid user id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
