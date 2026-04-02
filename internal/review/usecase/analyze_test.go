package usecase

import (
	"context"
	"errors"
	"strings"
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

func TestAnalyzeReviews_FallsBackToGeneratedSummaryWhenBlank(t *testing.T) {
	mockAI := &mockAIClient{
		result: &model.Insight{},
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
	assert.Equal(t, "AI analysis completed from 2 reviews.", result.Summary)
	assert.Equal(t, 2, result.ReviewCount)
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

func TestAnalyzeSteamReviews_PersistsFallbackSummaryWhenBlank(t *testing.T) {
	mockAI := &mockAIClient{
		model:  "qwen3:8b",
		result: &model.Insight{},
	}
	mockSteam := &mockSteamClient{
		details: &model.SteamGameDetails{
			AppID: "730",
			Title: "Counter-Strike 2",
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
	assert.Equal(t, "AI analysis completed from 2 reviews.", result.Summary)
	require.NotNil(t, mockAnalysisRepo.gotCompleteInput.Insight)
	assert.Equal(t, "AI analysis completed from 2 reviews.", mockAnalysisRepo.gotCompleteInput.Insight.Summary)
}

func TestRunSteamAnalysis_PersistsAdvancedModelName(t *testing.T) {
	mockAI := &mockAIClient{
		advancedModel: "qwen3:14b",
		detailedResult: &model.StructuredInsight{
			Summary: "Players praise the art direction and combat.",
			Sentiment: model.SentimentBreakdown{
				Positive: 1,
			},
			Praises: []model.StructuredInsightItem{
				{
					Label:      "art direction",
					Summary:    "Players praise the art direction.",
					Confidence: 0.92,
					Evidence: []model.EvidenceRef{
						{ReviewRef: 1, Quote: "The art direction is incredible."},
					},
				},
			},
		},
	}
	mockSteam := &mockSteamClient{
		reviews: []model.ReviewSteam{
			newSteamReview("The art direction is incredible.", true),
		},
	}
	mockAnalysisRepo := &mockAnalysisRepository{}

	uc := NewAnalyzeUseCase(mockAI, mockSteam, nil, mockAnalysisRepo)

	uc.runSteamAnalysis(context.Background(), "run-advanced-1", "730", 30, "english")

	assert.Equal(t, "run-advanced-1", mockAnalysisRepo.gotCompleteInput.RunID)
	assert.Equal(t, "qwen3:14b", mockAnalysisRepo.gotCompleteInput.ModelName)
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

func TestSanitizeStructuredInsight_BuildsFallbackEvidenceWhenMissing(t *testing.T) {
	report := &model.StructuredInsight{
		Issues: []model.StructuredInsightItem{
			{
				Label:      "performance",
				Summary:    "Players report stutters and fps drops.",
				Confidence: 0.9,
			},
		},
	}

	reviewTexts := []string{
		"The open world is beautiful and exploration feels great.",
		"I get frequent stutters and fps drops during large fights.",
	}

	sanitized := sanitizeStructuredInsight(report, reviewTexts)
	require.Len(t, sanitized.Issues, 1)
	require.NotEmpty(t, sanitized.Issues[0].Evidence)
	assert.Equal(t, 2, sanitized.Issues[0].Evidence[0].ReviewRef)
	assert.Contains(t, strings.ToLower(sanitized.Issues[0].Evidence[0].Quote), "stutters")
}

func TestSanitizeStructuredInsight_ReplacesInvalidEvidenceQuote(t *testing.T) {
	report := &model.StructuredInsight{
		Topics: []model.StructuredInsightItem{
			{
				Label:      "build variety",
				Summary:    "Many players mention different builds and weapons.",
				Confidence: 0.8,
				Evidence: []model.EvidenceRef{
					{
						ReviewRef: 1,
						Quote:     "this quote does not exist in the review",
					},
				},
			},
		},
	}

	reviewTexts := []string{
		"Tons of viable builds to try and many weapons feel useful.",
	}

	sanitized := sanitizeStructuredInsight(report, reviewTexts)
	require.Len(t, sanitized.Topics, 1)
	require.Len(t, sanitized.Topics[0].Evidence, 1)
	assert.Equal(t, 1, sanitized.Topics[0].Evidence[0].ReviewRef)
	assert.NotEqual(t, "this quote does not exist in the review", sanitized.Topics[0].Evidence[0].Quote)
	assert.Contains(t, strings.ToLower(reviewTexts[0]), strings.ToLower(sanitized.Topics[0].Evidence[0].Quote))
}

func TestSanitizeStructuredInsight_FallsBackToGeneratedSummaryWhenBlank(t *testing.T) {
	report := &model.StructuredInsight{
		Praises: []model.StructuredInsightItem{
			{
				Label:      "combat depth",
				Confidence: 0.92,
			},
			{
				Label:      "world exploration",
				Confidence: 0.88,
			},
		},
	}

	reviewTexts := []string{
		"The combat depth keeps me experimenting.",
		"Exploration is rewarding and packed with secrets.",
	}

	sanitized := sanitizeStructuredInsight(report, reviewTexts)
	assert.Equal(t, "Players frequently praise combat depth and world exploration.", sanitized.Summary)
}
