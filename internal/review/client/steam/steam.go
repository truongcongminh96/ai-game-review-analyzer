package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

var yearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)

type steamAppDetailsEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Name        string `json:"name"`
		HeaderImage string `json:"header_image"`
		ReleaseDate struct {
			Date string `json:"date"`
		} `json:"release_date"`
		Genres []struct {
			Description string `json:"description"`
		} `json:"genres"`
	} `json:"data"`
}

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

func (c ClientSteam) GetGameDetails(appID string) (*model.SteamGameDetails, error) {
	url := fmt.Sprintf(
		"https://store.steampowered.com/api/appdetails?appids=%s&l=english",
		appID,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch steam game details: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("steam game details returned status %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]steamAppDetailsEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode steam game details response: %w", err)
	}

	entry, ok := payload[appID]
	if !ok || !entry.Success {
		return nil, fmt.Errorf("steam game details not found for appId %s", appID)
	}

	title := strings.TrimSpace(entry.Data.Name)
	if title == "" {
		return nil, fmt.Errorf("steam game details missing title for appId %s", appID)
	}

	details := &model.SteamGameDetails{
		AppID:       appID,
		Title:       title,
		CoverURL:    strings.TrimSpace(entry.Data.HeaderImage),
		Genre:       joinGenres(entry.Data.Genres),
		ReleaseYear: parseReleaseYear(entry.Data.ReleaseDate.Date),
	}

	return details, nil
}

func joinGenres(genres []struct {
	Description string `json:"description"`
}) string {
	items := make([]string, 0, len(genres))
	for _, genre := range genres {
		trimmed := strings.TrimSpace(genre.Description)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return strings.Join(items, ", ")
}

func parseReleaseYear(date string) *int {
	match := yearPattern.FindString(strings.TrimSpace(date))
	if match == "" {
		return nil
	}

	year := 0
	if _, err := fmt.Sscanf(match, "%d", &year); err != nil || year == 0 {
		return nil
	}

	return &year
}
