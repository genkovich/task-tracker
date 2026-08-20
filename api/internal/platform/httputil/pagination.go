package httputil

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// PaginationParams holds cursor-based pagination parameters parsed from query strings.
type PaginationParams struct {
	Limit  int
	After  *uuid.UUID
	Before *uuid.UUID
}

// ParsePaginationParams extracts limit, after, and before from the request query string.
// Returns an error string (validation code + message) if both cursors are provided or a cursor is invalid.
func ParsePaginationParams(r *http.Request) (*PaginationParams, string, string) {
	p := &PaginationParams{Limit: DefaultLimit}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= MaxLimit {
			p.Limit = n
		}
	}

	if v := r.URL.Query().Get("after"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, "validation.invalid_cursor", "invalid after cursor"
		}
		p.After = &id
	}

	if v := r.URL.Query().Get("before"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, "validation.invalid_cursor", "invalid before cursor"
		}
		p.Before = &id
	}

	if p.After != nil && p.Before != nil {
		return nil, "validation.both_cursors", "cannot use both after and before cursors"
	}

	return p, "", ""
}
