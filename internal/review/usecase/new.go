package usecase

import (
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
)

type AnalyzeUseCase struct {
	aiClient     ai.Client
	steamClient  steam.Client
	gameRepo     GameRepository
	analysisRepo AnalysisRepository
}

func NewAnalyzeUseCase(
	aiClient ai.Client,
	steamClient steam.Client,
	gameRepo GameRepository,
	analysisRepo AnalysisRepository,
) *AnalyzeUseCase {
	return &AnalyzeUseCase{
		aiClient:     aiClient,
		steamClient:  steamClient,
		gameRepo:     gameRepo,
		analysisRepo: analysisRepo,
	}
}
