//go:build integration

package database_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

func TestMigrationsIntegration_UsersSchema(t *testing.T) {
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	// Apply all migrations.
	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// users table must carry the columns added across migrations 000002-000005.
	columns := []string{"id", "email", "role", "google_id", "position", "department", "bio", "timezone", "google_access_token"}
	for _, column := range columns {
		var exists bool
		err := c.DB.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'users' AND column_name = $1
			)`, column,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check column users.%s exists: %v", column, err)
		}
		if !exists {
			t.Errorf("expected column users.%s to exist after migrations up, but it does not", column)
		}
	}

	// The seed migration (000003) must have created the admin account.
	var admins int
	if err := c.DB.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE role = 'admin'`,
	).Scan(&admins); err != nil {
		t.Fatalf("count admin users: %v", err)
	}
	if admins != 1 {
		t.Errorf("expected exactly 1 seeded admin user, got %d", admins)
	}

	// Roll back everything except the init migration — the users table must
	// be gone. Count the steps from the embedded FS so new feature migrations
	// (board added 000006-000011) keep this test honest instead of stale.
	ups, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	steps := len(ups) - 1
	if err := database.RunMigrationsDown(migrations.FS, c.DSN, steps); err != nil {
		t.Fatalf("RunMigrationsDown(%d): %v", steps, err)
	}

	var exists bool
	err = c.DB.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'users'
		)`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check users table gone: %v", err)
	}
	if exists {
		t.Error("expected users table to be gone after migrations down, but it still exists")
	}
}
