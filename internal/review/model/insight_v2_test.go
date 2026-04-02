package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredInsightItemUnmarshalJSON_AcceptsStringEvidence(t *testing.T) {
	raw := []byte(`{
		"label": "Replayability and Content Variety",
		"summary": "Endless build combinations keep players engaged.",
		"confidence": 0.95,
		"evidence": [
			"Review 2: 'Replayability and build possibilities...'",
			"Review 13: 'Play with different loadouts...'"
		]
	}`)

	var item StructuredInsightItem
	err := json.Unmarshal(raw, &item)

	require.NoError(t, err)
	require.Len(t, item.Evidence, 2)
	assert.Equal(t, 2, item.Evidence[0].ReviewRef)
	assert.Equal(t, "Replayability and build possibilities...", item.Evidence[0].Quote)
	assert.Equal(t, 13, item.Evidence[1].ReviewRef)
	assert.Equal(t, "Play with different loadouts...", item.Evidence[1].Quote)
}

func TestStructuredInsightItemUnmarshalJSON_AcceptsObjectEvidence(t *testing.T) {
	raw := []byte(`{
		"label": "Exploration",
		"summary": "Players love discovery.",
		"confidence": 0.91,
		"evidence": [
			{"review_ref": 3, "quote": "every area feels worth exploring"}
		]
	}`)

	var item StructuredInsightItem
	err := json.Unmarshal(raw, &item)

	require.NoError(t, err)
	require.Len(t, item.Evidence, 1)
	assert.Equal(t, 3, item.Evidence[0].ReviewRef)
	assert.Equal(t, "every area feels worth exploring", item.Evidence[0].Quote)
}

func TestStructuredInsightToLegacy_FallsBackSummaryWhenBlank(t *testing.T) {
	report := &StructuredInsight{
		Praises: []StructuredInsightItem{
			{Label: "combat depth"},
			{Label: "world exploration"},
		},
	}

	legacy := report.ToLegacy(2)

	require.NotNil(t, legacy)
	assert.Equal(t, "Players frequently praise combat depth and world exploration.", legacy.Summary)
	assert.Equal(t, []string{"combat depth", "world exploration"}, legacy.PraisedFeatures)
	assert.Equal(t, 2, legacy.ReviewCount)
}
