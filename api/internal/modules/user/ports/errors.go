package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

var errorMap = []struct {
	target error
	appErr apperr.Error
}{
	{domain.ErrNotFound, apperr.Error{Code: "user.not_found", Message: "user not found", StatusCode: http.StatusNotFound}},
	{domain.ErrEmailAlreadyExists, apperr.Error{Code: "user.email_already_exists", Message: "email already exists", StatusCode: http.StatusConflict}},
	{domain.ErrForbidden, apperr.Error{Code: "user.forbidden", Message: "admin access required", StatusCode: http.StatusForbidden}},
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
