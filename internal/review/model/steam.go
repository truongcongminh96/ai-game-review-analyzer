package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ResponseSteam struct {
	Reviews []ReviewSteam `json:"reviews"`
}

type SteamFlexibleFloat64 float64

func (f *SteamFlexibleFloat64) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*f = 0
		return nil
	}

	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		*f = SteamFlexibleFloat64(number)
		return nil
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode flexible float64: %w", err)
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		*f = 0
		return nil
	}

	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("parse flexible float64 %q: %w", raw, err)
	}

	*f = SteamFlexibleFloat64(parsed)
	return nil
}

type ReviewSteam struct {
	RecommendationID  string               `json:"recommendationid"`
	Review            string               `json:"review"`
	VotedUp           bool                 `json:"voted_up"`
	Language          string               `json:"language"`
	TimestampCreated  int64                `json:"timestamp_created"`
	VotesUp           int                  `json:"votes_up"`
	VotesFunny        int                  `json:"votes_funny"`
	WeightedVoteScore SteamFlexibleFloat64 `json:"weighted_vote_score"`
	Author            struct {
		PlaytimeForever int `json:"playtime_forever"`
	} `json:"author"`
}
