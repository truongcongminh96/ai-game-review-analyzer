package http

import (
	"context"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type mockAnalyzeUseCase struct {
	result *model.Insight
	err    error

	queued   *model.AnalysisRunQueued
	detail   *model.AnalysisRunDetail
	evidence *model.AnalysisEvidencePage
	history  *model.GameHistory
	compare  *model.CompareAnalysisResult

	gotReviews  []string
	gotAppID    string
	gotLimit    int
	gotLanguage string
}

func (m *mockAnalyzeUseCase) AnalyzeReviews(_ context.Context, reviews []string) (*model.Insight, error) {
	m.gotReviews = reviews
	return m.result, m.err
}

func (m *mockAnalyzeUseCase) AnalyzeSteamReviews(_ context.Context, appID string, limit int, language string) (*model.Insight, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	m.gotLanguage = language
	return m.result, m.err
}

func (m *mockAnalyzeUseCase) RequestSteamAnalysis(_ context.Context, appID string, limit int, language string) (*model.AnalysisRunQueued, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	m.gotLanguage = language
	return m.queued, m.err
}

func (m *mockAnalyzeUseCase) GetAnalysisRun(_ context.Context, runID string) (*model.AnalysisRunDetail, error) {
	return m.detail, m.err
}

func (m *mockAnalyzeUseCase) GetAnalysisEvidence(_ context.Context, input model.AnalysisEvidenceQuery) (*model.AnalysisEvidencePage, error) {
	return m.evidence, m.err
}

func (m *mockAnalyzeUseCase) GetGameHistory(_ context.Context, appID string, limit int) (*model.GameHistory, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	return m.history, m.err
}

func (m *mockAnalyzeUseCase) CompareAnalysisRuns(_ context.Context, runA string, runB string) (*model.CompareAnalysisResult, error) {
	return m.compare, m.err
}

type mockHealthChecker struct {
	enabled bool
	err     error
}

func (m *mockHealthChecker) Enabled() bool {
	return m.enabled
}

func (m *mockHealthChecker) CheckHealth(context.Context) error {
	return m.err
}
