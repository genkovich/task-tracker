package ports

import "github.com/google/uuid"

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ExchangeCodeRequest struct {
	Code string `json:"code"`
}

type CurrentUserResponse struct {
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
}
