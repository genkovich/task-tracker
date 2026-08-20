package database

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	// Blank import registers the pgx/v5 database driver with golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending database migrations using the provided
// embedded filesystem and DSN. The fsys should contain .sql migration files
// at its root (e.g. embedded via go:embed).
func RunMigrations(fsys fs.FS, dsn string) error {
	source, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, pgx5DSN(dsn))
	if err != nil {
		return fmt.Errorf("migrations init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations up: %w", err)
	}

	return nil
}

// RunMigrationsDown rolls back the given number of migration steps.
func RunMigrationsDown(fsys fs.FS, dsn string, steps int) error {
	source, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, pgx5DSN(dsn))
	if err != nil {
		return fmt.Errorf("migrations init: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil {
		return fmt.Errorf("migrations down: %w", err)
	}

	return nil
}

// pgx5DSN converts a standard postgres:// DSN to the pgx5:// scheme
// required by the golang-migrate pgx/v5 driver.
func pgx5DSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") {
		return "pgx5://" + strings.TrimPrefix(dsn, "postgres://")
	}
	if strings.HasPrefix(dsn, "postgresql://") {
		return "pgx5://" + strings.TrimPrefix(dsn, "postgresql://")
	}
	return dsn
}
