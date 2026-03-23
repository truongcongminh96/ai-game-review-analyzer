package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func newSteamReview(review string, votedUp bool) model.ReviewSteam {
	return model.ReviewSteam{
		Review:   review,
		VotedUp:  votedUp,
		Language: "english",
	}
}

/*
Test: AnalyzeReviews success
*/
func TestAnalyzeReviews_Success(t *testing.T) {

	mockAI := &mockAIClient{
		result: &model.Insight{
			Summary: "great game",
		},
	}

	mockSteam := &mockSteamClient{}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

	reviews := []string{
		"Great combat system",
		"Beautiful open world",
	}

	result, err := uc.AnalyzeReviews(context.Background(), reviews)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.ReviewCount)
	assert.Equal(t, "great game", result.Summary)
}

/*
Test: empty reviews
*/
func TestAnalyzeReviews_EmptyReviews(t *testing.T) {

	mockAI := &mockAIClient{}
	mockSteam := &mockSteamClient{}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

	_, err := uc.AnalyzeReviews(context.Background(), []string{})

	require.Error(t, err)
}

/*
Test: AI client error
*/
func TestAnalyzeReviews_AIError(t *testing.T) {

	mockAI := &mockAIClient{
		err: errors.New("ai failure"),
	}

	mockSteam := &mockSteamClient{}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

	_, err := uc.AnalyzeReviews(context.Background(), []string{"good game"})

	require.Error(t, err)
	assert.Equal(t, "ai failure", err.Error())
}

/*
Test: AnalyzeSteamReviews success
*/
func TestAnalyzeSteamReviews_DefaultsAndSuccess(t *testing.T) {
	tests := []struct {
		name             string
		appID            string
		limit            int
		language         string
		mockSteamReviews []model.ReviewSteam
		mockSteamErr     error
		mockAIResult     *model.Insight
		mockAIErr        error

		wantErr         bool
		wantErrMsg      string
		wantReviewCount int
	}{
		{
			name:     "success with explicit limit and language",
			appID:    "12345",
			limit:    10,
			language: "english",
			mockSteamReviews: []model.ReviewSteam{
				newSteamReview("great gameplay", true),
				newSteamReview("nice graphics", true),
			},
			mockAIResult: &model.Insight{
				Summary: "good reviews",
			},
			wantErr:         false,
			wantReviewCount: 2,
		},
		{
			name:     "default limit when limit is zero",
			appID:    "12345",
			limit:    0,
			language: "english",
			mockSteamReviews: []model.ReviewSteam{
				newSteamReview("great gameplay", true),
			},
			mockAIResult: &model.Insight{
				Summary: "good reviews",
			},
			wantErr:         false,
			wantReviewCount: 1,
		},
		{
			name:     "default language when language is empty",
			appID:    "12345",
			limit:    10,
			language: "",
			mockSteamReviews: []model.ReviewSteam{
				newSteamReview("great gameplay", true),
			},
			mockAIResult: &model.Insight{
				Summary: "good reviews",
			},
			wantErr:         false,
			wantReviewCount: 1,
		},
		{
			name:     "default both limit and language",
			appID:    "12345",
			limit:    -1,
			language: "",
			mockSteamReviews: []model.ReviewSteam{
				newSteamReview("great gameplay", true),
				newSteamReview("nice graphics", true),
			},
			mockAIResult: &model.Insight{
				Summary: "good reviews",
			},
			wantErr:         false,
			wantReviewCount: 2,
		},
		{
			name:       "missing app id",
			appID:      "",
			limit:      10,
			language:   "english",
			wantErr:    true,
			wantErrMsg: "appId required",
		},
		{
			name:         "steam client error",
			appID:        "12345",
			limit:        10,
			language:     "english",
			mockSteamErr: errors.New("steam api error"),
			wantErr:      true,
			wantErrMsg:   "steam api error",
		},
		{
			name:     "ai client error",
			appID:    "12345",
			limit:    10,
			language: "english",
			mockSteamReviews: []model.ReviewSteam{
				newSteamReview("great gameplay", true),
			},
			mockAIErr:  errors.New("ai failure"),
			wantErr:    true,
			wantErrMsg: "ai failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAI := &mockAIClient{
				result: tt.mockAIResult,
				err:    tt.mockAIErr,
			}

			mockSteam := &mockSteamClient{
				reviews: tt.mockSteamReviews,
				err:     tt.mockSteamErr,
			}

			uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

			result, err := uc.AnalyzeSteamReviews(context.Background(), tt.appID, tt.limit, tt.language)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantReviewCount, result.ReviewCount)
		})
	}
}

