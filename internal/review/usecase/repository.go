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
	CompleteRun(ctx context.Context, input model.CompleteAnalysisRunInput) error
	MarkFailed(ctx context.Context, input model.FailAnalysisRunInput) error
}
