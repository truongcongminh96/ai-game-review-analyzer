package http

import (
	"encoding/json"
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func (h Handler) RequestSteamAnalysisHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req model.AnalyzeSteamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.useCase.RequestSteamAnalysis(r.Context(), req.AppID, req.Limit, req.Language)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	response := analysisRunQueuedResponse{
		RunID:           result.RunID,
		Status:          result.Status,
		CurrentStage:    result.CurrentStage,
		ProgressPercent: result.ProgressPercent,
	}
	response.Request.AppID = result.Request.AppID
	response.Request.Limit = result.Request.Limit
	response.Request.Language = result.Request.Language
	response.Links.Self = "/v2/analysis-runs/" + result.RunID
	response.Links.History = "/v2/games/" + result.Request.AppID + "/history"

	writeJSON(w, nethttp.StatusAccepted, response)
}

func (h Handler) GetAnalysisRunHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	runID := strings.TrimSpace(r.PathValue("runID"))

	result, err := h.useCase.GetAnalysisRun(r.Context(), runID)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func (h Handler) GetAnalysisEvidenceHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	runID := strings.TrimSpace(r.PathValue("runID"))
	kind := model.InsightKind(strings.TrimSpace(r.URL.Query().Get("kind")))
	label := strings.TrimSpace(r.URL.Query().Get("label"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := parsePositiveIntOrDefault(r.URL.Query().Get("limit"), 20)

	result, err := h.useCase.GetAnalysisEvidence(r.Context(), model.AnalysisEvidenceQuery{
		RunID:  runID,
		Kind:   kind,
		Label:  label,
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func (h Handler) GetGameHistoryHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	limit := parsePositiveIntOrDefault(r.URL.Query().Get("limit"), 10)

	result, err := h.useCase.GetGameHistory(r.Context(), appID, limit)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func (h Handler) CompareRunsHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	runA := strings.TrimSpace(r.URL.Query().Get("runA"))
	runB := strings.TrimSpace(r.URL.Query().Get("runB"))

	result, err := h.useCase.CompareAnalysisRuns(r.Context(), runA, runB)
	if err != nil {
		writeError(w, mapErrorToStatus(err), err.Error())
		return
	}

	writeJSON(w, nethttp.StatusOK, result)
}

func parsePositiveIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
