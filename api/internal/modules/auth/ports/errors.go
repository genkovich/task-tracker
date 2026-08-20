package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

var errorMap = []struct {
	target error
	appErr apperr.Error
}{
	{domain.ErrInvalidToken, apperr.Error{Code: "auth.invalid_token", Message: "invalid token", StatusCode: http.StatusUnauthorized}},
	{domain.ErrExpiredToken, apperr.Error{Code: "auth.expired_token", Message: "token has expired", StatusCode: http.StatusUnauthorized}},
	{domain.ErrInvalidCode, apperr.Error{Code: "auth.invalid_code", Message: "invalid authorization code", StatusCode: http.StatusBadRequest}},
	{domain.ErrGoogleAPIFailed, apperr.Error{Code: "auth.google_api_failed", Message: "google authentication failed", StatusCode: http.StatusBadGateway}},
	{domain.ErrEmailNotVerified, apperr.Error{Code: "auth.email_not_verified", Message: "email not verified with Google", StatusCode: http.StatusBadRequest}},
	{domain.ErrInvalidState, apperr.Error{Code: "auth.invalid_state", Message: "invalid or missing OAuth state", StatusCode: http.StatusBadRequest}},
	{domain.ErrUserNotFound, apperr.Error{Code: "auth.user_not_found", Message: "user not found", StatusCode: http.StatusNotFound}},
	{domain.ErrNotRefreshToken, apperr.Error{Code: "auth.not_refresh_token", Message: "provided token is not a refresh token", StatusCode: http.StatusUnauthorized}},
}

func mapError(err error) error {
	for _, m := range errorMap {
		if errors.Is(err, m.target) {
			mapped := m.appErr
			return &mapped
		}
	}
	return err
}
