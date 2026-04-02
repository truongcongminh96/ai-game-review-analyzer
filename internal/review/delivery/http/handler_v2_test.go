package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func TestGetAnalysisRunHandlerIncludesErrorMessage(t *testing.T) {
	mockUseCase := &mockAnalyzeUseCase{
		detail: &model.AnalysisRunDetail{
			RunID:           "run-123",
			Status:          model.AnalysisStatusFailed,
			CurrentStage:    model.AnalysisStageFailed,
			ProgressPercent: 100,
			ErrorMessage:    "failed to call ollama: connection refused",
			Game: model.GameView{
				AppID: "1245620",
				Title: "ELDEN RING",
			},
			Overview: &model.Insight{},
			Praises:  []model.AnalysisItemView{},
			Issues:   []model.AnalysisItemView{},
			Topics:   []model.AnalysisItemView{},
		},
	}
	handler := NewHandler(mockUseCase)

	req := httptest.NewRequest(nethttp.MethodGet, "/v2/analysis-runs/run-123", nil)
	req.SetPathValue("runID", "run-123")
	rec := httptest.NewRecorder()

	handler.GetAnalysisRunHandler(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d", nethttp.StatusOK, rec.Code)
	}

	var body model.AnalysisRunDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.ErrorMessage != "failed to call ollama: connection refused" {
		t.Fatalf("expected error message to round-trip, got %q", body.ErrorMessage)
	}
}