/*
Test: Steam client error
*/
func TestAnalyzeSteamReviews_SteamError(t *testing.T) {

	mockAI := &mockAIClient{}

	mockSteam := &mockSteamClient{
		err: errors.New("steam api error"),
	}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

	_, err := uc.AnalyzeSteamReviews(context.Background(), "12345", 10, "english")

	require.Error(t, err)
	assert.Equal(t, "steam api error", err.Error())
}

/*
Test: missing appId
*/
func TestAnalyzeSteamReviews_MissingAppID(t *testing.T) {

	mockAI := &mockAIClient{}
	mockSteam := &mockSteamClient{}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, nil)

	_, err := uc.AnalyzeSteamReviews(context.Background(), "", 10, "english")

	require.Error(t, err)
}

func TestAnalyzeSteamReviews_PersistsSuccess(t *testing.T) {
	mockAI := &mockAIClient{
		model: "qwen3:8b",
		result: &model.Insight{
			Summary:         "great overall",
			PraisedFeatures: []string{"combat"},
			CommonIssues:    []string{"matchmaking"},
			Topics:          []string{"multiplayer"},
		},
	}
	mockSteam := &mockSteamClient{
		details: &model.SteamGameDetails{
			AppID:    "730",
			Title:    "Counter-Strike 2",
			CoverURL: "https://cdn.example/cs2.jpg",
			Genre:    "Action",
		},
		reviews: []model.ReviewSteam{
			newSteamReview("great gameplay", true),
			newSteamReview("servers are rough", false),
		},
	}
	mockGameRepo := &mockGameRepository{
		game: &model.Game{
			ID:         "game-1",
			SteamAppID: "730",
			Title:      "Counter-Strike 2",
			Genre:      "Action",
		},
	}
	mockAnalysisRepo := &mockAnalysisRepository{
		run: &model.AnalysisRun{
			ID:     "run-1",
			GameID: "game-1",
		},
	}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, mockGameRepo, mockAnalysisRepo)

	result, err := uc.AnalyzeSteamReviews(context.Background(), "730", 30, "english")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "730", mockGameRepo.gotInput.SteamAppID)
	assert.Equal(t, "Counter-Strike 2", mockGameRepo.gotInput.Title)
	assert.Equal(t, "game-1", mockAnalysisRepo.gotCreateInput.GameID)
	assert.Equal(t, 30, mockAnalysisRepo.gotCreateInput.ReviewLimit)
	assert.Equal(t, "english", mockAnalysisRepo.gotCreateInput.Language)
	assert.Equal(t, "run-1", mockAnalysisRepo.gotCompleteInput.RunID)
	assert.Equal(t, "qwen3:8b", mockAnalysisRepo.gotCompleteInput.ModelName)
	assert.Equal(t, 2, mockAnalysisRepo.gotCompleteInput.ReviewCount)
	assert.Equal(t, 0, result.Sentiment.Neutral)
	assert.Equal(t, 1, result.Sentiment.Positive)
	assert.Equal(t, 1, result.Sentiment.Negative)
}

func TestAnalyzeSteamReviews_MarksRunFailedWhenAIAnalysisFails(t *testing.T) {
	mockAI := &mockAIClient{
		err: errors.New("ai failure"),
	}
	mockSteam := &mockSteamClient{
		reviews: []model.ReviewSteam{
			newSteamReview("great gameplay", true),
		},
	}
	mockGameRepo := &mockGameRepository{
		game: &model.Game{
			ID:         "game-1",
			SteamAppID: "730",
			Title:      "Steam App 730",
		},
	}
	mockAnalysisRepo := &mockAnalysisRepository{
		run: &model.AnalysisRun{
			ID:     "run-1",
			GameID: "game-1",
		},
	}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, mockGameRepo, mockAnalysisRepo)

	result, err := uc.AnalyzeSteamReviews(context.Background(), "730", 30, "english")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "run-1", mockAnalysisRepo.gotFailInput.RunID)
	assert.Equal(t, 1, mockAnalysisRepo.gotFailInput.ReviewCount)
	assert.Equal(t, "ai failure", mockAnalysisRepo.gotFailInput.ErrorMessage)
}
