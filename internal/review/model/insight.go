package model

type SentimentBreakdown struct {
	Positive int `json:"positive"`
	Neutral  int `json:"neutral"`
	Negative int `json:"negative"`
}

type Insight struct {
	PraisedFeatures []string           `json:"praised_features"`
	CommonIssues    []string           `json:"common_issues"`
	Topics          []string           `json:"topics"`
	ReviewCount     int                `json:"review_count"`
	Sentiment       SentimentBreakdown `json:"sentiment"`
	Summary         string             `json:"summary"`
}
