package ai

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type Client interface {
	AnalyzeReviews(reviews []string) (*model.Insight, error)
	ModelName() string
}
