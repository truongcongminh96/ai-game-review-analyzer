package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/models"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/service/analyze"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/steam"
)

type errorResponse struct {
	Error string `json:"error"`
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/analyze", AnalyzeHandler)
	mux.HandleFunc("/steam/analyze", AnalyzeSteamHandler)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.AnalyzeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Reviews = normalizeReviews(req.Reviews)
	if len(req.Reviews) == 0 {
		writeError(w, http.StatusBadRequest, "reviews cannot be empty")
		return
	}

	analyzeService := analyze.NewService()
	result, err := analyzeService.AnalyzeReviews(req.Reviews)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func normalizeReviews(reviews []string) []string {
	normalized := make([]string, 0, len(reviews))

	for _, review := range reviews {
		trimmed := strings.TrimSpace(review)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	return normalized
}

func AnalyzeSteamHandler(w http.ResponseWriter, r *http.Request) {

	var req models.AnalyzeSteamRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.AppID == "" {
		http.Error(w, "appId required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 30
	}

	if req.Language == "" {
		req.Language = "english"
	}

	steamClient := steam.NewClient()

	reviews, err := steamClient.GetReviews(req.AppID, req.Limit, req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	service := analyze.NewService()

	insight, err := service.AnalyzeReviews(reviews)
	insight.ReviewCount = len(reviews)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(insight)
	if err != nil {
		return
	}
}
