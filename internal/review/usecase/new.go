package usecase

import (
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
)

type AnalyzeUseCase struct {
	aiClient    ai.Client
	steamClient steam.Client
}

func NewAnalyzeUseCase(aiClient ai.Client, steamClient steam.Client) *AnalyzeUseCase {
	return &AnalyzeUseCase{
		aiClient:    aiClient,
		steamClient: steamClient,
	}
}
