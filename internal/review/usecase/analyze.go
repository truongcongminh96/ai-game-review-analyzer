package usecase

import (
	"fmt"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

// AnalyzeReviews analyzes player reviews using the AI client
// and returns a summarized Insight.
func (u *AnalyzeUseCase) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	reviews = normalizeReviews(reviews)
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	insight, err := u.aiClient.AnalyzeReviews(reviews)
	if err != nil {
		return nil, err
	}

	return sanitizeInsight(insight, len(reviews)), nil
}

func (u *AnalyzeUseCase) AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error) {
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

func sanitizeInsight(insight *model.Insight, reviewCount int) *model.Insight {
	if insight == nil {
		return &model.Insight{ReviewCount: reviewCount}
	}

	insight.PraisedFeatures = cleanStringList(insight.PraisedFeatures)
	insight.CommonIssues = cleanStringList(insight.CommonIssues)
	insight.Topics = cleanStringList(insight.Topics)
	insight.Summary = strings.TrimSpace(insight.Summary)
	insight.ReviewCount = reviewCount

	total := insight.Sentiment.Positive + insight.Sentiment.Neutral + insight.Sentiment.Negative
	if total != reviewCount {
		insight.Sentiment.Positive = 0
		insight.Sentiment.Neutral = 0
		insight.Sentiment.Negative = 0
	}

	return insight
}

func cleanStringList(items []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(items))

	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
