package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

const tokenByteLength = 16 // 128 bits, per ADR-0003 — never uuid.NewV7 (encodes a timestamp)

type PublicLinkService struct {
	repo        domain.PublicLinkRepository
	broadcaster *Broadcaster
}

func NewPublicLinkService(repo domain.PublicLinkRepository, broadcaster *Broadcaster) *PublicLinkService {
	return &PublicLinkService{repo: repo, broadcaster: broadcaster}
}

// GenerateLink creates a new unpredictable token, replacing any currently
// active link (repo-layer invariant), and broadcasts the change.
func (s *PublicLinkService) GenerateLink(ctx context.Context) (*domain.PublicLink, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	link, err := s.repo.Generate(ctx, token)
	if err != nil {
		return nil, err
	}

	s.broadcaster.Publish(BoardEvent{Type: EventLinkGenerated, Payload: link.ID})
	return link, nil
}

// DisableLink is idempotent — disabling an already-disabled link succeeds
// without error.
func (s *PublicLinkService) DisableLink(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Disable(ctx, id); err != nil {
		return err
	}
	s.broadcaster.Publish(BoardEvent{Type: EventLinkDisabled, Payload: id})
	return nil
}

func (s *PublicLinkService) GetActiveLink(ctx context.Context) (*domain.PublicLink, error) {
	return s.repo.GetActive(ctx)
}

func (s *PublicLinkService) ResolvePublicLink(ctx context.Context, token string) (*domain.PublicLink, error) {
	return s.repo.ResolveByToken(ctx, token)
}

func generateToken() (string, error) {
	b := make([]byte, tokenByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
