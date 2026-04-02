package usecase

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

var evidenceTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

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
		QueueDebug:      buildQueueDebugView(u.batchConfig, limit),
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

	detail, err := u.analysisRepo.GetRunDetail(ctx, runID)
	if err != nil {
		return nil, err
	}

	reviewTexts, err := u.analysisRepo.ListReviewTexts(ctx, runID)
	if err != nil {
		log.Printf("warning: failed to load review texts for run_id=%s debug metadata: %v", runID, err)
		return detail, nil
	}

	batches := u.buildReviewBatches(reviewTexts)
	detail.Debug = buildAnalysisDebugView(u.batchConfig, batches)

	return detail, nil
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

	batches := u.buildReviewBatches(reviewTexts)
	if len(batches) > 1 {
		log.Printf(
			"steam async analysis run_id=%s app_id=%s review_count=%d model=%s %s",
			runID,
			appID,
			len(reviewTexts),
			u.aiClient.AdvancedModelName(),
			summarizeReviewBatches(batches),
		)
	}

	report, err := u.analyzeReviewsDetailedInBatchesWithProgress(reviewTexts, func(completed int, total int) {
		progress := buildBatchAnalyzingProgressPercent(completed, total)
		if progress <= 65 {
			return
		}

		_ = u.analysisRepo.UpdateRunProgress(ctx, model.UpdateAnalysisRunProgressInput{
			RunID:           runID,
			Stage:           model.AnalysisStageAnalyzing,
			ProgressPercent: progress,
		})
		log.Printf(
			"steam async analysis run_id=%s app_id=%s completed_batch=%d/%d progress_percent=%d",
			runID,
			appID,
			completed,
			total,
			progress,
		)
	})
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

	if err := u.analysisRepo.CompleteRun(ctx, model.CompleteAnalysisRunInput{
		RunID:       runID,
		ReviewCount: len(reviewTexts),
		Report:      report,
		ModelName:   u.aiClient.AdvancedModelName(),
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

func sanitizeStructuredInsight(report *model.StructuredInsight, reviewTexts []string) *model.StructuredInsight {
	if report == nil {
		return &model.StructuredInsight{}
	}

	report.Praises = sanitizeStructuredItems(report.Praises, false, reviewTexts)
	report.Issues = sanitizeStructuredItems(report.Issues, true, reviewTexts)
	report.Topics = sanitizeStructuredItems(report.Topics, false, reviewTexts)
	report.Summary = strings.TrimSpace(report.Summary)
	if report.Summary == "" {
		report.Summary = buildFallbackStructuredInsightSummary(report, len(reviewTexts))
	}

	return report
}

func buildFallbackStructuredInsightSummary(report *model.StructuredInsight, reviewCount int) string {
	if report != nil {
		if labels := takeStructuredSummaryLabels(report.Praises, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently praise %s.", joinSummaryLabels(labels))
		}
		if labels := takeStructuredSummaryLabels(report.Topics, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently discuss %s.", joinSummaryLabels(labels))
		}
		if labels := takeStructuredSummaryLabels(report.Issues, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently mention issues with %s.", joinSummaryLabels(labels))
		}
	}

	if reviewCount > 0 {
		return fmt.Sprintf("AI analysis completed from %d reviews.", reviewCount)
	}

	return "AI analysis completed."
}

func takeStructuredSummaryLabels(items []model.StructuredInsightItem, limit int) []string {
	if limit <= 0 || len(items) == 0 {
		return nil
	}

	labels := make([]string, 0, limit)
	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		labels = append(labels, label)
		if len(labels) == limit {
			break
		}
	}

	return labels
}

func sanitizeStructuredItems(items []model.StructuredInsightItem, issue bool, reviewTexts []string) []model.StructuredInsightItem {
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

		item.Evidence = sanitizeEvidenceRefs(item, reviewTexts)
		if len(item.Evidence) == 0 {
			item.Evidence = buildFallbackEvidenceRefs(item, reviewTexts, 2)
		}
		result = append(result, item)
	}

	return result
}

