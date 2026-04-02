package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
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
			Debug: &model.AnalysisDebugView{
				BatchCount:     3,
				BatchSizeLimit: 50,
				BatchCharLimit: 14000,
				BatchSizes:     []int{50, 50, 21},
			},
			Praises: []model.AnalysisItemView{},
			Issues:  []model.AnalysisItemView{},
			Topics:  []model.AnalysisItemView{},
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
	if body.Debug == nil || body.Debug.BatchCount != 3 {
		t.Fatalf("expected debug batch_count to round-trip, got %+v", body.Debug)
	}
}

func TestRequestSteamAnalysisHandlerIncludesQueueDebug(t *testing.T) {
	mockUseCase := &mockAnalyzeUseCase{
		queued: &model.AnalysisRunQueued{
			RunID:           "run-queued-1",
			Status:          model.AnalysisStatusPending,
			CurrentStage:    model.AnalysisStageQueued,
			ProgressPercent: 0,
			QueueDebug: &model.AnalysisQueueDebugView{
				EstimatedBatchCount:       3,
				EstimatedReviewFetchPages: 2,
				BatchSizeLimit:            50,
				BatchCharLimit:            14000,
			},
		},
	}
	mockUseCase.queued.Request.AppID = "730"
	mockUseCase.queued.Request.Limit = 121
	mockUseCase.queued.Request.Language = "english"

	handler := NewHandler(mockUseCase)

	req := httptest.NewRequest(nethttp.MethodPost, "/v2/steam/analyze", strings.NewReader(`{"appId":"730","limit":121,"language":"english"}`))
	rec := httptest.NewRecorder()

	handler.RequestSteamAnalysisHandler(rec, req)

	if rec.Code != nethttp.StatusAccepted {
		t.Fatalf("expected status %d, got %d", nethttp.StatusAccepted, rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	queueDebug, ok := body["queue_debug"].(map[string]any)
	if !ok {
		t.Fatalf("expected queue_debug object, got %#v", body["queue_debug"])
	}
	if queueDebug["estimated_batch_count"] != float64(3) {
		t.Fatalf("expected estimated_batch_count 3, got %#v", queueDebug["estimated_batch_count"])
	}
	if queueDebug["estimated_review_fetch_pages"] != float64(2) {
		t.Fatalf("expected estimated_review_fetch_pages 2, got %#v", queueDebug["estimated_review_fetch_pages"])
	}
}
