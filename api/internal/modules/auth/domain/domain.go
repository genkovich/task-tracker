package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken        = errors.New("auth.invalid_token")
	ErrExpiredToken        = errors.New("auth.expired_token")
	ErrInvalidCode         = errors.New("auth.invalid_code")
	ErrGoogleAPIFailed     = errors.New("auth.google_api_failed")
	ErrEmailNotVerified    = errors.New("auth.email_not_verified")
	ErrInvalidState        = errors.New("auth.invalid_state")
	ErrUserNotFound        = errors.New("auth.user_not_found")
	ErrNotRefreshToken     = errors.New("auth.not_refresh_token")
	ErrInvalidExchangeCode = errors.New("auth.invalid_exchange_code")
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

type Claims struct {
	UserID    uuid.UUID
	Email     string
	Role      string
	TokenType string
}

type GoogleUserInfo struct {
	Sub           string
	Email         string
	Name          string
	GivenName     string
	FamilyName    string
	Picture       string
	EmailVerified bool
}

type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type TokenService interface {
	Generate(userID uuid.UUID, email, role string) (*TokenPair, error)
	Validate(token string) (*Claims, error)
	RefreshTokens(refreshToken string) (*TokenPair, error)
}

type GoogleProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)
	GetUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error)
}

type UserRepository interface {
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	UpdateGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	SyncProfile(ctx context.Context, userID uuid.UUID, avatarURL, firstName, familyName string) error
	UpdateGoogleTokens(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiry time.Time) error
}

type User struct {
	ID                 uuid.UUID
	Email              string
	FirstName          *string
	LastName           *string
	AvatarURL          *string
	GoogleID           *string
	Role               string
	Position           *string
	Department         *string
	Bio                *string
	Timezone           *string
	GoogleAccessToken  *string
	GoogleRefreshToken *string
	GoogleTokenExpiry  *time.Time
}
