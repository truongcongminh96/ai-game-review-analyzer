package analyze

import (
	"fmt"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/ai"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/models"
)

type Service struct {
	ollamaClient *ai.OllamaClient
}

func NewService() *Service {
	return &Service{
		ollamaClient: ai.NewOllamaClient(),
	}
}

func (s *Service) AnalyzeReviews(reviews []string) (*models.Insight, error) {
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	return s.ollamaClient.AnalyzeReviews(reviews)
}
