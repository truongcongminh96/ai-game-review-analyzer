package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAnalyzeReviews_UsesJSONFormat(t *testing.T) {
	var captured ollamaGenerateRequest

	client := OllamaClient{
		BaseURL: "http://ollama.local",
		Model:   "llama3",
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "http://ollama.local/api/generate", req.URL.String())
				require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(bytes.NewBufferString(
						`{"response":"{\"praised_features\":[\"combat\"],\"common_issues\":[\"performance\"],\"topics\":[\"exploration\"],\"sentiment\":{\"positive\":1,\"neutral\":0,\"negative\":0},\"summary\":\"Strong overall reception.\"}"}`,
					)),
				}, nil
			}),
		},
	}

	result, err := client.AnalyzeReviews([]string{"great combat"})

	require.NoError(t, err)
	require.NotNil(t, result)
	format, ok := captured.Format.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", format["type"])
	assert.False(t, captured.Stream)
	assert.Equal(t, "llama3", captured.Model)
	assert.Equal(t, "You are a strict JSON API. Return exactly one complete JSON object that matches the requested schema.", captured.System)
	assert.Equal(t, float64(0), captured.Options["temperature"])
	assert.Equal(t, float64(1024), captured.Options["num_predict"])
	assert.Equal(t, "Strong overall reception.", result.Summary)
}

func TestAnalyzeReviews_RetriesAfterInvalidJSON(t *testing.T) {
	var prompts []string
	callCount := 0

	client := OllamaClient{
		BaseURL: "http://ollama.local",
		Model:   "llama3",
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				callCount++

				var captured ollamaGenerateRequest
				require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
				prompts = append(prompts, captured.Prompt)

				body := `{"response":"{\"praised_features\":[\"combat\"],\"common_issues\":[\"performance\"],\"topics\":[\"exploration\"],\"sentiment\":{\"positive\":1,\"neutral\":0,\"negative\":0},\"summary\":\"Strong overall reception.\"}"}`
				if callCount == 1 {
					body = `{"response":"{\"sort\":\"most_relevant\",\"results\":[{\"title\":\"broken\"}","done_reason":"length"}`
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			}),
		},
	}

	result, err := client.AnalyzeReviews([]string{"great combat"})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, callCount)
	require.Len(t, prompts, 2)
	assert.Contains(t, prompts[1], "previous attempt returned invalid or incomplete JSON")
	assert.Equal(t, "Strong overall reception.", result.Summary)
}

func TestAnalyzeReviewsDetailed_UsesJSONFormat(t *testing.T) {
	var captured ollamaGenerateRequest

	client := OllamaClient{
		BaseURL: "http://ollama.local",
		Model:   "llama3",
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "http://ollama.local/api/generate", req.URL.String())
				require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(bytes.NewBufferString(
						`{"response":"{\"summary\":\"Detailed summary.\",\"sentiment\":{\"positive\":1,\"neutral\":0,\"negative\":0},\"praises\":[{\"label\":\"combat\",\"summary\":\"Players enjoy combat.\",\"confidence\":0.9,\"evidence\":[{\"review_ref\":1,\"quote\":\"great combat\"}]}],\"issues\":[],\"topics\":[]}"}`,
					)),
				}, nil
			}),
		},
	}

	result, err := client.AnalyzeReviewsDetailed([]string{"great combat"})

	require.NoError(t, err)
	require.NotNil(t, result)
	format, ok := captured.Format.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", format["type"])
	properties, ok := format["properties"].(map[string]any)
	require.True(t, ok)
	issuesSchema, ok := properties["issues"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(4), issuesSchema["maxItems"])
	topicsSchema, ok := properties["topics"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(5), topicsSchema["maxItems"])
	assert.False(t, captured.Stream)
	assert.Equal(t, float64(0), captured.Options["temperature"])
	assert.Equal(t, float64(1536), captured.Options["num_predict"])
	assert.Equal(t, "Detailed summary.", result.Summary)
	require.Len(t, result.Praises, 1)
	assert.Equal(t, "combat", result.Praises[0].Label)
}

func TestAnalyzeReviewsDetailed_RetryUsesExpandedPredictBudget(t *testing.T) {
	var numPredictValues []float64
	callCount := 0

	client := OllamaClient{
		BaseURL: "http://ollama.local",
		Model:   "llama3",
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				callCount++

				var captured ollamaGenerateRequest
				require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
				numPredict, ok := captured.Options["num_predict"].(float64)
				require.True(t, ok)
				numPredictValues = append(numPredictValues, numPredict)

				body := `{"response":"{\"summary\":\"Detailed summary.\",\"sentiment\":{\"positive\":1,\"neutral\":0,\"negative\":0},\"praises\":[{\"label\":\"combat\",\"summary\":\"Players enjoy combat.\",\"confidence\":0.9,\"evidence\":[{\"review_ref\":1,\"quote\":\"great combat\"}]}],\"issues\":[],\"topics\":[]}"}`
				if callCount == 1 {
					body = `{"response":"{\"issues\":[{\"label\":\"broken\"}","done_reason":"length"}`
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			}),
		},
	}

	result, err := client.AnalyzeReviewsDetailed([]string{"great combat"})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []float64{1536, 3072}, numPredictValues)
}
