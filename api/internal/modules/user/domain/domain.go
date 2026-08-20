package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

var (
	ErrNotFound           = errors.New("user.not_found")
	ErrEmailAlreadyExists = errors.New("user.email_already_exists")
	ErrForbidden          = errors.New("user.forbidden")
)

type User struct {
	ID         uuid.UUID
	Email      string
	FirstName  *string
	LastName   *string
	AvatarURL  *string
	GoogleID   *string
	Role       string
	Position   *string
	Department *string
	Bio        *string
	Timezone   *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ListParams struct {
	Limit  int
	After  *uuid.UUID // cursor: items older than this ID (next page)
	Before *uuid.UUID // cursor: items newer than this ID (prev page)
}

type ListResult struct {
	Users   []User
	HasNext bool
	HasPrev bool
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
