package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func (h Handler) HealthHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.healthChecker != nil && h.healthChecker.Enabled() {
		if err := h.healthChecker.CheckHealth(r.Context()); err != nil {
			writeJSON(w, nethttp.StatusServiceUnavailable, map[string]string{
				"status":   "degraded",
				"database": "unreachable",
			})
			return
		}

		writeJSON(w, nethttp.StatusOK, map[string]string{
			"status":   "ok",
			"database": "connected",
		})
		return
	}

	writeJSON(w, nethttp.StatusOK, map[string]string{
		"status":   "ok",
		"database": "disabled",
	})
}

func (h Handler) AnalyzeHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.AnalyzeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.useCase.AnalyzeReviews(r.Context(), req.Reviews)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func (h Handler) AnalyzeSteamHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.AnalyzeSteamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.useCase.AnalyzeSteamReviews(r.Context(), req.AppID, req.Limit, req.Language)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}
