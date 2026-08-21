package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ColumnTodo       = "todo"
	ColumnInProgress = "in_progress"
	ColumnDone       = "done"

	maxNameLength     = 200
	maxAssigneeLength = 100
)

var (
	ErrCardNotFound     = errors.New("card not found")
	ErrNameRequired     = errors.New("card name is required")
	ErrCardFieldTooLong = errors.New("card field exceeds its length limit")
	ErrLinkNotFound     = errors.New("public link not found")
	ErrLinkDisabled     = errors.New("public link is disabled")
)

// Card is a unit of work on the board — one name, an optional free-text
// assignee, and a fixed column. There is exactly one board system-wide, so
// Card carries no board/tenant reference.
type Card struct {
	ID           uuid.UUID
	Name         string
	Assignee     *string
	ColumnStatus string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate enforces the invariants spec §6 NFR requires before a write:
// a non-blank name within 200 chars, and an assignee (if present) within
// 100 chars.
func (c Card) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrNameRequired
	}
	if len(c.Name) > maxNameLength {
		return ErrCardFieldTooLong
	}
	if c.Assignee != nil && len(*c.Assignee) > maxAssigneeLength {
		return ErrCardFieldTooLong
	}
	return nil
}

// IsValidColumnStatus reports whether status is one of the three fixed
// columns (spec §3 Non-goals: no column CRUD).
func IsValidColumnStatus(status string) bool {
	switch status {
	case ColumnTodo, ColumnInProgress, ColumnDone:
		return true
	default:
		return false
	}
}

// PublicLink is one row per generated link (ADR-0003): an opaque, unguessable
// Token, and a Disabled marker that — once set — is never cleared.
type PublicLink struct {
	ID         uuid.UUID
	Token      string
	DisabledAt *time.Time
	CreatedAt  time.Time
}

// Active reports whether this link is still valid for public access.
func (l PublicLink) Active() bool {
	return l.DisabledAt == nil
}
