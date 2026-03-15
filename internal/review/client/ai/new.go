package ai

import (
	"net/http"
	"time"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

type OllamaClient struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOllamaClient(cfg config.Config) OllamaClient {
	return OllamaClient{
		BaseURL: cfg.OllamaBaseURL,
		Model:   cfg.OllamaModel,
		Client:  &http.Client{Timeout: 120 * time.Second},
	}
}
