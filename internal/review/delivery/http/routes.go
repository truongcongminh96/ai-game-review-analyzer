package http

import nethttp "net/http"

func RegisterRoutes(mux *nethttp.ServeMux, handler *Handler) {
	mux.HandleFunc("/health", handler.HealthHandler)
	mux.HandleFunc("/analyze", handler.AnalyzeHandler)
	mux.HandleFunc("/steam/analyze", handler.AnalyzeSteamHandler)
}
