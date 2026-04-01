package http

import nethttp "net/http"

func RegisterRoutes(mux *nethttp.ServeMux, handler *Handler) {
	mux.HandleFunc("/health", handler.HealthHandler)
	mux.HandleFunc("/analyze", handler.AnalyzeHandler)
	mux.HandleFunc("/steam/analyze", handler.AnalyzeSteamHandler)
	mux.HandleFunc("POST /v2/steam/analyze", handler.RequestSteamAnalysisHandler)
	mux.HandleFunc("GET /v2/analysis-runs/{runID}", handler.GetAnalysisRunHandler)
	mux.HandleFunc("GET /v2/analysis-runs/{runID}/evidence", handler.GetAnalysisEvidenceHandler)
	mux.HandleFunc("GET /v2/games/{appID}/history", handler.GetGameHistoryHandler)
	mux.HandleFunc("GET /v2/compare", handler.CompareRunsHandler)
}
