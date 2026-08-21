package ports

import (
	"errors"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

// errCommonNotFound is the shared, module-neutral not-found used by the
// public-board endpoints — deliberately NOT a tasks.* code, so a disabled or
// never-valid link can never be distinguished from any other bad address
// (AC-05, ADR-0003).
var errCommonNotFound = apperr.Error{Code: "common.not_found", Message: "not found", StatusCode: http.StatusNotFound}

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

// mapPublicError maps ErrLinkNotFound to the shared common.not_found code
// instead of tasks.link_not_found — used only by the public-board endpoints.
func mapPublicError(err error) error {
	if errors.Is(err, domain.ErrLinkNotFound) {
		mapped := errCommonNotFound
		return &mapped
	}
	return mapError(err)
}
