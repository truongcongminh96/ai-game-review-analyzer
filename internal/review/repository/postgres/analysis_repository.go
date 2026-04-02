package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/postgres"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalysisRepository struct {
	pool *pgxpool.Pool
}

func NewAnalysisRepository(client *platformpostgres.Client) *AnalysisRepository {
	if client == nil || !client.Enabled() {
		return nil
	}

	return &AnalysisRepository{
		pool: client.Pool(),
	}
}

func (r *AnalysisRepository) CreateRun(ctx context.Context, input model.CreateAnalysisRunInput) (*model.AnalysisRun, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("analysis repository is not configured")
	}

	run := &model.AnalysisRun{
		GameID:          input.GameID,
		Status:          model.AnalysisStatusPending,
		CurrentStage:    model.AnalysisStageQueued,
		ProgressPercent: 0,
	}

	query := `
		insert into public.analysis_runs (game_id, review_limit, language)
		values ($1, $2, $3)
		returning id
	`

	if err := r.pool.QueryRow(ctx, query, input.GameID, input.ReviewLimit, input.Language).Scan(&run.ID); err != nil {
		return nil, fmt.Errorf("create analysis run: %w", err)
	}

	return run, nil
}

func (r *AnalysisRepository) StartRun(ctx context.Context, runID string) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	if _, err := r.pool.Exec(ctx, `
		update public.analysis_runs
		set started_at = coalesce(started_at, now()),
			current_stage = 'queued',
			progress_percent = 1
		where id = $1
	`, runID); err != nil {
		return fmt.Errorf("start analysis run: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) UpdateRunProgress(ctx context.Context, input model.UpdateAnalysisRunProgressInput) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	if _, err := r.pool.Exec(ctx, `
		update public.analysis_runs
		set current_stage = $2,
			progress_percent = $3
		where id = $1
	`, input.RunID, input.Stage, input.ProgressPercent); err != nil {
		return fmt.Errorf("update analysis run progress: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) SaveReviewSnapshots(ctx context.Context, runID string, reviews []model.ReviewSnapshot) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin save review snapshots transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, review := range reviews {
		if _, err := tx.Exec(ctx, `
			insert into public.review_snapshots (
				analysis_run_id,
				source,
				source_review_id,
				review_index,
				review_text,
				voted_up,
				language,
				helpful_votes,
				funny_votes,
				weighted_vote_score,
				steam_created_at,
				playtime_forever_min
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			on conflict (analysis_run_id, review_index) do update
			set source = excluded.source,
				source_review_id = excluded.source_review_id,
				review_text = excluded.review_text,
				voted_up = excluded.voted_up,
				language = excluded.language,
				helpful_votes = excluded.helpful_votes,
				funny_votes = excluded.funny_votes,
				weighted_vote_score = excluded.weighted_vote_score,
				steam_created_at = excluded.steam_created_at,
				playtime_forever_min = excluded.playtime_forever_min,
				updated_at = now()
		`,
			runID,
			review.Source,
			nullIfEmpty(review.SourceReviewID),
			review.ReviewIndex,
			review.ReviewText,
			review.VotedUp,
			review.Language,
			review.HelpfulVotes,
			review.FunnyVotes,
			review.WeightedVoteScore,
			review.ReviewedAt,
			review.PlaytimeForeverMin,
		); err != nil {
			return fmt.Errorf("save review snapshot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save review snapshots transaction: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) CompleteRun(ctx context.Context, input model.CompleteAnalysisRunInput) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin complete analysis run transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	report := normalizeStructuredReport(input)
	if err := saveAnalysisResult(ctx, tx, input, report); err != nil {
		return err
	}

	if err := replaceInsightItems(ctx, tx, input.RunID, report); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		update public.analysis_runs
		set review_count = $2,
			status = 'success',
			current_stage = 'completed',
			progress_percent = 100,
			completed_at = now(),
			error_message = null
		where id = $1
	`, input.RunID, input.ReviewCount); err != nil {
		return fmt.Errorf("mark analysis run as success: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit complete analysis run transaction: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) MarkFailed(ctx context.Context, input model.FailAnalysisRunInput) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	if _, err := r.pool.Exec(ctx, `
		update public.analysis_runs
		set review_count = $2,
			status = 'failed',
			current_stage = 'failed',
			progress_percent = 100,
			completed_at = now(),
			error_message = $3
		where id = $1
	`, input.RunID, input.ReviewCount, input.ErrorMessage); err != nil {
		return fmt.Errorf("mark analysis run as failed: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) GetRunDetail(ctx context.Context, runID string) (*model.AnalysisRunDetail, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("analysis repository is not configured")
	}

	var (
		detail       model.AnalysisRunDetail
		reviewCount  int
		coverURL     sql.NullString
		genre        sql.NullString
		releaseYear  sql.NullInt32
		errorMessage sql.NullString
		summary      sql.NullString
		startedAt    sql.NullTime
		completedAt  sql.NullTime
		posSentiment sql.NullInt32
		neuSentiment sql.NullInt32
		negSentiment sql.NullInt32
	)

	query := `
		select
			ar.id,
			ar.status,
			ar.current_stage,
			ar.progress_percent,
			ar.requested_at,
			ar.started_at,
			ar.completed_at,
			ar.review_count,
			g.steam_app_id,
			g.title,
			g.cover_url,
			g.genre,
			g.release_year,
			ar.error_message,
			res.summary,
			res.sentiment_positive,
			res.sentiment_neutral,
			res.sentiment_negative
		from public.analysis_runs ar
		join public.games g on g.id = ar.game_id
		left join public.analysis_results res on res.analysis_run_id = ar.id
		where ar.id = $1
	`

	if err := r.pool.QueryRow(ctx, query, runID).Scan(
		&detail.RunID,
		&detail.Status,
		&detail.CurrentStage,
		&detail.ProgressPercent,
		&detail.RequestedAt,
		&startedAt,
		&completedAt,
		&reviewCount,
		&detail.Game.AppID,
		&detail.Game.Title,
		&coverURL,
		&genre,
		&releaseYear,
		&errorMessage,
		&summary,
		&posSentiment,
		&neuSentiment,
		&negSentiment,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("analysis run not found")
		}
		return nil, fmt.Errorf("get analysis run detail: %w", err)
	}

	if startedAt.Valid {
		detail.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		detail.CompletedAt = &completedAt.Time
	}
	if coverURL.Valid {
		detail.Game.CoverURL = coverURL.String
	}
	if genre.Valid {
		detail.Game.Genre = genre.String
	}
	if releaseYear.Valid {
		year := int(releaseYear.Int32)
		detail.Game.ReleaseYear = &year
	}
	if errorMessage.Valid {
		detail.ErrorMessage = errorMessage.String
	}

	detail.Overview = &model.Insight{
		Summary:     summary.String,
		ReviewCount: reviewCount,
		Sentiment: model.SentimentBreakdown{
			Positive: int(posSentiment.Int32),
			Neutral:  int(neuSentiment.Int32),
			Negative: int(negSentiment.Int32),
		},
	}

	praises, err := r.loadAnalysisItems(ctx, runID, model.InsightKindPraise)
	if err != nil {
		return nil, err
	}
	issues, err := r.loadAnalysisItems(ctx, runID, model.InsightKindIssue)
	if err != nil {
		return nil, err
	}
	topics, err := r.loadAnalysisItems(ctx, runID, model.InsightKindTopic)
	if err != nil {
		return nil, err
	}

	detail.Praises = praises
	detail.Issues = issues
	detail.Topics = topics

	return &detail, nil
}

func (r *AnalysisRepository) ListHistoryByAppID(ctx context.Context, appID string, limit int) (*model.GameHistory, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("analysis repository is not configured")
	}

	history := &model.GameHistory{}

	var coverURL sql.NullString
	var genre sql.NullString
	var releaseYear sql.NullInt32
	if err := r.pool.QueryRow(ctx, `
		select steam_app_id, title, cover_url, genre, release_year
		from public.games
		where steam_app_id = $1
	`, appID).Scan(&history.Game.AppID, &history.Game.Title, &coverURL, &genre, &releaseYear); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("load game history root: %w", err)
	}

	if coverURL.Valid {
		history.Game.CoverURL = coverURL.String
	}
	if genre.Valid {
		history.Game.Genre = genre.String
	}
	if releaseYear.Valid {
		year := int(releaseYear.Int32)
		history.Game.ReleaseYear = &year
	}

	rows, err := r.pool.Query(ctx, `
		select
			ar.id,
			ar.requested_at,
			ar.review_count,
			coalesce(res.summary, ''),
			coalesce(res.sentiment_positive, 0),
			coalesce(res.sentiment_neutral, 0),
			coalesce(res.sentiment_negative, 0)
		from public.analysis_runs ar
		join public.games g on g.id = ar.game_id
		left join public.analysis_results res on res.analysis_run_id = ar.id
		where g.steam_app_id = $1
		order by ar.requested_at desc
		limit $2
	`, appID, limit)
	if err != nil {
		return nil, fmt.Errorf("list game history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.AnalysisHistoryItem
		if err := rows.Scan(
			&item.RunID,
			&item.RequestedAt,
			&item.ReviewCount,
			&item.Summary,
			&item.Sentiment.Positive,
			&item.Sentiment.Neutral,
			&item.Sentiment.Negative,
		); err != nil {
			return nil, fmt.Errorf("scan game history: %w", err)
		}
		history.Items = append(history.Items, item)
	}

	return history, rows.Err()
}

func (r *AnalysisRepository) ListEvidence(ctx context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("analysis repository is not configured")
	}

	query := `
		select
			rs.id,
			e.quote,
			rs.review_text,
			rs.voted_up,
			rs.language,
			rs.helpful_votes,
			rs.funny_votes,
			rs.playtime_forever_min,
			rs.steam_created_at
		from public.analysis_item_evidence e
		join public.analysis_insight_items i on i.id = e.insight_item_id
		join public.review_snapshots rs on rs.id = e.review_snapshot_id
		where i.analysis_run_id = $1
		  and i.kind = $2
		  and lower(i.label) = lower($3)
		order by rs.helpful_votes desc, rs.review_index asc
		limit $4
	`

	rows, err := r.pool.Query(ctx, query, input.RunID, input.Kind, input.Label, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("list evidence: %w", err)
	}
	defer rows.Close()

	page := &model.AnalysisEvidencePage{
		Items: make([]model.EvidenceView, 0),
	}
	for rows.Next() {
		item, err := scanEvidenceRow(rows)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, item)
	}

	return page, rows.Err()
}

func (r *AnalysisRepository) loadAnalysisItems(ctx context.Context, runID string, kind model.InsightKind) ([]model.AnalysisItemView, error) {
	rows, err := r.pool.Query(ctx, `
		select id, kind, label, summary, severity, confidence, evidence_count
		from public.analysis_insight_items
		where analysis_run_id = $1 and kind = $2
		order by sort_order asc, label asc
	`, runID, kind)
	if err != nil {
		return nil, fmt.Errorf("load analysis items: %w", err)
	}
	defer rows.Close()

	items := make([]model.AnalysisItemView, 0)
	for rows.Next() {
		var (
			item     model.AnalysisItemView
			severity sql.NullInt32
		)

		if err := rows.Scan(&item.ID, &item.Kind, &item.Label, &item.Summary, &severity, &item.Confidence, &item.EvidenceCount); err != nil {
			return nil, fmt.Errorf("scan analysis item: %w", err)
		}
		if severity.Valid {
			value := int(severity.Int32)
			item.Severity = &value
		}

		sampleEvidence, err := r.loadEvidenceByItemID(ctx, item.ID, 3, false)
		if err != nil {
			return nil, err
		}
		item.SampleEvidence = sampleEvidence
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *AnalysisRepository) loadEvidenceByItemID(ctx context.Context, itemID string, limit int, includeReviewText bool) ([]model.EvidenceView, error) {
	query := `
		select
			rs.id,
			e.quote,
			rs.review_text,
			rs.voted_up,
			rs.language,
			rs.helpful_votes,
			rs.funny_votes,
			rs.playtime_forever_min,
			rs.steam_created_at
		from public.analysis_item_evidence e
		join public.review_snapshots rs on rs.id = e.review_snapshot_id
		where e.insight_item_id = $1
		order by rs.helpful_votes desc, rs.review_index asc
		limit $2
	`

	rows, err := r.pool.Query(ctx, query, itemID, limit)
	if err != nil {
		return nil, fmt.Errorf("load evidence by item id: %w", err)
	}
	defer rows.Close()

	items := make([]model.EvidenceView, 0)
	for rows.Next() {
		item, err := scanEvidenceRow(rows)
		if err != nil {
			return nil, err
		}
		if !includeReviewText {
			item.ReviewText = ""
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func scanEvidenceRow(scanner interface {
	Scan(dest ...any) error
}) (model.EvidenceView, error) {
	var (
		item        model.EvidenceView
		reviewedAt  sql.NullTime
		playtimeMin int
	)

	if err := scanner.Scan(
		&item.ReviewID,
		&item.Quote,
		&item.ReviewText,
		&item.VotedUp,
		&item.Language,
		&item.HelpfulVotes,
		&item.FunnyVotes,
		&playtimeMin,
		&reviewedAt,
	); err != nil {
		return model.EvidenceView{}, fmt.Errorf("scan evidence row: %w", err)
	}

	item.PlaytimeHours = float64(playtimeMin) / 60.0
	if reviewedAt.Valid {
		item.ReviewedAt = &reviewedAt.Time
	}

	return item, nil
}

func saveAnalysisResult(ctx context.Context, tx pgx.Tx, input model.CompleteAnalysisRunInput, report *model.StructuredInsight) error {
	legacy := input.Insight
	if legacy == nil && report != nil {
		legacy = report.ToLegacy(input.ReviewCount)
	}
	if legacy == nil {
		return fmt.Errorf("insight is required to complete analysis run")
	}

	praisedFeatures, err := json.Marshal(legacy.PraisedFeatures)
	if err != nil {
		return fmt.Errorf("marshal praised_features: %w", err)
	}
	commonIssues, err := json.Marshal(legacy.CommonIssues)
	if err != nil {
		return fmt.Errorf("marshal common_issues: %w", err)
	}
	topics, err := json.Marshal(legacy.Topics)
	if err != nil {
		return fmt.Errorf("marshal topics: %w", err)
	}

	var rawAIResponse any
	if report != nil && len(report.RawAIResponse) > 0 {
		rawAIResponse = string(report.RawAIResponse)
	}

	if _, err := tx.Exec(ctx, `
		insert into public.analysis_results (
			analysis_run_id,
			summary,
			praised_features,
			common_issues,
			topics,
			sentiment_positive,
			sentiment_neutral,
			sentiment_negative,
			model_name,
			raw_ai_response
		)
		values ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7, $8, $9, $10::jsonb)
		on conflict (analysis_run_id) do update
		set summary = excluded.summary,
			praised_features = excluded.praised_features,
			common_issues = excluded.common_issues,
			topics = excluded.topics,
			sentiment_positive = excluded.sentiment_positive,
			sentiment_neutral = excluded.sentiment_neutral,
			sentiment_negative = excluded.sentiment_negative,
			model_name = excluded.model_name,
			raw_ai_response = excluded.raw_ai_response,
			updated_at = now()
	`,
		input.RunID,
		legacy.Summary,
		string(praisedFeatures),
		string(commonIssues),
		string(topics),
		legacy.Sentiment.Positive,
		legacy.Sentiment.Neutral,
		legacy.Sentiment.Negative,
		input.ModelName,
		rawAIResponse,
	); err != nil {
		return fmt.Errorf("save analysis result: %w", err)
	}

	return nil
}

func replaceInsightItems(ctx context.Context, tx pgx.Tx, runID string, report *model.StructuredInsight) error {
	if report == nil {
		return nil
	}

	if _, err := tx.Exec(ctx, `delete from public.analysis_insight_items where analysis_run_id = $1`, runID); err != nil {
		return fmt.Errorf("clear previous insight items: %w", err)
	}

	snapshotIDs, err := loadSnapshotIDs(ctx, tx, runID)
	if err != nil {
		return err
	}

	for index, item := range report.Praises {
		if err := insertInsightItem(ctx, tx, runID, model.InsightKindPraise, index, item, snapshotIDs); err != nil {
			return err
		}
	}
	for index, item := range report.Issues {
		if err := insertInsightItem(ctx, tx, runID, model.InsightKindIssue, index, item, snapshotIDs); err != nil {
			return err
		}
	}
	for index, item := range report.Topics {
		if err := insertInsightItem(ctx, tx, runID, model.InsightKindTopic, index, item, snapshotIDs); err != nil {
			return err
		}
	}

	return nil
}

func loadSnapshotIDs(ctx context.Context, tx pgx.Tx, runID string) (map[int]string, error) {
	rows, err := tx.Query(ctx, `
		select id, review_index
		from public.review_snapshots
		where analysis_run_id = $1
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("load review snapshot ids: %w", err)
	}
	defer rows.Close()

	result := make(map[int]string)
	for rows.Next() {
		var id string
		var reviewIndex int
		if err := rows.Scan(&id, &reviewIndex); err != nil {
			return nil, fmt.Errorf("scan review snapshot id: %w", err)
		}
		result[reviewIndex] = id
	}

	return result, rows.Err()
}

func insertInsightItem(ctx context.Context, tx pgx.Tx, runID string, kind model.InsightKind, sortOrder int, item model.StructuredInsightItem, snapshotIDs map[int]string) error {
	evidence := filterEvidenceRefs(item.Evidence, snapshotIDs)

	var (
		itemID   string
		severity any
	)
	if item.Severity != nil {
		severity = *item.Severity
	}

	if err := tx.QueryRow(ctx, `
		insert into public.analysis_insight_items (
			analysis_run_id,
			kind,
			label,
			summary,
			severity,
			confidence,
			evidence_count,
			sort_order
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning id
	`, runID, kind, item.Label, item.Summary, severity, item.Confidence, len(evidence), sortOrder).Scan(&itemID); err != nil {
		return fmt.Errorf("insert analysis insight item: %w", err)
	}

	for _, ref := range evidence {
		if _, err := tx.Exec(ctx, `
			insert into public.analysis_item_evidence (
				insight_item_id,
				review_snapshot_id,
				quote,
				confidence
			)
			values ($1, $2, $3, $4)
		`, itemID, snapshotIDs[ref.ReviewRef], ref.Quote, item.Confidence); err != nil {
			return fmt.Errorf("insert analysis item evidence: %w", err)
		}
	}

	return nil
}

func filterEvidenceRefs(items []model.EvidenceRef, snapshotIDs map[int]string) []model.EvidenceRef {
	result := make([]model.EvidenceRef, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		if _, ok := snapshotIDs[item.ReviewRef]; !ok {
			continue
		}
		quote := strings.TrimSpace(item.Quote)
		if quote == "" {
			continue
		}
		key := fmt.Sprintf("%d|%s", item.ReviewRef, strings.ToLower(quote))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, model.EvidenceRef{
			ReviewRef: item.ReviewRef,
			Quote:     quote,
		})
	}

	return result
}

func normalizeStructuredReport(input model.CompleteAnalysisRunInput) *model.StructuredInsight {
	if input.Report != nil {
		return input.Report
	}
	if input.Insight != nil {
		return model.StructuredInsightFromLegacy(input.Insight)
	}

	return nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}
