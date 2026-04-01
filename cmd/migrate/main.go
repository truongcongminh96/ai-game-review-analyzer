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
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required to run migrations (legacy SUPABASE_DB_URL and MYSQL_DB_URL are still supported)")
	}

	if len(os.Args) < 2 {
		log.Fatal(usage())
	}

	m, err := newMigrator(cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}
	defer closeMigrator(m)

	command := os.Args[1]
	args := os.Args[2:]

	message, err := run(m, command, args)
	if err != nil {
		if dirty, ok := errors.AsType[migrate.ErrDirty](err); ok {
			log.Fatalf("database is dirty at version %d; fix the migration or run `go run ./cmd/migrate force <version>`", dirty.Version)
		}
		log.Fatal(err)
	}

	if message != "" {
		fmt.Println(message)
	}
}

func newMigrator(databaseDriver string, databaseURL string) (*migrate.Migrate, error) {
	driverName, err := normalizeMigrationDriver(databaseDriver)
	if err != nil {
		return nil, err
	}

	migrationsDir, err := filepath.Abs(filepath.Join("db", "migrations", "postgres"))
	if err != nil {
		return nil, fmt.Errorf("resolve postgres migrations dir: %w", err)
	}

	if driverName == config.DatabaseDriverMySQL {
		migrationsDir, err = filepath.Abs(filepath.Join("db", "migrations", "mysql"))
		if err != nil {
			return nil, fmt.Errorf("resolve mysql migrations dir: %w", err)
		}
	}

	if _, err := os.Stat(migrationsDir); err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(migrationsDir),
	}).String()

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database for migrations: %w", err)
	}

	switch driverName {
	case config.DatabaseDriverPostgres:
		driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create postgres migration driver: %w", err)
		}

		m, err := migrate.NewWithDatabaseInstance(sourceURL, driverName, driver)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create migrate instance: %w", err)
		}

		return m, nil

	case config.DatabaseDriverMySQL:
		_ = db.Close()

		m, err := migrate.New(sourceURL, normalizeMySQLMigrationURL(databaseURL))
		if err != nil {
			return nil, fmt.Errorf("create mysql migrate instance: %w", err)
		}

		return m, nil

	default:
		_ = db.Close()
		return nil, fmt.Errorf("unsupported DATABASE_DRIVER %q", driverName)
	}
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

		version, err := parseForceVersion(args[0])
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

func parseForceVersion(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < -1 {
		return 0, fmt.Errorf("force version must be -1 or a non-negative integer")
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
  go run ./cmd/migrate force -1
  go run ./cmd/migrate force 1
  go run ./cmd/migrate version`
}

func normalizeMigrationDriver(driver string) (string, error) {
	switch driver {
	case config.DatabaseDriverPostgres:
		return config.DatabaseDriverPostgres, nil
	case config.DatabaseDriverMySQL:
		return config.DatabaseDriverMySQL, nil
	case "":
		return "", fmt.Errorf("DATABASE_DRIVER is required or must be inferable from DATABASE_URL to run migrations")
	default:
		return "", fmt.Errorf("unsupported DATABASE_DRIVER %q", driver)
	}
}

func normalizeMySQLMigrationURL(databaseURL string) string {
	normalized := strings.TrimSpace(databaseURL)
	if strings.HasPrefix(strings.ToLower(normalized), "mysql://") {
		return normalized
	}

	return "mysql://" + normalized
}
