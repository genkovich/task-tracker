package infra

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type PostgresCardRepository struct {
	db *database.DB
}

func NewPostgresCardRepository(db *database.DB) *PostgresCardRepository {
	return &PostgresCardRepository{db: db}
}

func (r *PostgresCardRepository) Create(ctx context.Context, card *domain.Card) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO cards (id, name, assignee, column_status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		card.ID, card.Name, card.Assignee, card.ColumnStatus,
	).Scan(&card.CreatedAt, &card.UpdatedAt)
}

func (r *PostgresCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	var c domain.Card
	err := r.db.QueryRow(ctx,
		`SELECT id, name, assignee, column_status, created_at, updated_at
		 FROM cards WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Assignee, &c.ColumnStatus, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCardNotFound
		}
		return nil, err
	}
	return &c, nil
}

// List returns every card, ordered by column then by creation time within a
// column (no manual reordering — clarify decision).
func (r *PostgresCardRepository) List(ctx context.Context) ([]domain.Card, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, assignee, column_status, created_at, updated_at
		 FROM cards ORDER BY column_status, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []domain.Card{}
	for rows.Next() {
		var c domain.Card
		if err := rows.Scan(&c.ID, &c.Name, &c.Assignee, &c.ColumnStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

// Update unconditionally overwrites name/assignee and bumps updated_at
// (ADR-0002) — no version check, no rejection. Returns the full row (not
// just updated_at) so the caller always has a complete, contract-valid Card.
func (r *PostgresCardRepository) Update(ctx context.Context, card *domain.Card) error {
	err := r.db.QueryRow(ctx,
		`UPDATE cards SET name = $1, assignee = $2, updated_at = now()
		 WHERE id = $3
		 RETURNING column_status, created_at, updated_at`,
		card.Name, card.Assignee, card.ID,
	).Scan(&card.ColumnStatus, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCardNotFound
		}
		return err
	}
	return nil
}

// Move unconditionally overwrites the column and bumps updated_at
// (ADR-0002) — the last write the server processes wins.
func (r *PostgresCardRepository) Move(ctx context.Context, id uuid.UUID, columnStatus string) (*domain.Card, error) {
	var c domain.Card
	c.ID = id
	err := r.db.QueryRow(ctx,
		`UPDATE cards SET column_status = $1, updated_at = now()
		 WHERE id = $2
		 RETURNING name, assignee, column_status, created_at, updated_at`,
		columnStatus, id,
	).Scan(&c.Name, &c.Assignee, &c.ColumnStatus, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCardNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *PostgresCardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM cards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCardNotFound
	}
	return nil
}
