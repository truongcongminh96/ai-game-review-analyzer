package models

type Insight struct {
	PraisedFeatures []string `json:"praised_features"`
	CommonIssues    []string `json:"common_issues"`
	Sentiment       struct {
		Positive int `json:"positive"`
		Neutral  int `json:"neutral"`
		Negative int `json:"negative"`
	} `json:"sentiment"`
	Summary string `json:"summary"`
}
