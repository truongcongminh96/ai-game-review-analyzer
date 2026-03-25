package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	platformmysql "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/mysql"
	platformuuid "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/uuid"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalysisRepository struct {
	db *sql.DB
}

func NewAnalysisRepository(client *platformmysql.Client) *AnalysisRepository {
	if client == nil || !client.Enabled() {
		return nil
	}

	return &AnalysisRepository{
		db: client.DB(),
	}
}

func (r *AnalysisRepository) CreateRun(ctx context.Context, input model.CreateAnalysisRunInput) (*model.AnalysisRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("analysis repository is not configured")
	}

	runID, err := platformuuid.NewString()
	if err != nil {
		return nil, fmt.Errorf("generate analysis run id: %w", err)
	}

	run := &model.AnalysisRun{
		ID:     runID,
		GameID: input.GameID,
	}

	query := `
		insert into analysis_runs (id, game_id, review_limit, language)
		values (?, ?, ?, ?)
	`

	if _, err := r.db.ExecContext(
		ctx,
		query,
		run.ID,
		input.GameID,
		input.ReviewLimit,
		input.Language,
	); err != nil {
		return nil, fmt.Errorf("create analysis run: %w", err)
	}

	return run, nil
}

func (r *AnalysisRepository) CompleteRun(ctx context.Context, input model.CompleteAnalysisRunInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin complete analysis run transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := saveAnalysisResult(ctx, tx, input); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`
			update analysis_runs
			set
				review_count = ?,
				status = 'success',
				completed_at = current_timestamp(6),
				error_message = null
			where id = ?
		`,
		input.ReviewCount,
		input.RunID,
	); err != nil {
		return fmt.Errorf("mark analysis run as success: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit complete analysis run transaction: %w", err)
	}

	return nil
}

func (r *AnalysisRepository) MarkFailed(ctx context.Context, input model.FailAnalysisRunInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("analysis repository is not configured")
	}

	if _, err := r.db.ExecContext(
		ctx,
		`
			update analysis_runs
			set
				review_count = ?,
				status = 'failed',
				completed_at = current_timestamp(6),
				error_message = ?
			where id = ?
		`,
		input.ReviewCount,
		input.ErrorMessage,
		input.RunID,
	); err != nil {
		return fmt.Errorf("mark analysis run as failed: %w", err)
	}

	return nil
}

func saveAnalysisResult(ctx context.Context, tx *sql.Tx, input model.CompleteAnalysisRunInput) error {
	if input.Insight == nil {
		return fmt.Errorf("insight is required to complete analysis run")
	}

	resultID, err := platformuuid.NewString()
	if err != nil {
		return fmt.Errorf("generate analysis result id: %w", err)
	}

	praisedFeatures, err := json.Marshal(input.Insight.PraisedFeatures)
	if err != nil {
		return fmt.Errorf("marshal praised_features: %w", err)
	}

	commonIssues, err := json.Marshal(input.Insight.CommonIssues)
	if err != nil {
		return fmt.Errorf("marshal common_issues: %w", err)
	}

	topics, err := json.Marshal(input.Insight.Topics)
	if err != nil {
		return fmt.Errorf("marshal topics: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`
			insert into analysis_results (
				id,
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
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			on duplicate key update
				summary = values(summary),
				praised_features = values(praised_features),
				common_issues = values(common_issues),
				topics = values(topics),
				sentiment_positive = values(sentiment_positive),
				sentiment_neutral = values(sentiment_neutral),
				sentiment_negative = values(sentiment_negative),
				model_name = values(model_name),
				raw_ai_response = values(raw_ai_response),
				updated_at = current_timestamp(6)
		`,
		resultID,
		input.RunID,
		input.Insight.Summary,
		praisedFeatures,
		commonIssues,
		topics,
		input.Insight.Sentiment.Positive,
		input.Insight.Sentiment.Neutral,
		input.Insight.Sentiment.Negative,
		input.ModelName,
		nil,
	); err != nil {
		return fmt.Errorf("save analysis result: %w", err)
	}

	return nil
}
