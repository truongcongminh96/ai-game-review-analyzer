package http

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type AnalyzeUseCase interface {
	AnalyzeReviews(reviews []string) (*model.Insight, error)
	AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error)
}

type Handler struct {
	useCase AnalyzeUseCase
}

func NewHandler(useCase AnalyzeUseCase) *Handler {
	return &Handler{useCase: useCase}
}
