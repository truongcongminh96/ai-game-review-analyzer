package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

func main() {
	cfg := config.Load()
	if cfg.SupabaseDBURL == "" {
		log.Fatal("SUPABASE_DB_URL or DATABASE_URL is required to run migrations")
	}

	if len(os.Args) < 2 {
		log.Fatal(usage())
	}

	m, err := newMigrator(cfg.SupabaseDBURL)
	if err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}
	defer closeMigrator(m)

	command := os.Args[1]
	args := os.Args[2:]

	message, err := run(m, command, args)
	if err != nil {
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			log.Fatalf("database is dirty at version %d; fix the migration or run `go run ./cmd/migrate force <version>`", dirty.Version)
		}
		log.Fatal(err)
	}

	if message != "" {
		fmt.Println(message)
	}
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	migrationsDir, err := filepath.Abs(filepath.Join("db", "migrations"))
	if err != nil {
		return nil, fmt.Errorf("resolve migrations dir: %w", err)
	}

	if _, err := os.Stat(migrationsDir); err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(migrationsDir),
	}).String()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database for migrations: %w", err)
	}

	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create postgres migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}

	return m, nil
}

func closeMigrator(m *migrate.Migrate) {
	sourceErr, databaseErr := m.Close()
	if sourceErr != nil {
		log.Printf("warning: failed to close migration source: %v", sourceErr)
	}
	if databaseErr != nil {
		log.Printf("warning: failed to close migration database: %v", databaseErr)
	}
}

func run(m *migrate.Migrate, command string, args []string) (string, error) {
	switch command {
	case "up":
		if len(args) == 0 {
			return runUp(m.Up())
		}

		steps, err := parsePositiveInt(args[0], "up steps")
		if err != nil {
			return "", err
		}

		return runUp(m.Steps(steps))

	case "down":
		if len(args) == 0 {
			return "", ignoreNoChange(m.Down())
		}

		steps, err := parsePositiveInt(args[0], "down steps")
		if err != nil {
			return "", err
		}

		return "", ignoreNoChange(m.Steps(-steps))

	case "goto":
		if len(args) != 1 {
			return "", fmt.Errorf("goto requires exactly 1 version argument\n\n%s", usage())
		}

		version, err := parsePositiveInt(args[0], "goto version")
		if err != nil {
			return "", err
		}

		return "", ignoreNoChange(m.Migrate(uint(version)))

	case "force":
		if len(args) != 1 {
			return "", fmt.Errorf("force requires exactly 1 version argument\n\n%s", usage())
		}

		version, err := parseNonNegativeInt(args[0], "force version")
		if err != nil {
			return "", err
		}

		return "", m.Force(version)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				return "version=none dirty=false", nil
			}
			return "", fmt.Errorf("read migration version: %w", err)
		}

		return fmt.Sprintf("version=%d dirty=%t", version, dirty), nil

	default:
		return "", fmt.Errorf("unknown command %q\n\n%s", command, usage())
	}
}

func parsePositiveInt(raw, label string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", label)
	}

	return value, nil
}

func parseNonNegativeInt(raw, label string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", label)
	}

	return value, nil
}

func ignoreNoChange(err error) error {
	if err == nil || errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return err
}

func runUp(err error) (string, error) {
	if errors.Is(err, migrate.ErrNoChange) {
		return "No pending migrations. Database is already up to date.", nil
	}
	if err != nil {
		return "", err
	}

	return "Migrations applied successfully.", nil
}

func usage() string {
	return `usage:
  go run ./cmd/migrate up
  go run ./cmd/migrate up 1
  go run ./cmd/migrate down
  go run ./cmd/migrate down 1
  go run ./cmd/migrate goto 1
  go run ./cmd/migrate force 1
  go run ./cmd/migrate version`
}
