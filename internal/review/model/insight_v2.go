package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AnalysisStage string

const (
	AnalysisStageQueued          AnalysisStage = "queued"
	AnalysisStageFetchingReviews AnalysisStage = "fetching_reviews"
	AnalysisStageStoringReviews  AnalysisStage = "storing_reviews"
	AnalysisStageAnalyzing       AnalysisStage = "analyzing"
	AnalysisStageSaving          AnalysisStage = "saving"
	AnalysisStageCompleted       AnalysisStage = "completed"
	AnalysisStageFailed          AnalysisStage = "failed"
)

type InsightKind string

const (
	InsightKindPraise InsightKind = "praise"
	InsightKindIssue  InsightKind = "issue"
	InsightKindTopic  InsightKind = "topic"
)

type EvidenceRef struct {
	ReviewRef int    `json:"review_ref"`
	Quote     string `json:"quote"`
}

var evidenceStringPattern = regexp.MustCompile(`(?i)^\s*review\s*(\d+)\s*[:\-]\s*(.+?)\s*$`)

type StructuredInsightItem struct {
	Label      string        `json:"label"`
	Summary    string        `json:"summary"`
	Severity   *int          `json:"severity,omitempty"`
	Confidence float64       `json:"confidence"`
	Evidence   []EvidenceRef `json:"evidence"`
}

func (i *StructuredInsightItem) UnmarshalJSON(data []byte) error {
	type alias struct {
		Label      string            `json:"label"`
		Summary    string            `json:"summary"`
		Severity   *int              `json:"severity,omitempty"`
		Confidence float64           `json:"confidence"`
		Evidence   []json.RawMessage `json:"evidence"`
	}

	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	evidence := make([]EvidenceRef, 0, len(raw.Evidence))
	for _, entry := range raw.Evidence {
		ref, err := unmarshalEvidenceRef(entry)
		if err != nil {
			return err
		}
		evidence = append(evidence, ref)
	}

	i.Label = raw.Label
	i.Summary = raw.Summary
	i.Severity = raw.Severity
	i.Confidence = raw.Confidence
	i.Evidence = evidence

	return nil
}

func unmarshalEvidenceRef(data []byte) (EvidenceRef, error) {
	var ref EvidenceRef
	if err := json.Unmarshal(data, &ref); err == nil {
		return ref, nil
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return EvidenceRef{}, err
	}

	return parseEvidenceString(raw), nil
}

func parseEvidenceString(raw string) EvidenceRef {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EvidenceRef{}
	}

	if matches := evidenceStringPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
		reviewRef, err := strconv.Atoi(matches[1])
		if err == nil {
			return EvidenceRef{
				ReviewRef: reviewRef,
				Quote:     trimEvidenceQuote(matches[2]),
			}
		}
	}

	return EvidenceRef{
		Quote: trimEvidenceQuote(trimmed),
	}
}

func trimEvidenceQuote(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Trim(trimmed, `"'`)
	return strings.TrimSpace(trimmed)
}

type StructuredInsight struct {
	Summary       string                  `json:"summary"`
	Sentiment     SentimentBreakdown      `json:"sentiment"`
	Praises       []StructuredInsightItem `json:"praises"`
	Issues        []StructuredInsightItem `json:"issues"`
	Topics        []StructuredInsightItem `json:"topics"`
	RawAIResponse json.RawMessage         `json:"-"`
}

func (s *StructuredInsight) ToLegacy(reviewCount int) *Insight {
	if s == nil {
		return &Insight{ReviewCount: reviewCount}
	}

	result := &Insight{
		Summary:     strings.TrimSpace(s.Summary),
		ReviewCount: reviewCount,
		Sentiment:   s.Sentiment,
	}

	result.PraisedFeatures = extractInsightLabels(s.Praises)
	result.CommonIssues = extractInsightLabels(s.Issues)
	result.Topics = extractInsightLabels(s.Topics)
	if result.Summary == "" {
		result.Summary = buildStructuredLegacySummary(s, reviewCount)
	}

	return result
}

func StructuredInsightFromLegacy(insight *Insight) *StructuredInsight {
	if insight == nil {
		return nil
	}

	return &StructuredInsight{
		Summary:   insight.Summary,
		Sentiment: insight.Sentiment,
		Praises:   buildStructuredItems(insight.PraisedFeatures),
		Issues:    buildStructuredItems(insight.CommonIssues),
		Topics:    buildStructuredItems(insight.Topics),
	}
}

func (i *Insight) ToStructured() *StructuredInsight {
	return StructuredInsightFromLegacy(i)
}

func extractInsightLabels(items []StructuredInsightItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item.Label != "" {
			result = append(result, item.Label)
		}
	}

	return result
}

func buildStructuredItems(labels []string) []StructuredInsightItem {
	result := make([]StructuredInsightItem, 0, len(labels))
	for _, label := range labels {
		result = append(result, StructuredInsightItem{
			Label:      label,
			Summary:    label,
			Confidence: 0.5,
			Evidence:   []EvidenceRef{},
		})
	}

	return result
}

