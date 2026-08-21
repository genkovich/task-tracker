package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// tokenBytes is the size of the random token generated for a public link —
// 128 bits (ADR-0003: "128-bit random / UUIDv4"), hex-encoded on the wire.
const tokenBytes = 16

// LinkService implements the per-board public-link use-cases (issue/revoke,
// boards BRD-06).
type LinkService struct {
	repo  ports.Repository
	bcast ports.Broadcaster
}

// NewLinkService wires a LinkService against the given repository and
// broadcaster.
func NewLinkService(repo ports.Repository, bcast ports.Broadcaster) *LinkService {
	return &LinkService{repo: repo, bcast: bcast}
}

// IssuePublicLink issues a new opaque public-link token for boardID (AC-07).
// Returns domain.ErrLinkAlreadyActive if the board already has an active
// link (maps to the 409 board.link_already_active, contracts/openapi.yaml)
// and domain.ErrBoardNotFound for an unknown board; no write happens on
// either path.
func (s *LinkService) IssuePublicLink(ctx context.Context, boardID uuid.UUID) (*domain.PublicLink, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate public link token: %w", err)
	}

	link := &domain.PublicLink{
		ID:      uuid.Must(uuid.NewV7()),
		BoardID: boardID,
		Token:   token,
	}

	if err := s.repo.IssuePublicLink(ctx, link); err != nil {
		return nil, fmt.Errorf("issue public link: %w", err)
	}

	return link, nil
}

// RevokePublicLink deletes the board's active public link (AC-08) and closes
// any already-open SSE connections for its token (AC-11, sad.md §6 Flow 3).
func (s *LinkService) RevokePublicLink(ctx context.Context, boardID uuid.UUID) error {
	active, err := s.repo.PublicLinkByBoard(ctx, boardID)
	if err != nil {
		return fmt.Errorf("revoke public link: find active link: %w", err)
	}

	if err := s.repo.RevokePublicLink(ctx, boardID); err != nil {
		return fmt.Errorf("revoke public link: %w", err)
	}

	s.bcast.CloseToken(boardID, active.Token)
	return nil
}

// generateToken returns an opaque, unpredictable token (ADR-0003) suitable
// for use as a public-link's bearer credential.
func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
