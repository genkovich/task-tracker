package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func WriteJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		WriteJSON(w, ErrorResponse{
			Error: ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		}, appErr.StatusCode)
		return
	}

	// An unmapped error reaches the wire as an opaque 500 — log the detail
	// here or it is lost entirely; the response body stays generic.
	slog.Error("unmapped error written as 500", "error", err)
	WriteJSON(w, ErrorResponse{
		Error: ErrorBody{
			Code:    "internal_error",
			Message: "internal server error",
		},
	}, http.StatusInternalServerError)
}

func WriteValidationError(w http.ResponseWriter, code, message string) {
	WriteJSON(w, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	}, http.StatusBadRequest)
}
