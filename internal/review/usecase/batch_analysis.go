package usecase

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

const (
	mergedStructuredEvidenceMaxItems = 3
)

type reviewBatch struct {
	StartIndex int
	Reviews    []string
}

type structuredBatchResult struct {
	Offset int
	Report *model.StructuredInsight
}

func (u *AnalyzeUseCase) buildReviewBatches(reviews []string) []reviewBatch {
	if len(reviews) == 0 {
		return nil
	}

	maxReviews := u.batchConfig.MaxReviews
	maxChars := u.batchConfig.MaxChars
	batches := make([]reviewBatch, 0, max(1, len(reviews)/maxReviews+1))
	current := reviewBatch{StartIndex: 0}
	currentChars := 0

	for index, review := range reviews {
		reviewLen := len(review)
		if len(current.Reviews) > 0 && (len(current.Reviews) >= maxReviews || currentChars+reviewLen > maxChars) {
			batches = append(batches, current)
			current = reviewBatch{StartIndex: index}
			currentChars = 0
		}

		if len(current.Reviews) == 0 {
			current.StartIndex = index
		}
		current.Reviews = append(current.Reviews, review)
		currentChars += reviewLen
	}

	if len(current.Reviews) > 0 {
		batches = append(batches, current)
	}

	return batches
}

func (u *AnalyzeUseCase) analyzeReviewsInBatches(reviews []string) (*model.Insight, error) {
	batches := u.buildReviewBatches(reviews)
	if len(batches) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	if len(batches) == 1 {
		insight, err := u.aiClient.AnalyzeReviews(reviews)
		if err != nil {
			return nil, err
		}
		return sanitizeInsight(insight, len(reviews)), nil
	}

	partials := make([]*model.Insight, 0, len(batches))
	for _, batch := range batches {
		insight, err := u.aiClient.AnalyzeReviews(batch.Reviews)
		if err != nil {
			return nil, err
		}
		partials = append(partials, sanitizeInsight(insight, len(batch.Reviews)))
	}

	return sanitizeInsight(mergeLegacyInsights(partials, len(reviews)), len(reviews)), nil
}

func (u *AnalyzeUseCase) analyzeReviewsDetailedInBatches(reviewTexts []string) (*model.StructuredInsight, error) {
	return u.analyzeReviewsDetailedInBatchesWithProgress(reviewTexts, nil)
}

func (u *AnalyzeUseCase) analyzeReviewsDetailedInBatchesWithProgress(
	reviewTexts []string,
	onBatchComplete func(completed int, total int),
) (*model.StructuredInsight, error) {
	batches := u.buildReviewBatches(reviewTexts)
	if len(batches) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	if len(batches) == 1 {
		report, err := u.aiClient.AnalyzeReviewsDetailed(reviewTexts)
		if err != nil {
			return nil, err
		}
		return sanitizeStructuredInsight(report, reviewTexts), nil
	}

	results := make([]structuredBatchResult, 0, len(batches))
	for _, batch := range batches {
		report, err := u.aiClient.AnalyzeReviewsDetailed(batch.Reviews)
		if err != nil {
			return nil, err
		}
		results = append(results, structuredBatchResult{
			Offset: batch.StartIndex,
			Report: sanitizeStructuredInsight(report, batch.Reviews),
		})
		if onBatchComplete != nil {
			onBatchComplete(len(results), len(batches))
		}
	}

	return sanitizeStructuredInsight(mergeStructuredInsights(results, len(reviewTexts)), reviewTexts), nil
}

