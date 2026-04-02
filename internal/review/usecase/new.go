package usecase

import (
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
)

const (
	defaultAnalysisBatchMaxReviews = 60
	defaultAnalysisBatchMaxChars   = 18000
)

type BatchConfig struct {
	MaxReviews int
	MaxChars   int
}

type AnalyzeUseCaseOptions struct {
	BatchConfig BatchConfig
}

type AnalyzeUseCase struct {
	aiClient     ai.Client
	steamClient  steam.Client
	gameRepo     GameRepository
	analysisRepo AnalysisRepository
	batchConfig  BatchConfig
}

func NewAnalyzeUseCase(
	aiClient ai.Client,
	steamClient steam.Client,
	gameRepo GameRepository,
	analysisRepo AnalysisRepository,
) *AnalyzeUseCase {
	return NewAnalyzeUseCaseWithOptions(aiClient, steamClient, gameRepo, analysisRepo, AnalyzeUseCaseOptions{})
}

func NewAnalyzeUseCaseWithOptions(
	aiClient ai.Client,
	steamClient steam.Client,
	gameRepo GameRepository,
	analysisRepo AnalysisRepository,
	options AnalyzeUseCaseOptions,
) *AnalyzeUseCase {
	return &AnalyzeUseCase{
		aiClient:     aiClient,
		steamClient:  steamClient,
		gameRepo:     gameRepo,
		analysisRepo: analysisRepo,
		batchConfig:  normalizeBatchConfig(options.BatchConfig),
	}
}

func normalizeBatchConfig(config BatchConfig) BatchConfig {
	if config.MaxReviews <= 0 {
		config.MaxReviews = defaultAnalysisBatchMaxReviews
	}
	if config.MaxChars <= 0 {
		config.MaxChars = defaultAnalysisBatchMaxChars
	}
	return config
}
