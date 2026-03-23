package http

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalyzeUseCase interface {
	AnalyzeReviews(reviews []string) (*model.Insight, error)
	AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error)
}

type HealthChecker interface {
	Enabled() bool
	CheckHealth(ctx context.Context) error
}

type Handler struct {
	useCase       AnalyzeUseCase
	healthChecker HealthChecker
}

type noopHealthChecker struct{}

func (noopHealthChecker) Enabled() bool {
	return false
}

func (noopHealthChecker) CheckHealth(context.Context) error {
	return nil
}

func NewHandler(useCase AnalyzeUseCase, healthChecker ...HealthChecker) *Handler {
	var checker HealthChecker = noopHealthChecker{}
	if len(healthChecker) > 0 && healthChecker[0] != nil {
		checker = healthChecker[0]
	}

	return &Handler{
		useCase:       useCase,
		healthChecker: checker,
	}
}
