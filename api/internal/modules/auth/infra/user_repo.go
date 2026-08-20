package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
)

type PostgresUserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, google_access_token, google_refresh_token, google_token_expiry
		 FROM users WHERE google_id = $1`, googleID,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
		&u.Position, &u.Department, &u.Bio, &u.Timezone,
		&u.GoogleAccessToken, &u.GoogleRefreshToken, &u.GoogleTokenExpiry)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, google_access_token, google_refresh_token, google_token_expiry
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
		&u.Position, &u.Department, &u.Bio, &u.Timezone,
		&u.GoogleAccessToken, &u.GoogleRefreshToken, &u.GoogleTokenExpiry)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		user.ID, user.Email, user.FirstName, user.LastName, user.AvatarURL, user.GoogleID, user.Role,
		user.Position, user.Department, user.Bio, user.Timezone,
	)
	return err
}

func (r *PostgresUserRepository) UpdateGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET google_id = $1, updated_at = now() WHERE id = $2`,
		googleID, userID,
	)
	return err
}

func (r *PostgresUserRepository) SyncProfile(ctx context.Context, userID uuid.UUID, avatarURL, firstName, familyName string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users
		 SET avatar_url = COALESCE(NULLIF($2, ''), avatar_url),
		     first_name = COALESCE(NULLIF($3, ''), first_name),
		     last_name  = COALESCE(NULLIF($4, ''), last_name),
		     updated_at = now()
		 WHERE id = $1`,
		userID, avatarURL, firstName, familyName,
	)
	return err
}

func (r *PostgresUserRepository) UpdateGoogleTokens(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiry time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET google_access_token = $1, google_refresh_token = $2, google_token_expiry = $3, updated_at = now() WHERE id = $4`,
		accessToken, refreshToken, expiry, userID,
	)
	return err
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, avatar_url, google_id, role, position, department, bio, timezone, google_access_token, google_refresh_token, google_token_expiry
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.GoogleID, &u.Role,
		&u.Position, &u.Department, &u.Bio, &u.Timezone,
		&u.GoogleAccessToken, &u.GoogleRefreshToken, &u.GoogleTokenExpiry)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
