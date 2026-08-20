//go:build integration

package database_test

import (
	"context"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
)

func TestDatabaseIntegration_Connectivity(t *testing.T) {
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := c.DB.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestDatabaseIntegration_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)
	db := c.DB

	// CREATE TABLE
	_, err := db.Exec(ctx, `
		CREATE TABLE test_items (
			id    SERIAL PRIMARY KEY,
			name  TEXT NOT NULL,
			value INT  NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// INSERT
	var insertedID int
	err = db.QueryRow(ctx,
		`INSERT INTO test_items (name, value) VALUES ($1, $2) RETURNING id`,
		"item1", 100,
	).Scan(&insertedID)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if insertedID == 0 {
		t.Fatal("expected non-zero inserted id")
	}

	// SELECT
	var name string
	var value int
	err = db.QueryRow(ctx,
		`SELECT name, value FROM test_items WHERE id = $1`, insertedID,
	).Scan(&name, &value)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "item1" || value != 100 {
		t.Fatalf("select: got (%q, %d), want (\"item1\", 100)", name, value)
	}

	// UPDATE
	_, err = db.Exec(ctx,
		`UPDATE test_items SET value = $1 WHERE id = $2`, 200, insertedID,
	)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify UPDATE
	err = db.QueryRow(ctx,
		`SELECT value FROM test_items WHERE id = $1`, insertedID,
	).Scan(&value)
	if err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if value != 200 {
		t.Fatalf("select after update: got %d, want 200", value)
	}

	// DELETE
	tag, err := db.Exec(ctx,
		`DELETE FROM test_items WHERE id = $1`, insertedID,
	)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("delete: affected %d rows, want 1", tag.RowsAffected())
	}
}
