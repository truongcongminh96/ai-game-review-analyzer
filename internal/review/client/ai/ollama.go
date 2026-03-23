package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/prompt"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func (o OllamaClient) AnalyzeReviews(reviews []string) (*model.Insight, error) {
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	reqBody := ollamaGenerateRequest{
		Model:  o.Model,
		Prompt: prompt.BuildReviewAnalysisPrompt(reviews),
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Post(o.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(resp.Body)

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

	var insight model.Insight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		return nil, fmt.Errorf("failed to parse insight JSON: %w, llm_output=%s", err, cleaned)
	}

	return &insight, nil
}

func (o OllamaClient) ModelName() string {
	return strings.TrimSpace(o.Model)
}

func cleanJSONText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
