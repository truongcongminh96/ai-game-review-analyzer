package usecase

import (
	"fmt"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/ai"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/client/steam"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalyzeUsecase struct {
	aiClient    ai.Client
	steamClient steam.Client
}

func NewAnalyzeUsecase(aiClient ai.Client, steamClient steam.Client) *AnalyzeUsecase {
	return &AnalyzeUsecase{
		aiClient:    aiClient,
		steamClient: steamClient,
	}
}

func (u *AnalyzeUsecase) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	reviews = normalizeReviews(reviews)
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	insight, err := u.aiClient.AnalyzeReviews(reviews)
	if err != nil {
		return nil, err
	}

	insight.ReviewCount = len(reviews)
	return insight, nil
}

func (u *AnalyzeUsecase) AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("appId required")
	}

	if limit <= 0 {
		limit = 30
	}

	if strings.TrimSpace(language) == "" {
		language = "english"
	}

	reviews, err := u.steamClient.GetReviews(appID, limit, language)
	if err != nil {
		return nil, err
	}

	return u.AnalyzeReviews(reviews)
}

func normalizeReviews(reviews []string) []string {
	normalized := make([]string, 0, len(reviews))
	for _, review := range reviews {
		trimmed := strings.TrimSpace(review)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}
