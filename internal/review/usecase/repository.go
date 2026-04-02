package usecase

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type GameRepository interface {
	UpsertBySteamAppID(ctx context.Context, input model.GameUpsertInput) (*model.Game, error)
}

type AnalysisRepository interface {
	CreateRun(ctx context.Context, input model.CreateAnalysisRunInput) (*model.AnalysisRun, error)
	StartRun(ctx context.Context, runID string) error
	UpdateRunProgress(ctx context.Context, input model.UpdateAnalysisRunProgressInput) error
	SaveReviewSnapshots(ctx context.Context, runID string, reviews []model.ReviewSnapshot) error
	CompleteRun(ctx context.Context, input model.CompleteAnalysisRunInput) error
	MarkFailed(ctx context.Context, input model.FailAnalysisRunInput) error
	GetRunDetail(ctx context.Context, runID string) (*model.AnalysisRunDetail, error)
	ListReviewTexts(ctx context.Context, runID string) ([]string, error)
	ListHistoryByAppID(ctx context.Context, appID string, limit int) (*model.GameHistory, error)
	ListEvidence(ctx context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error)
}
