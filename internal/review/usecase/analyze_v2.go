package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func (u *AnalyzeUseCase) RequestSteamAnalysis(ctx context.Context, appID string, limit int, language string) (*model.AnalysisRunQueued, error) {
	if !u.persistenceEnabled() {
		return nil, fmt.Errorf("database-backed analysis is not enabled")
	}

	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("appId required")
	}

	if limit <= 0 {
		limit = 30
	}

	if strings.TrimSpace(language) == "" {
		language = "english"
	}

	run, err := u.prepareAnalysisRun(ctx, appID, limit, language)
	if err != nil {
		return nil, err
	}

	go u.runSteamAnalysis(context.WithoutCancel(ctx), run.ID, appID, limit, language)

	response := &model.AnalysisRunQueued{
		RunID:           run.ID,
		Status:          model.AnalysisStatusPending,
		CurrentStage:    model.AnalysisStageQueued,
		ProgressPercent: 0,
	}
	response.Request.AppID = appID
	response.Request.Limit = limit
	response.Request.Language = language

	return response, nil
}

func (u *AnalyzeUseCase) GetAnalysisRun(ctx context.Context, runID string) (*model.AnalysisRunDetail, error) {
	if !u.persistenceEnabled() {
		return nil, fmt.Errorf("database-backed analysis is not enabled")
	}

	if strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("runID required")
	}

	return u.analysisRepo.GetRunDetail(ctx, runID)
}

func (u *AnalyzeUseCase) GetAnalysisEvidence(ctx context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error) {
	if !u.persistenceEnabled() {
		return nil, fmt.Errorf("database-backed analysis is not enabled")
	}

	input.RunID = strings.TrimSpace(input.RunID)
	input.Label = strings.TrimSpace(input.Label)
	if input.RunID == "" {
		return nil, fmt.Errorf("runID required")
	}
	if input.Kind == "" {
		return nil, fmt.Errorf("kind required")
	}
	if input.Label == "" {
		return nil, fmt.Errorf("label required")
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}

	return u.analysisRepo.ListEvidence(ctx, input)
}

func (u *AnalyzeUseCase) GetGameHistory(ctx context.Context, appID string, limit int) (*model.GameHistory, error) {
	if !u.persistenceEnabled() {
		return nil, fmt.Errorf("database-backed analysis is not enabled")
	}

	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("appId required")
	}
	if limit <= 0 {
		limit = 10
	}

	return u.analysisRepo.ListHistoryByAppID(ctx, appID, limit)
}

func (u *AnalyzeUseCase) CompareAnalysisRuns(ctx context.Context, runA string, runB string) (*model.CompareAnalysisResult, error) {
	if !u.persistenceEnabled() {
		return nil, fmt.Errorf("database-backed analysis is not enabled")
	}

	runA = strings.TrimSpace(runA)
	runB = strings.TrimSpace(runB)
	if runA == "" || runB == "" {
		return nil, fmt.Errorf("runA and runB required")
	}

	left, err := u.analysisRepo.GetRunDetail(ctx, runA)
	if err != nil {
		return nil, err
	}

	right, err := u.analysisRepo.GetRunDetail(ctx, runB)
	if err != nil {
		return nil, err
	}

	result := &model.CompareAnalysisResult{
		RunA: model.CompareRunRef{
			RunID: runA,
			Label: "Previous",
		},
		RunB: model.CompareRunRef{
			RunID: runB,
			Label: "Current",
		},
		SentimentDelta: model.SentimentBreakdown{
			Positive: right.Overview.Sentiment.Positive - left.Overview.Sentiment.Positive,
			Neutral:  right.Overview.Sentiment.Neutral - left.Overview.Sentiment.Neutral,
			Negative: right.Overview.Sentiment.Negative - left.Overview.Sentiment.Negative,
		},
	}

	result.Issues = compareIssueChanges(left.Issues, right.Issues)
	result.Summary = buildCompareSummary(result.SentimentDelta, result.Issues)

	return result, nil
}

func (u *AnalyzeUseCase) runSteamAnalysis(ctx context.Context, runID string, appID string, limit int, language string) {
	if err := u.analysisRepo.StartRun(ctx, runID); err != nil {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ErrorMessage: err.Error(),
		})
		return
	}

	_ = u.analysisRepo.UpdateRunProgress(ctx, model.UpdateAnalysisRunProgressInput{
		RunID:           runID,
		Stage:           model.AnalysisStageFetchingReviews,
		ProgressPercent: 15,
	})

	rawReviews, err := u.steamClient.GetReviews(appID, limit, language)
	if err != nil {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ErrorMessage: err.Error(),
		})
		return
	}

	snapshots, reviewTexts := toSnapshots(runID, rawReviews)
	if len(reviewTexts) == 0 {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ErrorMessage: "reviews cannot be empty",
		})
		return
	}

	_ = u.analysisRepo.UpdateRunProgress(ctx, model.UpdateAnalysisRunProgressInput{
		RunID:           runID,
		Stage:           model.AnalysisStageStoringReviews,
		ProgressPercent: 35,
	})

	if err := u.analysisRepo.SaveReviewSnapshots(ctx, runID, snapshots); err != nil {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ReviewCount:  len(reviewTexts),
			ErrorMessage: err.Error(),
		})
		return
	}

	_ = u.analysisRepo.UpdateRunProgress(ctx, model.UpdateAnalysisRunProgressInput{
		RunID:           runID,
		Stage:           model.AnalysisStageAnalyzing,
		ProgressPercent: 65,
	})

	report, err := u.aiClient.AnalyzeReviewsDetailed(reviewTexts)
	if err != nil {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ReviewCount:  len(reviewTexts),
			ErrorMessage: err.Error(),
		})
		return
	}

	_ = u.analysisRepo.UpdateRunProgress(ctx, model.UpdateAnalysisRunProgressInput{
		RunID:           runID,
		Stage:           model.AnalysisStageSaving,
		ProgressPercent: 90,
	})

	report = sanitizeStructuredInsight(report, len(reviewTexts))
	if err := u.analysisRepo.CompleteRun(ctx, model.CompleteAnalysisRunInput{
		RunID:       runID,
		ReviewCount: len(reviewTexts),
		Report:      report,
		ModelName:   u.aiClient.ModelName(),
	}); err != nil {
		_ = u.analysisRepo.MarkFailed(ctx, model.FailAnalysisRunInput{
			RunID:        runID,
			ReviewCount:  len(reviewTexts),
			ErrorMessage: err.Error(),
		})
	}
}

