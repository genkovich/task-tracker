package infra

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type PostgresUserRepository struct {
	db *database.DB
}

func NewPostgresUserRepository(db *database.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING created_at, updated_at`,
		user.ID, user.Email, user.FirstName, user.LastName, user.AvatarURL, user.GoogleID, user.Role,
		user.Position, user.Department, user.Bio, user.Timezone,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if database.IsPgUniqueViolation(err) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
		&u.Position, &u.Department, &u.Bio, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
		&u.Position, &u.Department, &u.Bio, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) List(ctx context.Context, params domain.ListParams) (*domain.ListResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	fetchLimit := limit + 1

	switch {
	case params.After != nil:
		rows, err = r.db.Query(ctx,
			`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, created_at, updated_at
			 FROM users WHERE id < $1 ORDER BY id DESC LIMIT $2`,
			*params.After, fetchLimit)
	case params.Before != nil:
		rows, err = r.db.Query(ctx,
			`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, created_at, updated_at
			 FROM users WHERE id > $1 ORDER BY id ASC LIMIT $2`,
			*params.Before, fetchLimit)
	default:
		rows, err = r.db.Query(ctx,
			`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, created_at, updated_at
			 FROM users ORDER BY id DESC LIMIT $1`,
			fetchLimit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
			&u.Position, &u.Department, &u.Bio, &u.Timezone, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &domain.ListResult{}

	switch {
	case params.After != nil:
		result.HasPrev = true
		result.HasNext = len(users) > limit
		if result.HasNext {
			users = users[:limit]
		}
	case params.Before != nil:
		result.HasNext = true
		result.HasPrev = len(users) > limit
		if result.HasPrev {
			users = users[:limit]
		}
		slices.Reverse(users)
	default:
		result.HasPrev = false
		result.HasNext = len(users) > limit
		if result.HasNext {
			users = users[:limit]
		}
	}

	result.Users = users
	return result, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	err := r.db.QueryRow(ctx,
		`UPDATE users SET email = $1, first_name = $2, last_name = $3, avatar_url = $4, google_id = $5, role = $6,
		 position = $7, department = $8, bio = $9, timezone = $10, updated_at = now()
		 WHERE id = $11
		 RETURNING created_at, updated_at`,
		user.Email, user.FirstName, user.LastName, user.AvatarURL, user.GoogleID, user.Role,
		user.Position, user.Department, user.Bio, user.Timezone, user.ID,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if database.IsPgUniqueViolation(err) {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
