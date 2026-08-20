package auth

import (
	"github.com/genkovich/task-tracker/api/internal/modules/auth/app"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type Config struct {
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	AppEnv             string
}

func New(db *database.DB, cfg Config) *ports.Handler {
	userRepo := infra.NewUserRepository(db)
	tokenService := infra.NewJWTService(cfg.JWTSecret)
	googleProvider := infra.NewGoogleOAuthProvider(infra.GoogleOAuthConfig{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
	})
	svc := app.NewService(userRepo, tokenService, googleProvider, cfg.FrontendURL)
	secureCookie := cfg.AppEnv != "development"
	codeStore := infra.NewAuthCodeStore()
	return ports.NewHandler(svc, secureCookie, codeStore)
}

// tokenValidatorAdapter wraps infra.JWTService to implement authmw.TokenValidator.
type tokenValidatorAdapter struct {
	jwt *infra.JWTService
}

func NewTokenValidator(jwtSecret string) authmw.TokenValidator {
	return &tokenValidatorAdapter{jwt: infra.NewJWTService(jwtSecret)}
}

func (a *tokenValidatorAdapter) Validate(token string) (*authmw.Claims, error) {
	dc, err := a.jwt.Validate(token)
	if err != nil {
		return nil, err
	}
	if dc.TokenType != domain.TokenTypeAccess {
		return nil, domain.ErrInvalidToken
	}
	return &authmw.Claims{
		UserID: dc.UserID,
		Email:  dc.Email,
		Role:   dc.Role,
	}, nil
}
