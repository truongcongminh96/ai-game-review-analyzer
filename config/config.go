package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	DatabaseDriverPostgres = "postgres"
	DatabaseDriverMySQL    = "mysql"
)

type Config struct {
	ServerPort               string
	OllamaBaseURL            string
	OllamaModel              string
	OllamaModelV1            string
	OllamaModelV2            string
	OllamaTimeoutSec         int
	DatabaseDriver           string
	DatabaseURL              string
	DatabaseMaxConns         int
	DatabaseMinConns         int
	DatabaseHealthTimeoutSec int
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("warning: .env file not found, using system env")
	}

	databaseDriver := resolveDatabaseDriver()
	ollamaModel := getEnv("OLLAMA_MODEL", "llama3.2:3b")

	return Config{
		ServerPort:               getEnv("SERVER_PORT", "8080"),
		OllamaBaseURL:            getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:              ollamaModel,
		OllamaModelV1:            getFirstEnv(ollamaModel, "OLLAMA_MODEL_V1", "OLLAMA_MODEL"),
		OllamaModelV2:            getFirstEnv(ollamaModel, "OLLAMA_MODEL_V2", "OLLAMA_MODEL"),
		OllamaTimeoutSec:         getEnvAsInt("OLLAMA_TIMEOUT_SEC", 300),
		DatabaseDriver:           databaseDriver,
		DatabaseURL:              resolveDatabaseURL(databaseDriver),
		DatabaseMaxConns:         resolveDatabaseInt(databaseDriver, 5, "DATABASE_MAX_CONNS"),
		DatabaseMinConns:         resolveDatabaseInt(databaseDriver, 0, "DATABASE_MIN_CONNS"),
		DatabaseHealthTimeoutSec: resolveDatabaseInt(databaseDriver, 5, "DATABASE_HEALTH_TIMEOUT_SEC"),
	}
}

func resolveDatabaseDriver() string {
	if driver := normalizeDatabaseDriver(getFirstEnv("", "DATABASE_DRIVER", "DB_DRIVER")); driver != "" {
		return driver
	}

	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		return inferDriverFromURL(databaseURL)
	}

	if strings.TrimSpace(os.Getenv("SUPABASE_DB_URL")) != "" {
		return DatabaseDriverPostgres
	}

	if strings.TrimSpace(os.Getenv("MYSQL_DB_URL")) != "" {
		return DatabaseDriverMySQL
	}

	return ""
}

func resolveDatabaseURL(driver string) string {
	switch driver {
	case DatabaseDriverMySQL:
		return getFirstEnv("", "DATABASE_URL", "MYSQL_DB_URL")
	case DatabaseDriverPostgres:
		return getFirstEnv("", "DATABASE_URL", "SUPABASE_DB_URL")
	default:
		return getFirstEnv("", "DATABASE_URL", "SUPABASE_DB_URL", "MYSQL_DB_URL")
	}
}

func resolveDatabaseInt(driver string, fallback int, genericKey string) int {
	keys := []string{genericKey}

	switch driver {
	case DatabaseDriverMySQL:
		if mysqlKey := mysqlDatabaseEnvKey(genericKey); mysqlKey != "" {
			keys = append(keys, mysqlKey)
		}
	case DatabaseDriverPostgres:
		if postgresKey := postgresDatabaseEnvKey(genericKey); postgresKey != "" {
			keys = append(keys, postgresKey)
		}
	default:
		if postgresKey := postgresDatabaseEnvKey(genericKey); postgresKey != "" {
			keys = append(keys, postgresKey)
		}
		if mysqlKey := mysqlDatabaseEnvKey(genericKey); mysqlKey != "" {
			keys = append(keys, mysqlKey)
		}
	}

	return getFirstEnvAsInt(fallback, keys...)
}

func postgresDatabaseEnvKey(genericKey string) string {
	switch genericKey {
	case "DATABASE_MAX_CONNS":
		return "SUPABASE_DB_MAX_CONNS"
	case "DATABASE_MIN_CONNS":
		return "SUPABASE_DB_MIN_CONNS"
	case "DATABASE_HEALTH_TIMEOUT_SEC":
		return "SUPABASE_DB_HEALTH_TIMEOUT_SEC"
	default:
		return ""
	}
}

func mysqlDatabaseEnvKey(genericKey string) string {
	switch genericKey {
	case "DATABASE_MAX_CONNS":
		return "MYSQL_DB_MAX_CONNS"
	case "DATABASE_MIN_CONNS":
		return "MYSQL_DB_MIN_CONNS"
	case "DATABASE_HEALTH_TIMEOUT_SEC":
		return "MYSQL_DB_HEALTH_TIMEOUT_SEC"
	default:
		return ""
	}
}

func normalizeDatabaseDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "":
		return ""
	case "postgres", "postgresql":
		return DatabaseDriverPostgres
	case "mysql":
		return DatabaseDriverMySQL
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func inferDriverFromURL(databaseURL string) string {
	normalized := strings.ToLower(strings.TrimSpace(databaseURL))

	switch {
	case strings.HasPrefix(normalized, "postgres://"), strings.HasPrefix(normalized, "postgresql://"):
		return DatabaseDriverPostgres
	case strings.HasPrefix(normalized, "mysql://"):
		return DatabaseDriverMySQL
	case strings.Contains(normalized, "@tcp("), strings.Contains(normalized, "@unix("):
		return DatabaseDriverMySQL
	case strings.Contains(normalized, "host=") && strings.Contains(normalized, "user="):
		return DatabaseDriverPostgres
	case strings.Contains(normalized, "dbname="):
		return DatabaseDriverPostgres
	default:
		return ""
	}
}

func getFirstEnv(fallback string, keys ...string) string {
	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			return value
		}
	}

	return fallback
}

func getFirstEnvAsInt(fallback int, keys ...string) int {
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}

		return intValue
	}

	return fallback
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return intValue
}
