package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
)

type Service struct {
	repo domain.UserRepository
}

func NewService(repo domain.UserRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, email string, firstName, lastName *string, role string) (*domain.User, error) {
	if role == "" {
		role = domain.RoleMember
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:        id,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) List(ctx context.Context, params domain.ListParams) (*domain.ListResult, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, email string, firstName, lastName, avatarURL *string, role string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName
	user.AvatarURL = avatarURL
	user.Role = role

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

type UpdateProfileParams struct {
	FirstName  *string
	LastName   *string
	Position   *string
	Department *string
	Bio        *string
	Timezone   *string
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, params UpdateProfileParams) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.FirstName = params.FirstName
	user.LastName = params.LastName
	user.Position = params.Position
	user.Department = params.Department
	user.Bio = params.Bio
	user.Timezone = params.Timezone

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) UpdateAvatarURL(ctx context.Context, id uuid.UUID, avatarURL *string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.AvatarURL = avatarURL
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
