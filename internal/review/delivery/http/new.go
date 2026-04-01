package http

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalyzeUseCase interface {
	AnalyzeReviews(ctx context.Context, reviews []string) (*model.Insight, error)
	AnalyzeSteamReviews(ctx context.Context, appID string, limit int, language string) (*model.Insight, error)
	RequestSteamAnalysis(ctx context.Context, appID string, limit int, language string) (*model.AnalysisRunQueued, error)
	GetAnalysisRun(ctx context.Context, runID string) (*model.AnalysisRunDetail, error)
	GetAnalysisEvidence(ctx context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error)
	GetGameHistory(ctx context.Context, appID string, limit int) (*model.GameHistory, error)
	CompareAnalysisRuns(ctx context.Context, runA string, runB string) (*model.CompareAnalysisResult, error)
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
