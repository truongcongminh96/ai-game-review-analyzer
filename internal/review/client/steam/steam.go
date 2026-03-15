package steam

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func (c ClientSteam) GetReviews(appID string, limit int, language string) ([]model.ReviewSteam, error) {
	url := fmt.Sprintf(
		"https://store.steampowered.com/appreviews/%s?json=1&num_per_page=%d&language=%s",
		appID,
		limit,
		language,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call steam: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("steam returned status %d: %s", resp.StatusCode, string(raw))
	}

	var data model.ResponseSteam
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode steam response: %w", err)
	}

	reviews := make([]model.ReviewSteam, 0, len(data.Reviews))
	for _, r := range data.Reviews {
		if r.Review == "" {
			continue
		}
		reviews = append(reviews, r)
	}

	return reviews, nil
}
