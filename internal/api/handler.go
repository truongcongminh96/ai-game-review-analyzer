package api

import (
	"encoding/json"
	"net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/analyzer"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/models"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/analyze", AnalyzeHandler)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"status": "ok",
	}

	_ = json.NewEncoder(w).Encode(response)
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req models.AnalyzeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	reviewAnalyzer := analyzer.NewReviewAnalyzer()

	resp := models.AnalyzeReviewResponse{
		Message:         "reviews analyzed successfully",
		ReviewCount:     len(req.Reviews),
		Reviews:         req.Reviews,
		PraisedFeatures: reviewAnalyzer.ExtractPraisedFeatures(req.Reviews),
		CommonIssues:    reviewAnalyzer.ExtractCommonIssues(req.Reviews),
	}

	_ = json.NewEncoder(w).Encode(resp)
}
