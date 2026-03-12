package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/config"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/models"
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

func NewOllamaClient() *OllamaClient {
	cfg := config.Load()

	return &OllamaClient{
		BaseURL: cfg.OllamaBaseURL,
		Model:   cfg.OllamaModel,
		Client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (o *OllamaClient) AnalyzeReviews(reviews []string) (*models.Insight, error) {
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	var reviewLines []string
	for i, review := range reviews {
		reviewLines = append(reviewLines, fmt.Sprintf("%d. %s", i+1, review))
	}

	prompt := fmt.Sprintf(`
You are a game analytics AI.

Analyze the following player reviews.

Return ONLY valid JSON.
Do not add markdown.
Do not wrap in backticks.

Format:
{
  "praised_features": [],
  "common_issues": [],
  "sentiment": {
    "positive": 0,
    "neutral": 0,
    "negative": 0
  },
  "summary": ""
}

Reviews:
%s
`, strings.Join(reviewLines, "\n"))

	reqBody := ollamaGenerateRequest{
		Model:  o.Model,
		Prompt: prompt,
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Post(
		o.BaseURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(raw))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(raw, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w, raw=%s", err, string(raw))
	}

	cleaned := cleanJSONText(ollamaResp.Response)

	var insight models.Insight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		return nil, fmt.Errorf("failed to parse insight JSON: %w, llm_output=%s", err, cleaned)
	}

	return &insight, nil
}

func cleanJSONText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
