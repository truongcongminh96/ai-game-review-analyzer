package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type AnalyzeUsecase interface {
	AnalyzeReviews(reviews []string) (*model.Insight, error)
	AnalyzeSteamReviews(appID string, limit int, language string) (*model.Insight, error)
}

type Handler struct {
	usecase AnalyzeUsecase
}

func NewHandler(usecase AnalyzeUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) HealthHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, nethttp.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) AnalyzeHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.AnalyzeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.usecase.AnalyzeReviews(req.Reviews)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func (h *Handler) AnalyzeSteamHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.AnalyzeSteamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.usecase.AnalyzeSteamReviews(req.AppID, req.Limit, req.Language)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}
