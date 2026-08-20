package dbtest

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container holds a running PostgreSQL test container and a connected database.
type Container struct {
	container *postgres.PostgresContainer
	DB        *database.DB
	DSN       string
}

// StartPostgres launches a PostgreSQL container and returns a ready-to-use
// database connection. The container is automatically terminated via t.Cleanup.
func StartPostgres(ctx context.Context, t *testing.T) *Container {
	t.Helper()

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping: docker is not available")
	}

	pgContainer, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("dbtest: start postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			t.Logf("dbtest: terminate container after dsn error: %v", termErr)
		}
		t.Fatalf("dbtest: get connection string: %v", err)
	}

	db, err := database.New(ctx, dsn)
	if err != nil {
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			t.Logf("dbtest: terminate container after connect error: %v", termErr)
		}
		t.Fatalf("dbtest: connect to database: %v", err)
	}

	c := &Container{
		container: pgContainer,
		DB:        db,
		DSN:       dsn,
	}

	t.Cleanup(func() {
		c.Close(context.Background())
	})

	return c
}

// Close shuts down the database connection and terminates the container.
func (c *Container) Close(ctx context.Context) {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.container != nil {
		_ = c.container.Terminate(ctx)
	}
}