func sanitizeEvidenceRefs(item model.StructuredInsightItem, reviewTexts []string) []model.EvidenceRef {
	result := make([]model.EvidenceRef, 0, len(item.Evidence))
	seen := make(map[string]struct{})
	keywords := buildEvidenceKeywords(item)

	for _, evidence := range item.Evidence {
		evidence.Quote = strings.TrimSpace(evidence.Quote)
		if evidence.ReviewRef <= 0 || evidence.ReviewRef > len(reviewTexts) {
			continue
		}

		reviewText := reviewTexts[evidence.ReviewRef-1]
		if evidence.Quote == "" || !containsEvidenceQuote(reviewText, evidence.Quote) {
			evidence.Quote = extractEvidenceQuote(reviewText, keywords)
		}
		if evidence.Quote == "" {
			continue
		}

		key := fmt.Sprintf("%d|%s", evidence.ReviewRef, strings.ToLower(evidence.Quote))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, evidence)
	}

	return result
}

func buildFallbackEvidenceRefs(item model.StructuredInsightItem, reviewTexts []string, limit int) []model.EvidenceRef {
	if limit <= 0 || len(reviewTexts) == 0 {
		return nil
	}

	keywords := buildEvidenceKeywords(item)
	type candidate struct {
		ref   int
		score int
		quote string
	}

	candidates := make([]candidate, 0, len(reviewTexts))
	for index, reviewText := range reviewTexts {
		score := scoreEvidenceMatch(reviewText, keywords)
		if score == 0 {
			continue
		}

		quote := extractEvidenceQuote(reviewText, keywords)
		if quote == "" {
			continue
		}

		candidates = append(candidates, candidate{
			ref:   index + 1,
			score: score,
			quote: quote,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].ref < candidates[j].ref
		}
		return candidates[i].score > candidates[j].score
	})

	result := make([]model.EvidenceRef, 0, limit)
	seenQuotes := make(map[string]struct{})
	for _, candidate := range candidates {
		key := strings.ToLower(candidate.quote)
		if _, ok := seenQuotes[key]; ok {
			continue
		}
		seenQuotes[key] = struct{}{}
		result = append(result, model.EvidenceRef{
			ReviewRef: candidate.ref,
			Quote:     candidate.quote,
		})
		if len(result) == limit {
			break
		}
	}

	return result
}

func buildEvidenceKeywords(item model.StructuredInsightItem) []string {
	rawTokens := evidenceTokenPattern.FindAllString(strings.ToLower(item.Label+" "+item.Summary), -1)
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {},
		"into": {}, "are": {}, "was": {}, "were": {}, "have": {}, "has": {}, "had": {},
		"too": {}, "very": {}, "game": {}, "players": {}, "player": {}, "about": {},
		"some": {}, "many": {}, "more": {}, "less": {}, "still": {}, "than": {},
		"when": {}, "where": {}, "their": {}, "them": {}, "they": {}, "feel": {},
	}

	keywords := make([]string, 0, len(rawTokens))
	seen := make(map[string]struct{})
	for _, token := range rawTokens {
		if len(token) <= 2 {
			continue
		}
		if _, ok := stopWords[token]; ok {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		keywords = append(keywords, token)
	}

	return keywords
}

func scoreEvidenceMatch(reviewText string, keywords []string) int {
	normalized := strings.ToLower(reviewText)
	score := 0
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			score++
		}
	}

	return score
}

func extractEvidenceQuote(reviewText string, keywords []string) string {
	segments := splitEvidenceSegments(reviewText)
	bestSegment := ""
	bestScore := 0

	for _, segment := range segments {
		score := scoreEvidenceMatch(segment, keywords)
		if score > bestScore {
			bestScore = score
			bestSegment = segment
		}
	}

	if bestScore == 0 {
		if len(segments) == 0 {
			return ""
		}
		bestSegment = segments[0]
	}

	return trimEvidenceQuote(bestSegment, 180)
}

func splitEvidenceSegments(reviewText string) []string {
	parts := strings.FieldsFunc(reviewText, func(r rune) bool {
		switch r {
		case '.', '!', '?', '\n', '\r':
			return true
		default:
			return false
		}
	})

	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}

	return segments
}

func trimEvidenceQuote(text string, maxLen int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}

	cut := text[:maxLen]
	lastSpace := strings.LastIndex(cut, " ")
	if lastSpace > 0 {
		cut = cut[:lastSpace]
	}

	return strings.TrimSpace(cut)
}

func containsEvidenceQuote(reviewText string, quote string) bool {
	reviewNormalized := strings.ToLower(strings.Join(strings.Fields(reviewText), " "))
	quoteNormalized := strings.ToLower(strings.Join(strings.Fields(quote), " "))
	return strings.Contains(reviewNormalized, quoteNormalized)
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
