package steam

import (
	"bytes"
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

func TestGetReviews_PaginatesWhenLimitExceedsSingleSteamPage(t *testing.T) {
	callCount := 0

	client := ClientSteam{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				callCount++
				query := req.URL.Query()
				assert.Equal(t, "1", query.Get("json"))
				assert.Equal(t, "english", query.Get("language"))
				assert.Equal(t, "100", query.Get("num_per_page"))
				assert.Equal(t, "recent", query.Get("filter"))

				body := `{"reviews":[{"recommendationid":"rev-1","review":"First paged review","voted_up":true,"language":"english"},{"recommendationid":"rev-2","review":"Second paged review","voted_up":false,"language":"english"}],"cursor":"cursor-2"}`
				if callCount == 1 {
					assert.Equal(t, "*", query.Get("cursor"))
				} else {
					assert.Equal(t, "cursor-2", query.Get("cursor"))
					body = `{"reviews":[{"recommendationid":"rev-3","review":"Third paged review","voted_up":true,"language":"english"}],"cursor":""}`
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			}),
		},
	}

	reviews, err := client.GetReviews("730", 150, "english")

	require.NoError(t, err)
	require.Len(t, reviews, 3)
	assert.Equal(t, 2, callCount)
	assert.Equal(t, "rev-1", reviews[0].RecommendationID)
	assert.Equal(t, "rev-3", reviews[2].RecommendationID)
}

func TestGetReviews_DoesNotUseCursorForSinglePageRequests(t *testing.T) {
	client := ClientSteam{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				assert.Equal(t, "30", query.Get("num_per_page"))
				assert.Empty(t, query.Get("cursor"))
				assert.Empty(t, query.Get("filter"))

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBufferString(`{"reviews":[{"recommendationid":"rev-1","review":"Single page review","voted_up":true,"language":"english"}]}`)),
				}, nil
			}),
		},
	}

	reviews, err := client.GetReviews("730", 30, "english")

	require.NoError(t, err)
	require.Len(t, reviews, 1)
	assert.Equal(t, "rev-1", reviews[0].RecommendationID)
}
