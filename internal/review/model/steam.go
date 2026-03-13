package model

type ResponseSteam struct {
	Reviews []ReviewSteam `json:"reviews"`
}

type ReviewSteam struct {
	Review   string `json:"review"`
	VotedUp  bool   `json:"voted_up"`
	Language string `json:"language"`
}
