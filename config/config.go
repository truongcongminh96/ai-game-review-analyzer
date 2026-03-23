package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort                 string
	OllamaBaseURL              string
	OllamaModel                string
	OllamaTimeoutSec           int
	SupabaseDBURL              string
	SupabaseDBMaxConns         int
	SupabaseDBMinConns         int
	SupabaseDBHealthTimeoutSec int
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("warning: .env file not found, using system env")
	}

	return Config{
		ServerPort:                 getEnv("SERVER_PORT", "8080"),
		OllamaBaseURL:              getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:                getEnv("OLLAMA_MODEL", "llama3.2:3b"),
		OllamaTimeoutSec:           getEnvAsInt("OLLAMA_TIMEOUT_SEC", 300),
		SupabaseDBURL:              getFirstEnv("", "SUPABASE_DB_URL", "DATABASE_URL"),
		SupabaseDBMaxConns:         getEnvAsInt("SUPABASE_DB_MAX_CONNS", 5),
		SupabaseDBMinConns:         getEnvAsInt("SUPABASE_DB_MIN_CONNS", 0),
		SupabaseDBHealthTimeoutSec: getEnvAsInt("SUPABASE_DB_HEALTH_TIMEOUT_SEC", 5),
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
