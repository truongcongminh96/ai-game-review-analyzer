package http

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type analysisRunQueuedResponse struct {
	RunID           string                        `json:"run_id"`
	Status          model.AnalysisStatus          `json:"status"`
	CurrentStage    model.AnalysisStage           `json:"current_stage"`
	ProgressPercent int                           `json:"progress_percent"`
	QueueDebug      *model.AnalysisQueueDebugView `json:"queue_debug,omitempty"`
	Request         struct {
		AppID    string `json:"app_id"`
		Limit    int    `json:"limit"`
		Language string `json:"language"`
	} `json:"request"`
	Links struct {
		Self    string `json:"self"`
		History string `json:"history"`
	} `json:"links"`
}
