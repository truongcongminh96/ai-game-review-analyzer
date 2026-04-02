package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
	platformmysql "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/mysql"
	platformpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/postgres"
	aiClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	steamClient "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
	reviewHTTP "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/delivery/http"
	reviewmysql "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/repository/mysql"
	reviewpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/repository/postgres"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/usecase"
)

func main() {
	cfg := config.Load()
	mux := http.NewServeMux()
	ctx := context.Background()

	gameRepo, analysisRepo, healthChecker, closeDatabase, err := initializePersistence(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer closeDatabase()

	ollama := aiClient.NewOllamaClient(cfg)
	steam := steamClient.NewClient()
	analyzeUseCase := usecase.NewAnalyzeUseCaseWithOptions(
		ollama,
		steam,
		gameRepo,
		analysisRepo,
		usecase.AnalyzeUseCaseOptions{
			BatchConfig: usecase.BatchConfig{
				MaxReviews: cfg.AnalysisBatchMaxReviews,
				MaxChars:   cfg.AnalysisBatchMaxChars,
			},
		},
	)

	handler := reviewHTTP.NewHandler(analyzeUseCase, healthChecker)
	reviewHTTP.RegisterRoutes(mux, handler)

	addr := ":" + cfg.ServerPort
	log.Printf("server running at %s", addr)

	if err := http.ListenAndServe(addr, reviewHTTP.WithCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func initializePersistence(
	ctx context.Context,
	cfg config.Config,
) (usecase.GameRepository, usecase.AnalysisRepository, reviewHTTP.HealthChecker, func(), error) {
	noopClose := func() {}

	switch cfg.DatabaseDriver {
	case "":
		log.Println("database integration disabled")
		return nil, nil, nil, noopClose, nil

	case config.DatabaseDriverPostgres:
		client, err := platformpostgres.New(ctx, cfg)
		if err != nil {
			return nil, nil, nil, noopClose, err
		}

		if client != nil {
			log.Println("postgres connection established")
		}

		return reviewpostgres.NewGameRepository(client), reviewpostgres.NewAnalysisRepository(client), client, func() {
			if client != nil {
				client.Close()
			}
		}, nil

	case config.DatabaseDriverMySQL:
		client, err := platformmysql.New(ctx, cfg)
		if err != nil {
			return nil, nil, nil, noopClose, err
		}

		if client != nil {
			log.Println("mysql connection established")
		}

		return reviewmysql.NewGameRepository(client), reviewmysql.NewAnalysisRepository(client), client, func() {
			if client != nil {
				client.Close()
			}
		}, nil

	default:
		return nil, nil, nil, noopClose, fmt.Errorf("unsupported DATABASE_DRIVER %q", cfg.DatabaseDriver)
	}
}
