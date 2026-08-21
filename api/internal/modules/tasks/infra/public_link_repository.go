package infra

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type PostgresPublicLinkRepository struct {
	db *database.DB
}

func NewPostgresPublicLinkRepository(db *database.DB) *PostgresPublicLinkRepository {
	return &PostgresPublicLinkRepository{db: db}
}

// Generate disables any currently active link, then inserts a new one — a
// single transaction, so there is never more than one active link
// (data-model.md's service-layer invariant).
func (r *PostgresPublicLinkRepository) Generate(ctx context.Context, token string) (*domain.PublicLink, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`UPDATE public_links SET disabled_at = now() WHERE disabled_at IS NULL`,
	); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	link := &domain.PublicLink{ID: id, Token: token}
	if err := tx.QueryRow(ctx,
		`INSERT INTO public_links (id, token) VALUES ($1, $2) RETURNING created_at`,
		link.ID, link.Token,
	).Scan(&link.CreatedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return link, nil
}

func (r *PostgresPublicLinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PublicLink, error) {
	var l domain.PublicLink
	err := r.db.QueryRow(ctx,
		`SELECT id, token, disabled_at, created_at FROM public_links WHERE id = $1`, id,
	).Scan(&l.ID, &l.Token, &l.DisabledAt, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLinkNotFound
		}
		return nil, err
	}
	return &l, nil
}

// GetActive returns the current active link, or nil if none is active — a
// normal board state, never an error.
func (r *PostgresPublicLinkRepository) GetActive(ctx context.Context) (*domain.PublicLink, error) {
	var l domain.PublicLink
	err := r.db.QueryRow(ctx,
		`SELECT id, token, disabled_at, created_at FROM public_links WHERE disabled_at IS NULL`,
	).Scan(&l.ID, &l.Token, &l.DisabledAt, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// ResolveByToken finds an active link by token. A disabled or never-valid
// token is reported identically (ErrLinkNotFound) — the caller must not be
// able to tell the two apart (AC-05).
func (r *PostgresPublicLinkRepository) ResolveByToken(ctx context.Context, token string) (*domain.PublicLink, error) {
	var l domain.PublicLink
	err := r.db.QueryRow(ctx,
		`SELECT id, token, disabled_at, created_at FROM public_links WHERE token = $1 AND disabled_at IS NULL`, token,
	).Scan(&l.ID, &l.Token, &l.DisabledAt, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLinkNotFound
		}
		return nil, err
	}
	return &l, nil
}

// Disable is idempotent: disabling an already-disabled link is a no-op
// success, not an error.
func (r *PostgresPublicLinkRepository) Disable(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`UPDATE public_links SET disabled_at = now() WHERE id = $1 AND disabled_at IS NULL`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		// Either the id doesn't exist, or it's already disabled — distinguish
		// via a lookup so the caller gets ErrLinkNotFound only for a real miss.
		if _, err := r.GetByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
