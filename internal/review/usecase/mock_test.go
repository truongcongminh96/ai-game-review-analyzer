package usecase

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type mockAIClient struct {
	result         *model.Insight
	resultQueue    []*model.Insight
	detailedResult *model.StructuredInsight
	detailedQueue  []*model.StructuredInsight
	err            error
	model          string
	standardModel  string
	advancedModel  string
	reviewCalls    [][]string
	detailedCalls  [][]string
}

func (m *mockAIClient) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	m.reviewCalls = append(m.reviewCalls, append([]string(nil), reviews...))
	if len(m.resultQueue) > 0 {
		result := m.resultQueue[0]
		m.resultQueue = m.resultQueue[1:]
		return result, m.err
	}
	return m.result, m.err
}

func (m *mockAIClient) AnalyzeReviewsDetailed(reviews []string) (*model.StructuredInsight, error) {
	m.detailedCalls = append(m.detailedCalls, append([]string(nil), reviews...))
	if len(m.detailedQueue) > 0 {
		result := m.detailedQueue[0]
		m.detailedQueue = m.detailedQueue[1:]
		return result, m.err
	}
	if m.detailedResult != nil || m.err != nil {
		return m.detailedResult, m.err
	}
	return model.StructuredInsightFromLegacy(m.result), nil
}

func (m *mockAIClient) StandardModelName() string {
	if m.standardModel != "" {
		return m.standardModel
	}
	if m.model != "" {
		return m.model
	}
	return m.advancedModel
}

func (m *mockAIClient) AdvancedModelName() string {
	if m.advancedModel != "" {
		return m.advancedModel
	}
	if m.model != "" {
		return m.model
	}
	return m.standardModel
}

type mockSteamClient struct {
	reviews   []model.ReviewSteam
	err       error
	details   *model.SteamGameDetails
	detailErr error

	gotAppID    string
	gotLimit    int
	gotLanguage string
}

func (m *mockSteamClient) GetReviews(appID string, limit int, language string) ([]model.ReviewSteam, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	m.gotLanguage = language
	return m.reviews, m.err
}

func (m *mockSteamClient) GetGameDetails(appID string) (*model.SteamGameDetails, error) {
	m.gotAppID = appID
	return m.details, m.detailErr
}

type mockGameRepository struct {
	game     *model.Game
	err      error
	gotInput model.GameUpsertInput
}

func (m *mockGameRepository) UpsertBySteamAppID(_ context.Context, input model.GameUpsertInput) (*model.Game, error) {
	m.gotInput = input
	return m.game, m.err
}

type mockAnalysisRepository struct {
	run         *model.AnalysisRun
	detail      *model.AnalysisRunDetail
	history     *model.GameHistory
	evidence    *model.AnalysisEvidencePage
	createErr   error
	completeErr error
	failErr     error
	startErr    error
	progressErr error
	snapshotErr error
	detailErr   error
	historyErr  error
	evidenceErr error

	gotCreateInput   model.CreateAnalysisRunInput
	gotCompleteInput model.CompleteAnalysisRunInput
	gotFailInput     model.FailAnalysisRunInput
	gotProgressInput model.UpdateAnalysisRunProgressInput
	progressInputs   []model.UpdateAnalysisRunProgressInput
	gotSnapshotsRun  string
	gotSnapshots     []model.ReviewSnapshot
	reviewTexts      []string
}

func (m *mockAnalysisRepository) CreateRun(_ context.Context, input model.CreateAnalysisRunInput) (*model.AnalysisRun, error) {
	m.gotCreateInput = input
	return m.run, m.createErr
}

func (m *mockAnalysisRepository) CompleteRun(_ context.Context, input model.CompleteAnalysisRunInput) error {
	m.gotCompleteInput = input
	return m.completeErr
}

func (m *mockAnalysisRepository) StartRun(_ context.Context, runID string) error {
	if m.run == nil {
		m.run = &model.AnalysisRun{ID: runID}
	}
	return m.startErr
}

func (m *mockAnalysisRepository) UpdateRunProgress(_ context.Context, input model.UpdateAnalysisRunProgressInput) error {
	m.gotProgressInput = input
	m.progressInputs = append(m.progressInputs, input)
	return m.progressErr
}

func (m *mockAnalysisRepository) SaveReviewSnapshots(_ context.Context, runID string, reviews []model.ReviewSnapshot) error {
	m.gotSnapshotsRun = runID
	m.gotSnapshots = reviews
	return m.snapshotErr
}

func (m *mockAnalysisRepository) MarkFailed(_ context.Context, input model.FailAnalysisRunInput) error {
	m.gotFailInput = input
	return m.failErr
}

func (m *mockAnalysisRepository) GetRunDetail(_ context.Context, runID string) (*model.AnalysisRunDetail, error) {
	if m.detail != nil {
		return m.detail, m.detailErr
	}
	return &model.AnalysisRunDetail{
		RunID: runID,
		Overview: &model.Insight{
			Sentiment: model.SentimentBreakdown{},
		},
	}, m.detailErr
}

func (m *mockAnalysisRepository) ListReviewTexts(_ context.Context, runID string) ([]string, error) {
	if len(m.reviewTexts) > 0 {
		return append([]string(nil), m.reviewTexts...), nil
	}

	if len(m.gotSnapshots) > 0 {
		result := make([]string, 0, len(m.gotSnapshots))
		for _, snapshot := range m.gotSnapshots {
			result = append(result, snapshot.ReviewText)
		}
		return result, nil
	}

	return nil, nil
}

func (m *mockAnalysisRepository) ListHistoryByAppID(_ context.Context, appID string, limit int) (*model.GameHistory, error) {
	return m.history, m.historyErr
}

func (m *mockAnalysisRepository) ListEvidence(_ context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error) {
	return m.evidence, m.evidenceErr
}
