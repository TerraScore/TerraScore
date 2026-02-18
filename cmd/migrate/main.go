package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		direction = flag.String("direction", "up", "Migration direction: up or down")
		dbURL     = flag.String("db", "", "Database URL")
		steps     = flag.Int("steps", 0, "Number of steps (0 = all)")
		path      = flag.String("path", "db/migrations", "Path to migration files")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DB_URL")
	}
	if *dbURL == "" {
		log.Fatal("database URL required: use -db flag or DB_URL env var")
	}

	m, err := migrate.New("file://"+*path, *dbURL)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	default:
		log.Fatalf("unknown direction: %s (use 'up' or 'down')", *direction)
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}

	version, dirty, _ := m.Version()
	fmt.Printf("migration %s complete — version: %d, dirty: %v\n", *direction, version, dirty)
}
