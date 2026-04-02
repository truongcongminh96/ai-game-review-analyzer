package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

var yearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)

const steamReviewPageSize = 100

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
	if limit <= 0 {
		return nil, nil
	}

	reviews := make([]model.ReviewSteam, 0, min(limit, steamReviewPageSize))
	usePagination := limit > steamReviewPageSize
	cursor := "*"

	for len(reviews) < limit {
		pageSize := min(limit-len(reviews), steamReviewPageSize)
		pageReviews, nextCursor, err := c.getReviewPage(appID, pageSize, language, usePagination, cursor)
		if err != nil {
			return nil, err
		}

		for _, review := range pageReviews {
			if strings.TrimSpace(review.Review) == "" {
				continue
			}
			reviews = append(reviews, review)
			if len(reviews) == limit {
				return reviews, nil
			}
		}

		if !usePagination || len(pageReviews) == 0 || nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	return reviews, nil
}

func (c ClientSteam) getReviewPage(appID string, limit int, language string, usePagination bool, cursor string) ([]model.ReviewSteam, string, error) {
	requestURL, err := buildReviewURL(appID, limit, language, usePagination, cursor)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.httpClient.Get(requestURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to call steam: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("steam returned status %d: %s", resp.StatusCode, string(raw))
	}

	var data struct {
		Reviews []model.ReviewSteam `json:"reviews"`
		Cursor  string              `json:"cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, "", fmt.Errorf("failed to decode steam response: %w", err)
	}

	return data.Reviews, data.Cursor, nil
}

func buildReviewURL(appID string, limit int, language string, usePagination bool, cursor string) (string, error) {
	requestURL, err := url.Parse(fmt.Sprintf("https://store.steampowered.com/appreviews/%s", appID))
	if err != nil {
		return "", fmt.Errorf("build steam review url: %w", err)
	}

	query := requestURL.Query()
	query.Set("json", "1")
	query.Set("num_per_page", fmt.Sprintf("%d", limit))
	query.Set("language", language)
	if usePagination {
		query.Set("filter", "recent")
		if strings.TrimSpace(cursor) == "" {
			cursor = "*"
		}
		query.Set("cursor", cursor)
	}
	requestURL.RawQuery = query.Encode()

	return requestURL.String(), nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
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
