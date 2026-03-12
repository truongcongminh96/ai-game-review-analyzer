package config

import "os"

type Config struct {
	ServerPort    string
	OllamaBaseURL string
	OllamaModel   string
}

func Load() Config {
	return Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "llama3.2:3b"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
