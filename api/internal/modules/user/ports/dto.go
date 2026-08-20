package ports

import (
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
)

type CreateUserRequest struct {
	Email     string  `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Role      string  `json:"role"`
}

type UpdateUserRequest struct {
	Email     string  `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
}

type UpdateProfileRequest struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	Position   *string `json:"position"`
	Department *string `json:"department"`
	Bio        *string `json:"bio"`
	Timezone   *string `json:"timezone"`
}

type UserResponse struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FirstName  *string   `json:"first_name"`
	LastName   *string   `json:"last_name"`
	AvatarURL  *string   `json:"avatar_url"`
	Role       string    `json:"role"`
	Position   *string   `json:"position"`
	Department *string   `json:"department"`
	Bio        *string   `json:"bio"`
	Timezone   *string   `json:"timezone"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ListResponse struct {
	Users   []UserResponse `json:"users"`
	HasNext bool           `json:"has_next"`
	HasPrev bool           `json:"has_prev"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		AvatarURL:  u.AvatarURL,
		Role:       u.Role,
		Position:   u.Position,
		Department: u.Department,
		Bio:        u.Bio,
		Timezone:   u.Timezone,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
