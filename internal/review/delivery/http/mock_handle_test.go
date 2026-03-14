package http

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type mockAnalyzeUseCase struct {
	result *model.Insight
	err    error

	gotReviews  []string
	gotAppID    string
	gotLimit    int
	gotLanguage string
}

func (m *mockAnalyzeUseCase) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	m.gotReviews = reviews
	return m.result, m.err
}

func (m *mockAnalyzeUseCase) AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	m.gotLanguage = language
	return m.result, m.err
}
