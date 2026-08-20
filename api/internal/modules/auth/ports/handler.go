package ports

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/app"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/infra"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

const (
	oauthStateCookieName = "oauth_state"
	oauthStateTTL        = 5 * time.Minute
	oauthStateBytes      = 32
)

type Handler struct {
	service      *app.Service
	secureCookie bool
	codeStore    *infra.AuthCodeStore
}

func NewHandler(service *app.Service, secureCookie bool, codeStore *infra.AuthCodeStore) *Handler {
	return &Handler{service: service, secureCookie: secureCookie, codeStore: codeStore}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Get("/google", h.handleGoogleAuth)
		r.Get("/google/callback", h.handleGoogleCallback)
		r.Post("/refresh", h.handleRefresh)
		r.Post("/exchange", h.handleExchangeCode)
	})
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/auth/me", h.handleMe)
}

// @Summary     Start Google OAuth flow
// @Tags        auth
// @Success     302 "Redirect to Google OAuth"
// @Router      /auth/google [get]
func (h *Handler) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, oauthStateBytes)
	if _, err := rand.Read(b); err != nil {
		httputil.WriteError(w, fmt.Errorf("generate state: %w", err))
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	googleURL := h.service.GetGoogleAuthURL(state)
	http.Redirect(w, r, googleURL, http.StatusFound)
}

// @Summary     Google OAuth callback
// @Tags        auth
// @Param       code  query string true "Authorization code"
// @Param       state query string true "State parameter"
// @Success     302   "Redirect to frontend with tokens"
// @Failure     400   {object} httputil.ErrorResponse
// @Failure     502   {object} httputil.ErrorResponse
// @Router      /auth/google/callback [get]
func (h *Handler) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate OAuth state (CSRF protection).
	queryState := r.URL.Query().Get("state")
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || queryState == "" ||
		subtle.ConstantTimeCompare([]byte(queryState), []byte(cookie.Value)) != 1 {
		// Clear cookie on failure.
		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookieName,
			Value:    "",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.secureCookie,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
		httputil.WriteError(w, mapError(domain.ErrInvalidState))
		return
	}

	// Clear state cookie — single use.
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		httputil.WriteValidationError(w, "validation.code_required", "authorization code is required")
		return
	}

	tokens, err := h.service.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		slog.Error("handleGoogleCallback failed", "error", err)
		httputil.WriteError(w, mapError(err))
		return
	}

	authCode, err := h.codeStore.Store(tokens)
	if err != nil {
		slog.Error("failed to store auth code", "error", err)
		httputil.WriteError(w, fmt.Errorf("failed to generate auth code: %w", err))
		return
	}

	redirectURL := fmt.Sprintf("%s/auth/callback?code=%s",
		h.service.FrontendURL(),
		authCode,
	)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// @Summary     Refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     RefreshTokenRequest true "Refresh token"
// @Success     200     {object} TokenResponse
// @Failure     400     {object} httputil.ErrorResponse
// @Failure     401     {object} httputil.ErrorResponse
// @Router      /auth/refresh [post]
func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		httputil.WriteValidationError(w, "validation.refresh_token_required", "refresh_token is required")
		return
	}

	tokens, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, http.StatusOK)
}

// @Summary     Get current user
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} CurrentUserResponse
// @Failure     401 {object} httputil.ErrorResponse
// @Router      /auth/me [get]
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, CurrentUserResponse{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		AvatarURL:  user.AvatarURL,
		Role:       user.Role,
		Position:   user.Position,
		Department: user.Department,
		Bio:        user.Bio,
		Timezone:   user.Timezone,
	}, http.StatusOK)
}

// @Summary     Exchange one-time code for tokens
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     ExchangeCodeRequest true "One-time auth code"
// @Success     200     {object} TokenResponse
// @Failure     400     {object} httputil.ErrorResponse
// @Failure     401     {object} httputil.ErrorResponse
// @Router      /auth/exchange [post]
func (h *Handler) handleExchangeCode(w http.ResponseWriter, r *http.Request) {
	var req ExchangeCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteValidationError(w, "validation.invalid_body", "invalid request body")
		return
	}

	if req.Code == "" {
		httputil.WriteValidationError(w, "validation.code_required", "code is required")
		return
	}

	tokens, err := h.codeStore.Exchange(req.Code)
	if err != nil {
		httputil.WriteJSON(w, httputil.ErrorResponse{
			Error: httputil.ErrorBody{
				Code:    "auth.invalid_exchange_code",
				Message: "invalid or expired exchange code",
			},
		}, http.StatusUnauthorized)
		return
	}

	httputil.WriteJSON(w, TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, http.StatusOK)
}
