package ai

import (
	"net/http"
	"time"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

type ollamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Format  any            `json:"format,omitempty"`
	Stream  bool           `json:"stream"`
	System  string         `json:"system,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Response   string `json:"response"`
	DoneReason string `json:"done_reason,omitempty"`
}

type OllamaClient struct {
	BaseURL       string
	Model         string
	StandardModel string
	AdvancedModel string
	Client        *http.Client
}

func NewOllamaClient(cfg config.Config) OllamaClient {
	return OllamaClient{
		BaseURL:       cfg.OllamaBaseURL,
		Model:         cfg.OllamaModel,
		StandardModel: cfg.OllamaModelV1,
		AdvancedModel: cfg.OllamaModelV2,
		Client:        &http.Client{Timeout: time.Duration(cfg.OllamaTimeoutSec) * time.Second},
	}
}
