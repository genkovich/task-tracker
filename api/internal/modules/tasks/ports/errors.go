package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

var errorMap = []struct {
	target error
	appErr apperr.Error
}{
	{domain.ErrCardNotFound, apperr.Error{Code: "tasks.card_not_found", Message: "card not found", StatusCode: http.StatusNotFound}},
	{domain.ErrNameRequired, apperr.Error{Code: "tasks.name_required", Message: "card name is required", StatusCode: http.StatusBadRequest}},
	{domain.ErrCardFieldTooLong, apperr.Error{Code: "tasks.field_too_long", Message: "a card field exceeds its length limit", StatusCode: http.StatusBadRequest}},
	{domain.ErrInvalidColumn, apperr.Error{Code: "tasks.invalid_column", Message: "column_status must be one of todo, in_progress, done", StatusCode: http.StatusBadRequest}},
	{domain.ErrLinkNotFound, apperr.Error{Code: "tasks.link_not_found", Message: "public link not found", StatusCode: http.StatusNotFound}},
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
