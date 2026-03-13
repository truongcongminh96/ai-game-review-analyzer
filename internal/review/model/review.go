package model

type AnalyzeReviewRequest struct {
	Reviews []string `json:"reviews"`
}

type AnalyzeSteamRequest struct {
	AppID    string `json:"appId"`
	Limit    int    `json:"limit"`
	Language string `json:"language"`
}
