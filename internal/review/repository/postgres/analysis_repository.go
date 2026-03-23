package postgres

import (
	"context"
	"fmt"

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
		GameID: input.GameID,
	}

	query := `
		insert into public.analysis_runs (game_id, review_limit, language, genre)
		values ($1, $2, $3, $4)
		returning id
	`

	if err := r.pool.QueryRow(
		ctx,
		query,
		input.GameID,
		input.ReviewLimit,
		input.Language,
		input.Genre,
	).Scan(&run.ID); err != nil {
		return nil, fmt.Errorf("create analysis run: %w", err)
	}

	return run, nil
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

	if err := saveAnalysisResult(ctx, tx, input); err != nil {
		return err
	}

	if _, err := tx.Exec(
		ctx,
		`
			update public.analysis_runs
			set
				review_count = $2,
				status = 'success',
				completed_at = now(),
				error_message = null
			where id = $1
		`,
		input.RunID,
		input.ReviewCount,
	); err != nil {
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

	if _, err := r.pool.Exec(
		ctx,
		`
			update public.analysis_runs
			set
				review_count = $2,
				status = 'failed',
				completed_at = now(),
				error_message = $3
			where id = $1
		`,
		input.RunID,
		input.ReviewCount,
		input.ErrorMessage,
	); err != nil {
		return fmt.Errorf("mark analysis run as failed: %w", err)
	}

	return nil
}

func saveAnalysisResult(ctx context.Context, tx pgx.Tx, input model.CompleteAnalysisRunInput) error {
	if input.Insight == nil {
		return fmt.Errorf("insight is required to complete analysis run")
	}

	if _, err := tx.Exec(
		ctx,
		`
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
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			on conflict (analysis_run_id) do update
			set
				summary = excluded.summary,
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
		input.Insight.Summary,
		input.Insight.PraisedFeatures,
		input.Insight.CommonIssues,
		input.Insight.Topics,
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