func mergeLegacyInsights(partials []*model.Insight, reviewCount int) *model.Insight {
	merged := &model.Insight{
		ReviewCount: reviewCount,
	}

	praised := make([][]string, 0, len(partials))
	issues := make([][]string, 0, len(partials))
	topics := make([][]string, 0, len(partials))

	for _, partial := range partials {
		if partial == nil {
			continue
		}

		merged.Sentiment.Positive += partial.Sentiment.Positive
		merged.Sentiment.Neutral += partial.Sentiment.Neutral
		merged.Sentiment.Negative += partial.Sentiment.Negative
		praised = append(praised, partial.PraisedFeatures)
		issues = append(issues, partial.CommonIssues)
		topics = append(topics, partial.Topics)
	}

	merged.PraisedFeatures = mergeLegacyLabels(praised, 5)
	merged.CommonIssues = mergeLegacyLabels(issues, 5)
	merged.Topics = mergeLegacyLabels(topics, 6)
	merged.Summary = buildMergedSummary(reviewCount, merged.PraisedFeatures, merged.CommonIssues, merged.Topics)

	return merged
}

func mergeLegacyLabels(groups [][]string, limit int) []string {
	type aggregate struct {
		Label      string
		Count      int
		FirstIndex int
	}

	aggregates := make(map[string]*aggregate)
	order := 0

	for _, group := range groups {
		for _, item := range group {
			label := strings.TrimSpace(item)
			if label == "" {
				continue
			}

			key := strings.ToLower(label)
			entry, ok := aggregates[key]
			if !ok {
				entry = &aggregate{
					Label:      label,
					FirstIndex: order,
				}
				aggregates[key] = entry
			}
			entry.Count++
			order++
		}
	}

	items := make([]aggregate, 0, len(aggregates))
	for _, entry := range aggregates {
		items = append(items, *entry)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			if items[i].FirstIndex == items[j].FirstIndex {
				return items[i].Label < items[j].Label
			}
			return items[i].FirstIndex < items[j].FirstIndex
		}
		return items[i].Count > items[j].Count
	})

	if len(items) > limit {
		items = items[:limit]
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Label)
	}

	return result
}

func mergeStructuredInsights(results []structuredBatchResult, reviewCount int) *model.StructuredInsight {
	merged := &model.StructuredInsight{}
	merged.Sentiment = mergeStructuredSentiment(results)
	merged.Praises = mergeStructuredItems(results, func(report *model.StructuredInsight) []model.StructuredInsightItem {
		return report.Praises
	}, false, 4)
	merged.Issues = mergeStructuredItems(results, func(report *model.StructuredInsight) []model.StructuredInsightItem {
		return report.Issues
	}, true, 4)
	merged.Topics = mergeStructuredItems(results, func(report *model.StructuredInsight) []model.StructuredInsightItem {
		return report.Topics
	}, false, 5)

	merged.Summary = buildMergedSummary(
		reviewCount,
		takeStructuredSummaryLabels(merged.Praises, 2),
		takeStructuredSummaryLabels(merged.Issues, 2),
		takeStructuredSummaryLabels(merged.Topics, 2),
	)

	return merged
}

func mergeStructuredSentiment(results []structuredBatchResult) model.SentimentBreakdown {
	merged := model.SentimentBreakdown{}
	for _, result := range results {
		if result.Report == nil {
			continue
		}
		merged.Positive += result.Report.Sentiment.Positive
		merged.Neutral += result.Report.Sentiment.Neutral
		merged.Negative += result.Report.Sentiment.Negative
	}
	return merged
}

