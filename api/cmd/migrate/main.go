package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/migrations"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	steps := flag.Int("steps", 1, "number of migrations to roll back (only for down)")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	switch *direction {
	case "up":
		fmt.Println("applying migrations...")
		if err := database.RunMigrations(migrations.FS, dsn); err != nil {
			log.Fatalf("migrations up failed: %v", err)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		fmt.Printf("rolling back %d migration(s)...\n", *steps)
		if err := database.RunMigrationsDown(migrations.FS, dsn, *steps); err != nil {
			log.Fatalf("migrations down failed: %v", err)
		}
		fmt.Println("rollback completed successfully")

	default:
		log.Fatalf("unknown direction %q: use 'up' or 'down'", *direction)
	}
}
