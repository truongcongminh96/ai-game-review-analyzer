package main

import (
	"log"
	"net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
	aiClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	steamClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
	reviewHTTP "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/delivery/http"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/usecase"
)

func main() {
	cfg := config.Load()
	mux := http.NewServeMux()

	ollama := aiClient.NewOllamaClient()
	steam := steamClient.NewClient()
	analyzeUsecase := usecase.NewAnalyzeUsecase(ollama, steam)

	handler := reviewHTTP.NewHandler(analyzeUsecase)
	reviewHTTP.RegisterRoutes(mux, handler)

	addr := ":" + cfg.ServerPort
	log.Printf("server running at %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