func buildStructuredLegacySummary(report *StructuredInsight, reviewCount int) string {
	if report != nil {
		if labels := takeStructuredLegacyLabels(report.Praises, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently praise %s.", joinStructuredLegacyLabels(labels))
		}
		if labels := takeStructuredLegacyLabels(report.Topics, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently discuss %s.", joinStructuredLegacyLabels(labels))
		}
		if labels := takeStructuredLegacyLabels(report.Issues, 2); len(labels) > 0 {
			return fmt.Sprintf("Players frequently mention issues with %s.", joinStructuredLegacyLabels(labels))
		}
	}

	if reviewCount > 0 {
		return fmt.Sprintf("AI analysis completed from %d reviews.", reviewCount)
	}

	return "AI analysis completed."
}

func takeStructuredLegacyLabels(items []StructuredInsightItem, limit int) []string {
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

func joinStructuredLegacyLabels(labels []string) string {
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

type ReviewSnapshot struct {
	ID                 string     `json:"id"`
	AnalysisRunID      string     `json:"analysis_run_id"`
	Source             string     `json:"source"`
	SourceReviewID     string     `json:"source_review_id"`
	ReviewIndex        int        `json:"review_index"`
	ReviewText         string     `json:"review_text"`
	VotedUp            bool       `json:"voted_up"`
	Language           string     `json:"language"`
	HelpfulVotes       int        `json:"helpful_votes"`
	FunnyVotes         int        `json:"funny_votes"`
	WeightedVoteScore  float64    `json:"weighted_vote_score"`
	PlaytimeForeverMin int        `json:"playtime_forever_min"`
	ReviewedAt         *time.Time `json:"reviewed_at,omitempty"`
}

type UpdateAnalysisRunProgressInput struct {
	RunID           string
	Stage           AnalysisStage
	ProgressPercent int
}

type AnalysisRunQueued struct {
	RunID           string                  `json:"run_id"`
	Status          AnalysisStatus          `json:"status"`
	CurrentStage    AnalysisStage           `json:"current_stage"`
	ProgressPercent int                     `json:"progress_percent"`
	QueueDebug      *AnalysisQueueDebugView `json:"queue_debug,omitempty"`
	Request         struct {
		AppID    string `json:"app_id"`
		Limit    int    `json:"limit"`
		Language string `json:"language"`
	} `json:"request"`
}

type AnalysisQueueDebugView struct {
	EstimatedBatchCount       int `json:"estimated_batch_count"`
	EstimatedReviewFetchPages int `json:"estimated_review_fetch_pages"`
	BatchSizeLimit            int `json:"batch_size_limit"`
	BatchCharLimit            int `json:"batch_char_limit"`
}

type EvidenceView struct {
	ReviewID      string     `json:"review_id"`
	Quote         string     `json:"quote"`
	ReviewText    string     `json:"review_text,omitempty"`
	VotedUp       bool       `json:"voted_up"`
	Language      string     `json:"language"`
	HelpfulVotes  int        `json:"helpful_votes"`
	FunnyVotes    int        `json:"funny_votes"`
	PlaytimeHours float64    `json:"playtime_hours"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
}

type AnalysisItemView struct {
	ID             string         `json:"id"`
	Kind           InsightKind    `json:"kind"`
	Label          string         `json:"label"`
	Summary        string         `json:"summary"`
	Severity       *int           `json:"severity,omitempty"`
	Confidence     float64        `json:"confidence"`
	EvidenceCount  int            `json:"evidence_count"`
	SampleEvidence []EvidenceView `json:"sample_evidence"`
}

type AnalysisRunDetail struct {
	RunID           string             `json:"run_id"`
	Status          AnalysisStatus     `json:"status"`
	CurrentStage    AnalysisStage      `json:"current_stage"`
	ProgressPercent int                `json:"progress_percent"`
	RequestedAt     time.Time          `json:"requested_at"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	Game            GameView           `json:"game"`
	Overview        *Insight           `json:"overview"`
	Debug           *AnalysisDebugView `json:"debug,omitempty"`
	Praises         []AnalysisItemView `json:"praises"`
	Issues          []AnalysisItemView `json:"issues"`
	Topics          []AnalysisItemView `json:"topics"`
}

type AnalysisDebugView struct {
	BatchCount     int   `json:"batch_count"`
	BatchSizeLimit int   `json:"batch_size_limit"`
	BatchCharLimit int   `json:"batch_char_limit"`
	BatchSizes     []int `json:"batch_sizes,omitempty"`
}

type AnalysisHistoryItem struct {
	RunID       string             `json:"run_id"`
	RequestedAt time.Time          `json:"requested_at"`
	ReviewCount int                `json:"review_count"`
	Sentiment   SentimentBreakdown `json:"sentiment"`
	Summary     string             `json:"summary"`
}

type GameHistory struct {
	Game  GameView              `json:"game"`
	Items []AnalysisHistoryItem `json:"items"`
}

type AnalysisEvidenceQuery struct {
	RunID  string
	Kind   InsightKind
	Label  string
	Limit  int
	Cursor string
}

type AnalysisEvidencePage struct {
	Items      []EvidenceView `json:"items"`
	NextCursor *string        `json:"next_cursor,omitempty"`
}

type CompareRunRef struct {
	RunID string `json:"run_id"`
	Label string `json:"label"`
}

type CompareAnalysisItemChange struct {
	Label  string `json:"label"`
	Change string `json:"change"`
}

type CompareAnalysisResult struct {
	RunA           CompareRunRef               `json:"run_a"`
	RunB           CompareRunRef               `json:"run_b"`
	Summary        string                      `json:"summary"`
	SentimentDelta SentimentBreakdown          `json:"sentiment_delta"`
	Issues         []CompareAnalysisItemChange `json:"issues"`
}