func mergeStructuredItems(
	results []structuredBatchResult,
	selectItems func(*model.StructuredInsight) []model.StructuredInsightItem,
	issue bool,
	limit int,
) []model.StructuredInsightItem {
	type aggregate struct {
		Label          string
		Summary        string
		Count          int
		ConfidenceSum  float64
		BestConfidence float64
		SeveritySum    int
		SeverityCount  int
		FirstIndex     int
		Evidence       []model.EvidenceRef
		EvidenceSeen   map[string]struct{}
	}

	aggregates := make(map[string]*aggregate)
	order := 0

	for _, result := range results {
		if result.Report == nil {
			continue
		}

		for _, item := range shiftStructuredEvidenceRefs(selectItems(result.Report), result.Offset) {
			label := strings.TrimSpace(item.Label)
			if label == "" {
				continue
			}

			key := strings.ToLower(label)
			entry, ok := aggregates[key]
			if !ok {
				entry = &aggregate{
					Label:          label,
					Summary:        strings.TrimSpace(item.Summary),
					BestConfidence: item.Confidence,
					FirstIndex:     order,
					EvidenceSeen:   make(map[string]struct{}),
				}
				aggregates[key] = entry
			}

			entry.Count++
			entry.ConfidenceSum += item.Confidence
			if item.Confidence >= entry.BestConfidence && len(strings.TrimSpace(item.Summary)) >= len(entry.Summary) {
				entry.Summary = strings.TrimSpace(item.Summary)
				entry.BestConfidence = item.Confidence
			}
			if issue && item.Severity != nil {
				entry.SeveritySum += *item.Severity
				entry.SeverityCount++
			}

			for _, evidence := range item.Evidence {
				key := fmt.Sprintf("%d|%s", evidence.ReviewRef, strings.ToLower(strings.TrimSpace(evidence.Quote)))
				if _, seen := entry.EvidenceSeen[key]; seen {
					continue
				}
				entry.EvidenceSeen[key] = struct{}{}
				entry.Evidence = append(entry.Evidence, evidence)
			}

			order++
		}
	}

	items := make([]model.StructuredInsightItem, 0, len(aggregates))
	for _, entry := range aggregates {
		item := model.StructuredInsightItem{
			Label:      entry.Label,
			Summary:    strings.TrimSpace(entry.Summary),
			Confidence: clampConfidence(entry.ConfidenceSum / float64(entry.Count)),
			Evidence:   limitEvidenceRefs(entry.Evidence, mergedStructuredEvidenceMaxItems),
		}
		if item.Summary == "" {
			item.Summary = item.Label
		}
		if issue && entry.SeverityCount > 0 {
			value := int(math.Round(float64(entry.SeveritySum) / float64(entry.SeverityCount)))
			if value < 1 {
				value = 1
			}
			if value > 5 {
				value = 5
			}
			item.Severity = &value
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		left := aggregates[strings.ToLower(items[i].Label)]
		right := aggregates[strings.ToLower(items[j].Label)]
		if left.Count == right.Count {
			if items[i].Confidence == items[j].Confidence {
				if left.FirstIndex == right.FirstIndex {
					return items[i].Label < items[j].Label
				}
				return left.FirstIndex < right.FirstIndex
			}
			return items[i].Confidence > items[j].Confidence
		}
		return left.Count > right.Count
	})

	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func shiftStructuredEvidenceRefs(items []model.StructuredInsightItem, offset int) []model.StructuredInsightItem {
	if offset == 0 {
		return items
	}

	shifted := make([]model.StructuredInsightItem, 0, len(items))
	for _, item := range items {
		clone := item
		if len(item.Evidence) > 0 {
			clone.Evidence = make([]model.EvidenceRef, 0, len(item.Evidence))
			for _, evidence := range item.Evidence {
				if evidence.ReviewRef > 0 {
					evidence.ReviewRef += offset
				}
				clone.Evidence = append(clone.Evidence, evidence)
			}
		}
		shifted = append(shifted, clone)
	}

	return shifted
}

func limitEvidenceRefs(items []model.EvidenceRef, limit int) []model.EvidenceRef {
	if len(items) == 0 || limit <= 0 {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ReviewRef == items[j].ReviewRef {
			return items[i].Quote < items[j].Quote
		}
		return items[i].ReviewRef < items[j].ReviewRef
	})

	if len(items) > limit {
		items = items[:limit]
	}

	result := make([]model.EvidenceRef, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}

	return result
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*1000) / 1000
}

