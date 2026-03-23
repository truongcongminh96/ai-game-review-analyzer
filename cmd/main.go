package main

import (
	"context"
	"log"
	"net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
	platformpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/postgres"
	aiClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	steamClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
	reviewHTTP "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/delivery/http"
	reviewpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/repository/postgres"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/usecase"
)

func main() {
	cfg := config.Load()
	mux := http.NewServeMux()
	ctx := context.Background()

	supabase, err := platformpostgres.New(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize supabase connection: %v", err)
	}
	if supabase != nil {
		defer supabase.Close()
		log.Println("supabase connection established")
	}

	ollama := aiClient.NewOllamaClient(cfg)
	steam := steamClient.NewClient()
	gameRepo := reviewpostgres.NewGameRepository(supabase)
	analysisRepo := reviewpostgres.NewAnalysisRepository(supabase)
	analyzeUseCase := usecase.NewAnalyzeUseCase(ollama, steam, gameRepo, analysisRepo)

	handler := reviewHTTP.NewHandler(analyzeUseCase, supabase)
	reviewHTTP.RegisterRoutes(mux, handler)

	addr := ":" + cfg.ServerPort
	log.Printf("server running at %s", addr)

	if err := http.ListenAndServe(addr, reviewHTTP.WithCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
