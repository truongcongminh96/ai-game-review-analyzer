package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

// AnalyzeReviews analyzes player reviews using the AI client
// and returns a summarized Insight.
func (u *AnalyzeUseCase) AnalyzeReviews(_ context.Context, reviews []string) (*model.Insight, error) {
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

func (u *AnalyzeUseCase) AnalyzeSteamReviews(ctx context.Context, appID string, limit int, language string) (*model.Insight, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("appId required")
	}

	if limit <= 0 {
		limit = 30
	}

	if strings.TrimSpace(language) == "" {
		language = "english"
	}

	var persistedRun *model.AnalysisRun
	if u.persistenceEnabled() {
		run, err := u.prepareAnalysisRun(ctx, appID, limit, language)
		if err != nil {
			return nil, err
		}
		persistedRun = run
	}

	steamReviews, err := u.steamClient.GetReviews(appID, limit, language)
	if err != nil {
		if markErr := u.markRunFailed(ctx, persistedRun, 0, err); markErr != nil {
			return nil, fmt.Errorf("steam fetch failed: %v; additionally failed to update analysis run: %w", err, markErr)
		}
		return nil, err
	}

	reviews := make([]string, 0, len(steamReviews))
	sentiment := model.SentimentBreakdown{}

	for _, r := range steamReviews {
		trimmed := strings.TrimSpace(r.Review)
		if trimmed == "" {
			continue
		}

		reviews = append(reviews, trimmed)

		if r.VotedUp {
			sentiment.Positive++
		} else {
			sentiment.Negative++
		}
	}

	insight, err := u.AnalyzeReviews(ctx, reviews)
	if err != nil {
		if markErr := u.markRunFailed(ctx, persistedRun, len(reviews), err); markErr != nil {
			return nil, fmt.Errorf("analysis failed: %v; additionally failed to update analysis run: %w", err, markErr)
		}
		return nil, err
	}

	insight.Sentiment = sentiment
	insight.ReviewCount = len(reviews)

	if err := u.completeRun(ctx, persistedRun, insight); err != nil {
		return nil, err
	}

	return insight, nil
}

func (u *AnalyzeUseCase) persistenceEnabled() bool {
	return u.gameRepo != nil && u.analysisRepo != nil
}

func (u *AnalyzeUseCase) prepareAnalysisRun(ctx context.Context, appID string, limit int, language string) (*model.AnalysisRun, error) {
	gameInput := buildGameUpsertInput(appID)

	details, err := u.steamClient.GetGameDetails(appID)
	if err == nil && details != nil {
		gameInput = applyGameDetails(gameInput, details)
	}

	game, err := u.gameRepo.UpsertBySteamAppID(ctx, gameInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert game: %w", err)
	}

	run, err := u.analysisRepo.CreateRun(ctx, model.CreateAnalysisRunInput{
		GameID:      game.ID,
		ReviewLimit: limit,
		Language:    language,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis run: %w", err)
	}

	return run, nil
}

func (u *AnalyzeUseCase) markRunFailed(ctx context.Context, run *model.AnalysisRun, reviewCount int, cause error) error {
	if run == nil || u.analysisRepo == nil || cause == nil {
		return nil
	}

	return u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
		RunID:        run.ID,
		ReviewCount:  reviewCount,
		ErrorMessage: cause.Error(),
	})
}

func (u *AnalyzeUseCase) completeRun(ctx context.Context, run *model.AnalysisRun, insight *model.Insight) error {
	if run == nil || u.analysisRepo == nil {
		return nil
	}

	report := model.StructuredInsightFromLegacy(insight)
	if err := u.analysisRepo.CompleteRun(ctx, model.CompleteAnalysisRunInput{
		RunID:       run.ID,
		ReviewCount: insight.ReviewCount,
		Insight:     insight,
		Report:      report,
		ModelName:   u.aiClient.StandardModelName(),
	}); err != nil {
		if markErr := u.markRunFailed(ctx, run, insight.ReviewCount, err); markErr != nil {
			return fmt.Errorf("failed to save analysis result: %v; additionally failed to mark analysis run as failed: %w", err, markErr)
		}
		return fmt.Errorf("failed to save analysis result: %w", err)
	}

	return nil
}

func buildGameUpsertInput(appID string) model.GameUpsertInput {
	return model.GameUpsertInput{
		SteamAppID:          appID,
		Title:               fmt.Sprintf("Steam App %s", appID),
		PreferExistingTitle: true,
	}
}

func applyGameDetails(input model.GameUpsertInput, details *model.SteamGameDetails) model.GameUpsertInput {
	if details == nil {
		return input
	}

	if title := strings.TrimSpace(details.Title); title != "" {
		input.Title = title
		input.PreferExistingTitle = false
	}
	input.CoverURL = optionalString(details.CoverURL)
	input.Genre = optionalString(details.Genre)
	input.ReleaseYear = details.ReleaseYear

	return input
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
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
		insight = &model.Insight{}
	}

	insight.PraisedFeatures = cleanStringList(insight.PraisedFeatures)
	insight.CommonIssues = cleanStringList(insight.CommonIssues)
	insight.Topics = cleanStringList(insight.Topics)
	insight.Summary = strings.TrimSpace(insight.Summary)
	if insight.Summary == "" {
		insight.Summary = buildFallbackInsightSummary(insight, reviewCount)
	}
	insight.ReviewCount = reviewCount

	total := insight.Sentiment.Positive + insight.Sentiment.Neutral + insight.Sentiment.Negative
	if total < 0 {
		insight.Sentiment.Positive = 0
		insight.Sentiment.Neutral = 0
		insight.Sentiment.Negative = 0
	}

	return insight
}

func buildFallbackInsightSummary(insight *model.Insight, reviewCount int) string {
	if insight != nil {
		if labels := takeSummaryLabels(insight.PraisedFeatures, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently praise %s.", joinSummaryLabels(labels))
		}
		if labels := takeSummaryLabels(insight.Topics, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently discuss %s.", joinSummaryLabels(labels))
		}
		if labels := takeSummaryLabels(insight.CommonIssues, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently mention issues with %s.", joinSummaryLabels(labels))
		}
	}

	if reviewCount > 0 {
		return fmt.Sprintf("AI analysis completed from %d reviews.", reviewCount)
	}

	return "AI analysis completed."
}

func takeSummaryLabels(items []string, limit int) []string {
	if limit <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) < limit {
		limit = len(items)
	}

	labels := make([]string, 0, limit)
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		labels = append(labels, trimmed)
		if len(labels) == limit {
			break
		}
	}

	return labels
}

func joinSummaryLabels(labels []string) string {
	switch len(labels) {
	case 0:
		return ""
	case 1:
		return labels[0]
	case 2:
		return labels[0] + " and " + labels[1]
	default:
		return strings.Join(labels[:len(labels)-1], ", ") + ", and " + labels[len(labels)-1]
	}
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