func toSnapshots(runID string, steamReviews []model.ReviewSteam) ([]model.ReviewSnapshot, []string) {
	snapshots := make([]model.ReviewSnapshot, 0, len(steamReviews))
	reviewTexts := make([]string, 0, len(steamReviews))

	for _, review := range steamReviews {
		trimmed := strings.TrimSpace(review.Review)
		if trimmed == "" {
			continue
		}

		index := len(reviewTexts) + 1
		reviewTexts = append(reviewTexts, trimmed)

		var reviewedAt *time.Time
		if review.TimestampCreated > 0 {
			t := time.Unix(review.TimestampCreated, 0).UTC()
			reviewedAt = &t
		}

		snapshots = append(snapshots, model.ReviewSnapshot{
			AnalysisRunID:      runID,
			Source:             "steam",
			SourceReviewID:     strings.TrimSpace(review.RecommendationID),
			ReviewIndex:        index,
			ReviewText:         trimmed,
			VotedUp:            review.VotedUp,
			Language:           strings.TrimSpace(review.Language),
			HelpfulVotes:       review.VotesUp,
			FunnyVotes:         review.VotesFunny,
			WeightedVoteScore:  float64(review.WeightedVoteScore),
			PlaytimeForeverMin: review.Author.PlaytimeForever,
			ReviewedAt:         reviewedAt,
		})
	}

	return snapshots, reviewTexts
}

func sanitizeStructuredInsight(report *model.StructuredInsight, reviewCount int) *model.StructuredInsight {
	if report == nil {
		return (&model.StructuredInsight{}).ToLegacy(reviewCount).ToStructured()
	}

	report.Summary = strings.TrimSpace(report.Summary)
	report.Praises = sanitizeStructuredItems(report.Praises, false)
	report.Issues = sanitizeStructuredItems(report.Issues, true)
	report.Topics = sanitizeStructuredItems(report.Topics, false)

	return report
}

func sanitizeStructuredItems(items []model.StructuredInsightItem, issue bool) []model.StructuredInsightItem {
	seen := make(map[string]struct{})
	result := make([]model.StructuredInsightItem, 0, len(items))

	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}

		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		item.Label = label
		item.Summary = strings.TrimSpace(item.Summary)
		if item.Summary == "" {
			item.Summary = label
		}
		if item.Confidence < 0 {
			item.Confidence = 0
		}
		if item.Confidence > 1 {
			item.Confidence = 1
		}
		if issue && item.Severity != nil {
			if *item.Severity < 1 {
				value := 1
				item.Severity = &value
			}
			if *item.Severity > 5 {
				value := 5
				item.Severity = &value
			}
		}
		if !issue {
			item.Severity = nil
		}

		item.Evidence = sanitizeEvidenceRefs(item.Evidence)
		result = append(result, item)
	}

	return result
}

func sanitizeEvidenceRefs(items []model.EvidenceRef) []model.EvidenceRef {
	result := make([]model.EvidenceRef, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		item.Quote = strings.TrimSpace(item.Quote)
		if item.ReviewRef <= 0 || item.Quote == "" {
			continue
		}
		key := fmt.Sprintf("%d|%s", item.ReviewRef, strings.ToLower(item.Quote))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}

	return result
}

func compareIssueChanges(left []model.AnalysisItemView, right []model.AnalysisItemView) []model.CompareAnalysisItemChange {
	leftMap := make(map[string]struct{}, len(left))
	rightMap := make(map[string]struct{}, len(right))

	for _, item := range left {
		leftMap[strings.ToLower(item.Label)] = struct{}{}
	}
	for _, item := range right {
		rightMap[strings.ToLower(item.Label)] = struct{}{}
	}

	changes := make([]model.CompareAnalysisItemChange, 0, len(left)+len(right))
	for _, item := range right {
		key := strings.ToLower(item.Label)
		if _, ok := leftMap[key]; ok {
			continue
		}
		changes = append(changes, model.CompareAnalysisItemChange{
			Label:  item.Label,
			Change: "new",
		})
	}

	for _, item := range left {
		key := strings.ToLower(item.Label)
		if _, ok := rightMap[key]; ok {
			continue
		}
		changes = append(changes, model.CompareAnalysisItemChange{
			Label:  item.Label,
			Change: "resolved",
		})
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Label < changes[j].Label
	})

	return changes
}

func buildCompareSummary(delta model.SentimentBreakdown, changes []model.CompareAnalysisItemChange) string {
	summary := []string{
		fmt.Sprintf("Positive %+d, neutral %+d, negative %+d.", delta.Positive, delta.Neutral, delta.Negative),
	}

	if len(changes) > 0 {
		top := changes[0]
		summary = append(summary, fmt.Sprintf("Issue change: %s is %s.", top.Label, top.Change))
	}

	return strings.Join(summary, " ")
}