func buildMergedSummary(reviewCount int, praises []string, issues []string, topics []string) string {
	if reviewCount <= 0 {
		return "AI analysis completed."
	}

	sentences := make([]string, 0, 2)
	if len(praises) > 0 {
		sentences = append(sentences, fmt.Sprintf("Across %d reviews, players frequently praise %s.", reviewCount, joinSummaryLabels(praises)))
	} else if len(topics) > 0 {
		sentences = append(sentences, fmt.Sprintf("Across %d reviews, players frequently discuss %s.", reviewCount, joinSummaryLabels(topics)))
	} else if len(issues) > 0 {
		sentences = append(sentences, fmt.Sprintf("Across %d reviews, players frequently mention issues with %s.", reviewCount, joinSummaryLabels(issues)))
	}

	if len(issues) > 0 {
		sentences = append(sentences, fmt.Sprintf("Common issues center on %s.", joinSummaryLabels(issues)))
	} else if len(praises) == 0 && len(topics) > 0 {
		sentences = append(sentences, fmt.Sprintf("The strongest recurring themes are %s.", joinSummaryLabels(topics)))
	}

	if len(sentences) == 0 {
		return fmt.Sprintf("AI analysis completed from %d reviews.", reviewCount)
	}

	return strings.Join(sentences[:min(2, len(sentences))], " ")
}

func buildBatchAnalyzingProgressPercent(completed int, total int) int {
	if total <= 1 {
		return 65
	}
	if completed <= 0 {
		return 65
	}
	if completed >= total {
		return 85
	}

	return 65 + int(math.Round(float64(completed)*20/float64(total)))
}

func summarizeReviewBatches(batches []reviewBatch) string {
	if len(batches) == 0 {
		return "batches=0"
	}

	previewLimit := min(6, len(batches))
	sizes := make([]string, 0, previewLimit)
	totalReviews := 0
	minReviews := 0
	maxReviews := 0

	for index, batch := range batches {
		size := len(batch.Reviews)
		totalReviews += size
		if index == 0 || size < minReviews {
			minReviews = size
		}
		if size > maxReviews {
			maxReviews = size
		}
		if index < previewLimit {
			sizes = append(sizes, fmt.Sprintf("%d", size))
		}
	}

	suffix := ""
	if len(batches) > previewLimit {
		suffix = ",..."
	}

	averageReviews := int(math.Round(float64(totalReviews) / float64(len(batches))))
	return fmt.Sprintf(
		"batches=%d avg_reviews=%d min_reviews=%d max_reviews=%d sizes=[%s%s]",
		len(batches),
		averageReviews,
		minReviews,
		maxReviews,
		strings.Join(sizes, ","),
		suffix,
	)
}

func buildAnalysisDebugView(config BatchConfig, batches []reviewBatch) *model.AnalysisDebugView {
	if len(batches) == 0 {
		return nil
	}

	batchSizes := make([]int, 0, len(batches))
	for _, batch := range batches {
		batchSizes = append(batchSizes, len(batch.Reviews))
	}

	return &model.AnalysisDebugView{
		BatchCount:     len(batches),
		BatchSizeLimit: config.MaxReviews,
		BatchCharLimit: config.MaxChars,
		BatchSizes:     batchSizes,
	}
}

func buildQueueDebugView(config BatchConfig, reviewLimit int) *model.AnalysisQueueDebugView {
	if reviewLimit <= 0 {
		return nil
	}

	estimatedBatchCount := int(math.Ceil(float64(reviewLimit) / float64(config.MaxReviews)))
	if estimatedBatchCount < 1 {
		estimatedBatchCount = 1
	}

	return &model.AnalysisQueueDebugView{
		EstimatedBatchCount:       estimatedBatchCount,
		EstimatedReviewFetchPages: int(math.Ceil(float64(reviewLimit) / 100.0)),
		BatchSizeLimit:            config.MaxReviews,
		BatchCharLimit:            config.MaxChars,
	}
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
