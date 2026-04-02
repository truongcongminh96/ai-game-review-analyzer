package ai

func legacyInsightSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"praised_features": map[string]any{
				"type":     "array",
				"maxItems": 5,
				"items": map[string]any{
					"type":      "string",
					"maxLength": 80,
				},
			},
			"common_issues": map[string]any{
				"type":     "array",
				"maxItems": 5,
				"items": map[string]any{
					"type":      "string",
					"maxLength": 80,
				},
			},
			"topics": map[string]any{
				"type":     "array",
				"maxItems": 6,
				"items": map[string]any{
					"type":      "string",
					"maxLength": 48,
				},
			},
			"sentiment": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"positive": map[string]any{"type": "integer"},
					"neutral":  map[string]any{"type": "integer"},
					"negative": map[string]any{"type": "integer"},
				},
				"required":             []string{"positive", "neutral", "negative"},
				"additionalProperties": false,
			},
			"summary": map[string]any{"type": "string", "maxLength": 600},
		},
		"required": []string{
			"praised_features",
			"common_issues",
			"topics",
			"sentiment",
			"summary",
		},
		"additionalProperties": false,
	}
}

func structuredInsightSchema() map[string]any {
	evidenceSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"review_ref": map[string]any{"type": "integer"},
			"quote":      map[string]any{"type": "string", "maxLength": 220},
		},
		"required":             []string{"review_ref", "quote"},
		"additionalProperties": false,
	}

	itemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label":      map[string]any{"type": "string", "maxLength": 64},
			"summary":    map[string]any{"type": "string", "maxLength": 220},
			"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"evidence": map[string]any{
				"type":     "array",
				"maxItems": 1,
				"items":    evidenceSchema,
			},
		},
		"required":             []string{"label", "summary", "confidence", "evidence"},
		"additionalProperties": false,
	}

	issueSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label":      map[string]any{"type": "string", "maxLength": 64},
			"summary":    map[string]any{"type": "string", "maxLength": 220},
			"severity":   map[string]any{"type": "integer", "minimum": 1, "maximum": 5},
			"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"evidence": map[string]any{
				"type":     "array",
				"maxItems": 1,
				"items":    evidenceSchema,
			},
		},
		"required":             []string{"label", "summary", "severity", "confidence", "evidence"},
		"additionalProperties": false,
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{"type": "string", "maxLength": 700},
			"sentiment": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"positive": map[string]any{"type": "integer"},
					"neutral":  map[string]any{"type": "integer"},
					"negative": map[string]any{"type": "integer"},
				},
				"required":             []string{"positive", "neutral", "negative"},
				"additionalProperties": false,
			},
			"praises": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    itemSchema,
			},
			"issues": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    issueSchema,
			},
			"topics": map[string]any{
				"type":     "array",
				"maxItems": 5,
				"items":    itemSchema,
			},
		},
		"required":             []string{"summary", "sentiment", "praises", "issues", "topics"},
		"additionalProperties": false,
	}
}

func standardOllamaOptions(retry bool) map[string]any {
	numPredict := 1024
	if retry {
		numPredict = 1536
	}

	return map[string]any{
		"temperature": 0,
		"num_predict": numPredict,
	}
}

func advancedOllamaOptions(retry bool) map[string]any {
	numPredict := 1536
	if retry {
		numPredict = 3072
	}

	return map[string]any{
		"temperature": 0,
		"num_predict": numPredict,
	}
}
