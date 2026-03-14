package usecase

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type mockAIClient struct {
	result *model.Insight
	err    error
}

func (m *mockAIClient) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	return m.result, m.err
}

type mockSteamClient struct {
	reviews []string
	err     error

	gotAppID    string
	gotLimit    int
	gotLanguage string
}

func (m *mockSteamClient) GetReviews(appID string, limit int, language string) ([]string, error) {
	m.gotAppID = appID
	m.gotLimit = limit
	m.gotLanguage = language
	return m.reviews, m.err
}
