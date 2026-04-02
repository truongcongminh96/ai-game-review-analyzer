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

	reqBody := o.newJSONGenerateRequest(
		o.StandardModelName(),
		prompt.BuildReviewAnalysisPrompt(reviews),
		legacyInsightSchema(),
		standardOllamaOptions(false),
	)

	cleaned, err := o.generateJSON(reqBody)
	if err != nil {
		return nil, err
	}

	insight, parseErr := parseLegacyInsight(cleaned)
	if parseErr == nil {
		return insight, nil
	}

	retryBody := o.newJSONGenerateRequest(
		o.StandardModelName(),
		prompt.BuildReviewAnalysisRetryPrompt(reviews),
		legacyInsightSchema(),
		standardOllamaOptions(true),
	)
	retried, retryErr := o.generateJSON(retryBody)
	if retryErr != nil {
		return nil, parseErr
	}

	insight, parseErr = parseLegacyInsight(retried)
	if parseErr != nil {
		return nil, parseErr
	}

	return insight, nil
}

func (o OllamaClient) AnalyzeReviewsDetailed(reviews []string) (*model.StructuredInsight, error) {
	if len(reviews) == 0 {
		return nil, fmt.Errorf("reviews cannot be empty")
	}

	reqBody := o.newJSONGenerateRequest(
		o.AdvancedModelName(),
		prompt.BuildReviewAnalysisPromptV2(reviews),
		structuredInsightSchema(),
		advancedOllamaOptions(false),
	)

	cleaned, err := o.generateJSON(reqBody)
	if err != nil {
		return nil, err
	}

	insight, parseErr := parseStructuredInsight(cleaned)
	if parseErr == nil {
		return insight, nil
	}

	retryBody := o.newJSONGenerateRequest(
		o.AdvancedModelName(),
		prompt.BuildReviewAnalysisPromptV2Retry(reviews),
		structuredInsightSchema(),
		advancedOllamaOptions(true),
	)
	retried, retryErr := o.generateJSON(retryBody)
	if retryErr != nil {
		return nil, parseErr
	}

	insight, parseErr = parseStructuredInsight(retried)
	if parseErr != nil {
		return nil, parseErr
	}

	return insight, nil
}

func (o OllamaClient) generateJSON(reqBody ollamaGenerateRequest) (string, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := o.Client.Post(o.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to call ollama: %w", err)
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(resp.Body)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(raw))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(raw, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w, raw=%s", err, string(raw))
	}

	return cleanJSONText(ollamaResp.Response), nil
}

func parseLegacyInsight(cleaned string) (*model.Insight, error) {
	if err := validateExpectedKeys(cleaned, []string{
		"praised_features",
		"common_issues",
		"topics",
		"sentiment",
		"summary",
	}); err != nil {
		return nil, fmt.Errorf("failed to parse insight JSON: %w, llm_output=%s", err, cleaned)
	}

	var insight model.Insight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		return nil, fmt.Errorf("failed to parse insight JSON: %w, llm_output=%s", err, cleaned)
	}

	return &insight, nil
}

func parseStructuredInsight(cleaned string) (*model.StructuredInsight, error) {
	if err := validateExpectedKeys(cleaned, []string{
		"summary",
		"sentiment",
		"praises",
		"issues",
		"topics",
	}); err != nil {
		return nil, fmt.Errorf("failed to parse structured insight JSON: %w, llm_output=%s", err, cleaned)
	}

	var insight model.StructuredInsight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		return nil, fmt.Errorf("failed to parse structured insight JSON: %w, llm_output=%s", err, cleaned)
	}

	insight.RawAIResponse = json.RawMessage([]byte(cleaned))

	return &insight, nil
}

func (o OllamaClient) newJSONGenerateRequest(modelName string, promptText string, schema any, options map[string]any) ollamaGenerateRequest {
	return ollamaGenerateRequest{
		Model:   strings.TrimSpace(modelName),
		Prompt:  promptText,
		Format:  schema,
		Stream:  false,
		System:  "You are a strict JSON API. Return exactly one complete JSON object that matches the requested schema.",
		Options: options,
	}
}

func (o OllamaClient) StandardModelName() string {
	if modelName := strings.TrimSpace(o.StandardModel); modelName != "" {
		return modelName
	}
	if modelName := strings.TrimSpace(o.Model); modelName != "" {
		return modelName
	}
	return strings.TrimSpace(o.AdvancedModel)
}

func (o OllamaClient) AdvancedModelName() string {
	if modelName := strings.TrimSpace(o.AdvancedModel); modelName != "" {
		return modelName
	}
	if modelName := strings.TrimSpace(o.Model); modelName != "" {
		return modelName
	}
	return strings.TrimSpace(o.StandardModel)
}

func validateExpectedKeys(cleaned string, requiredKeys []string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &envelope); err != nil {
		return err
	}

	for _, key := range requiredKeys {
		if _, ok := envelope[key]; !ok {
			return fmt.Errorf("missing required key %q", key)
		}
	}

	return nil
}

func (o OllamaClient) ModelName() string { return o.StandardModelName() }

func cleanJSONText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
