package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) GetReviews(appID string, limit int, language string) ([]string, error) {
	url := fmt.Sprintf(
		"https://store.steampowered.com/appreviews/%s?json=1&num_per_page=%d&language=%s",
		appID,
		limit,
		language,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	var data ResponseSteam

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	reviews := make([]string, 0)

	for _, r := range data.Reviews {

		if r.Review == "" {
			continue
		}

		reviews = append(reviews, r.Review)
	}

	return reviews, nil
}
