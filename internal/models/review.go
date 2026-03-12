package models

type AnalyzeReviewRequest struct {
	Reviews []string `json:"reviews"`
}

type AnalyzeReviewResponse struct {
	Message         string          `json:"message"`
	ReviewCount     int             `json:"review_count"`
	Reviews         []string        `json:"reviews"`
	PraisedFeatures []string        `json:"praised_features"`
	CommonIssues    []string        `json:"common_issues"`
	Sentiment       SentimentResult `json:"sentiment"`
}
