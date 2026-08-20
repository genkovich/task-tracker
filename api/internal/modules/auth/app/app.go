package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
)

const roleMember = "member"

type Service struct {
	userRepo       domain.UserRepository
	tokenService   domain.TokenService
	googleProvider domain.GoogleProvider
	frontendURL    string
}

func NewService(
	userRepo domain.UserRepository,
	tokenService domain.TokenService,
	googleProvider domain.GoogleProvider,
	frontendURL string,
) *Service {
	return &Service{
		userRepo:       userRepo,
		tokenService:   tokenService,
		googleProvider: googleProvider,
		frontendURL:    frontendURL,
	}
}

func (s *Service) GetGoogleAuthURL(state string) string {
	return s.googleProvider.GetAuthURL(state)
}

func (s *Service) FrontendURL() string {
	return s.frontendURL
}

func (s *Service) HandleGoogleCallback(ctx context.Context, code string) (*domain.TokenPair, error) {
	oauthTokens, err := s.googleProvider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	googleUser, err := s.googleProvider.GetUserInfo(ctx, oauthTokens.AccessToken)
	if err != nil {
		return nil, err
	}

	if !googleUser.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	user, err := s.findOrCreateUser(ctx, googleUser)
	if err != nil {
		return nil, err
	}

	if oauthTokens.RefreshToken != "" {
		expiry := time.Now().Add(time.Duration(oauthTokens.ExpiresIn) * time.Second)
		if err := s.userRepo.UpdateGoogleTokens(ctx, user.ID, oauthTokens.AccessToken, oauthTokens.RefreshToken, expiry); err != nil {
			slog.Error("failed to save google tokens", "user_id", user.ID, "error", err)
		}
	}

	return s.tokenService.Generate(user.ID, user.Email, user.Role)
}

func (s *Service) GetCurrentUser(token string) (*domain.Claims, error) {
	return s.tokenService.Validate(token)
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *Service) RefreshToken(refreshToken string) (*domain.TokenPair, error) {
	return s.tokenService.RefreshTokens(refreshToken)
}

func (s *Service) findOrCreateUser(ctx context.Context, googleUser *domain.GoogleUserInfo) (*domain.User, error) {
	// Try to find by Google ID.
	user, err := s.userRepo.GetByGoogleID(ctx, googleUser.Sub)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if err := s.userRepo.SyncProfile(ctx, user.ID, googleUser.Picture, googleUser.GivenName, googleUser.FamilyName); err != nil {
			slog.Error("failed to sync profile", "user_id", user.ID, "error", err)
		}
		return user, nil
	}

	// Try to find by email — link Google ID to existing user.
	user, err = s.userRepo.GetByEmail(ctx, googleUser.Email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if err := s.userRepo.UpdateGoogleID(ctx, user.ID, googleUser.Sub); err != nil {
			return nil, err
		}
		if err := s.userRepo.SyncProfile(ctx, user.ID, googleUser.Picture, googleUser.GivenName, googleUser.FamilyName); err != nil {
			slog.Error("failed to sync profile", "user_id", user.ID, "error", err)
		}
		user.GoogleID = &googleUser.Sub
		return user, nil
	}

	// Create new user.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	newUser := &domain.User{
		ID:       id,
		Email:    googleUser.Email,
		GoogleID: &googleUser.Sub,
		Role:     roleMember,
	}

	if googleUser.GivenName != "" {
		newUser.FirstName = &googleUser.GivenName
	}
	if googleUser.FamilyName != "" {
		newUser.LastName = &googleUser.FamilyName
	}
	if googleUser.Picture != "" {
		newUser.AvatarURL = &googleUser.Picture
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}
