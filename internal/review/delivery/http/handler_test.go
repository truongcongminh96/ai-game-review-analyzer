package http

import (
	"bytes"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func TestHealthHandler(t *testing.T) {
	handler := NewHandler(&mockAnalyzeUseCase{})

	tests := []struct {
		name        string
		method      string
		wantStatus  int
		wantBody    map[string]string
		wantErrBody *errorResponse
	}{
		{
			name:       "returns ok for GET request",
			method:     nethttp.MethodGet,
			wantStatus: nethttp.StatusOK,
			wantBody: map[string]string{
				"status": "ok",
			},
		},
		{
			name:       "returns method not allowed for non GET request",
			method:     nethttp.MethodPost,
			wantStatus: nethttp.StatusMethodNotAllowed,
			wantErrBody: &errorResponse{
				Error: "method not allowed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			handler.HealthHandler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content type application/json, got %q", got)
			}

			if tt.wantBody != nil {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if body["status"] != tt.wantBody["status"] {
					t.Fatalf("expected status body %q, got %q", tt.wantBody["status"], body["status"])
				}
			}

			if tt.wantErrBody != nil {
				var body errorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode error response body: %v", err)
				}

				if body.Error != tt.wantErrBody.Error {
					t.Fatalf("expected error %q, got %q", tt.wantErrBody.Error, body.Error)
				}
			}
		})
	}
}

func TestAnalyzeHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockResult     *model.Insight
		mockErr        error
		wantStatus     int
		wantErr        string
		wantSummary    string
		wantReviewBody []string
	}{
		{
			name:       "returns method not allowed for non POST request",
			method:     nethttp.MethodGet,
			wantStatus: nethttp.StatusMethodNotAllowed,
			wantErr:    "method not allowed",
		},
		{
			name:       "returns bad request for invalid json body",
			method:     nethttp.MethodPost,
			body:       "{invalid json}",
			wantStatus: nethttp.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "returns bad request when use case returns error",
			method:     nethttp.MethodPost,
			body:       `{"reviews":["good game"]}`,
			mockErr:    errors.New("reviews cannot be empty"),
			wantStatus: nethttp.StatusBadRequest,
			wantErr:    "reviews cannot be empty",
		},
		{
			name:   "returns ok when analyze reviews succeeds",
			method: nethttp.MethodPost,
			body:   `{"reviews":["great combat","nice story"]}`,
			mockResult: &model.Insight{
				Summary:     "players like the game",
				ReviewCount: 2,
			},
			wantStatus:     nethttp.StatusOK,
			wantSummary:    "players like the game",
			wantReviewBody: []string{"great combat", "nice story"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := &mockAnalyzeUseCase{
				result: tt.mockResult,
				err:    tt.mockErr,
			}
			handler := NewHandler(mockUseCase)

			req := httptest.NewRequest(tt.method, "/analyze", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler.AnalyzeHandler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content type application/json, got %q", got)
			}

			if tt.wantErr != "" {
				var body errorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode error response body: %v", err)
				}

				if body.Error != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, body.Error)
				}
			}

			if tt.wantSummary != "" {
				var body model.Insight
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode success response body: %v", err)
				}

				if body.Summary != tt.wantSummary {
					t.Fatalf("expected summary %q, got %q", tt.wantSummary, body.Summary)
				}
			}

			if tt.wantReviewBody != nil && !reflect.DeepEqual(mockUseCase.gotReviews, tt.wantReviewBody) {
				t.Fatalf("expected reviews %v, got %v", tt.wantReviewBody, mockUseCase.gotReviews)
			}
		})
	}
}

func TestAnalyzeSteamHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		body         string
		mockResult   *model.Insight
		mockErr      error
		wantStatus   int
		wantErr      string
		wantSummary  string
		wantAppID    string
		wantLimit    int
		wantLanguage string
	}{
		{
			name:       "returns method not allowed for non POST request",
			method:     nethttp.MethodGet,
			wantStatus: nethttp.StatusMethodNotAllowed,
			wantErr:    "method not allowed",
		},
		{
			name:       "returns bad request for invalid json body",
			method:     nethttp.MethodPost,
			body:       "{invalid json}",
			wantStatus: nethttp.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "returns bad request when use case returns error",
			method:     nethttp.MethodPost,
			body:       `{"appId":"730","limit":30,"language":"english"}`,
			mockErr:    errors.New("appId required"),
			wantStatus: nethttp.StatusBadRequest,
			wantErr:    "appId required",
		},
		{
			name:   "returns ok when steam analysis succeeds",
			method: nethttp.MethodPost,
			body:   `{"appId":"730","limit":30,"language":"english"}`,
			mockResult: &model.Insight{
				Summary:     "steam reviews analyzed",
				ReviewCount: 30,
			},
			wantStatus:   nethttp.StatusOK,
			wantSummary:  "steam reviews analyzed",
			wantAppID:    "730",
			wantLimit:    30,
			wantLanguage: "english",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := &mockAnalyzeUseCase{
				result: tt.mockResult,
				err:    tt.mockErr,
			}
			handler := NewHandler(mockUseCase)

			req := httptest.NewRequest(tt.method, "/steam/analyze", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler.AnalyzeSteamHandler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content type application/json, got %q", got)
			}

			if tt.wantErr != "" {
				var body errorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode error response body: %v", err)
				}

				if body.Error != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, body.Error)
				}
			}

			if tt.wantSummary != "" {
				var body model.Insight
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode success response body: %v", err)
				}

				if body.Summary != tt.wantSummary {
					t.Fatalf("expected summary %q, got %q", tt.wantSummary, body.Summary)
				}
			}

			if tt.wantAppID != "" && mockUseCase.gotAppID != tt.wantAppID {
				t.Fatalf("expected appID %q, got %q", tt.wantAppID, mockUseCase.gotAppID)
			}

			if tt.wantLimit != 0 && mockUseCase.gotLimit != tt.wantLimit {
				t.Fatalf("expected limit %d, got %d", tt.wantLimit, mockUseCase.gotLimit)
			}

			if tt.wantLanguage != "" && mockUseCase.gotLanguage != tt.wantLanguage {
				t.Fatalf("expected language %q, got %q", tt.wantLanguage, mockUseCase.gotLanguage)
			}
		})
	}
}
