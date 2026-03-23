package usecase

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type mockAIClient struct {
	result *model.Insight
	err    error
	model  string
}

func (m *mockAIClient) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	return m.result, m.err
}

func (m *mockAIClient) ModelName() string {
	return m.model
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
	createErr   error
	completeErr error
	failErr     error

	gotCreateInput   model.CreateAnalysisRunInput
	gotCompleteInput model.CompleteAnalysisRunInput
	gotFailInput     model.FailAnalysisRunInput
}

func (m *mockAnalysisRepository) CreateRun(_ context.Context, input model.CreateAnalysisRunInput) (*model.AnalysisRun, error) {
	m.gotCreateInput = input
	return m.run, m.createErr
}

func (m *mockAnalysisRepository) CompleteRun(_ context.Context, input model.CompleteAnalysisRunInput) error {
	m.gotCompleteInput = input
	return m.completeErr
}

func (m *mockAnalysisRepository) MarkFailed(_ context.Context, input model.FailAnalysisRunInput) error {
	m.gotFailInput = input
	return m.failErr
}
