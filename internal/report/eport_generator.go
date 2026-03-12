package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type GeneratorReport struct {
	apiKey     string
	httpClient *http.Client
}

func NewReportGenerator() *GeneratorReport {
	return &GeneratorReport{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GeneratorReport) GenerateSummary(ctx context.Context, reviews []string) (string, error) {
	if g.apiKey == "" {
		return "", errors.New("missing OPENAI_API_KEY")
	}

	prompt := buildPrompt(reviews)

	reqBody := map[string]any{
		"model": "gpt-4.1-mini",
		"input": prompt,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request to OpenAI: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("openai api error: status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal OpenAI response: %w", err)
	}

	if apiResp.OutputText != "" {
		return apiResp.OutputText, nil
	}

	for _, item := range apiResp.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				return content.Text, nil
			}
		}
	}

	return "", errors.New("no summary text returned from OpenAI")
}

func buildPrompt(reviews []string) string {
	type payload struct {
		Reviews []string `json:"reviews"`
	}

	data, _ := json.MarshalIndent(payload{Reviews: reviews}, "", "  ")

	return `You are an AI game review analyst.

Your task:
- Read the player reviews
- Write a concise summary in 2 sentences maximum
- Mention praised features
- Mention major complaints
- Keep the tone professional and product-oriented

Player reviews:
` + string(data)
}

type ResponsesAPIResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}
