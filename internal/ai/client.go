package ai

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/models"

type Client interface {
	AnalyzeReviews(reviews []string) (*models.Insight, error)
}
