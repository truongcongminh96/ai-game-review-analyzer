package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/api"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/config"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	addr := ":" + cfg.ServerPort
	log.Printf("Server running at http://localhost:%s", cfg.ServerPort)
	log.Printf("Ollama base URL: %s", cfg.OllamaBaseURL)
	log.Printf("Ollama model: %s", cfg.OllamaModel)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
