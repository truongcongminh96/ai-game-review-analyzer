package models

type AnalyzeSteamRequest struct {
	AppID    string `json:"appId"`
	Limit    int    `json:"limit"`
	Language string `json:"language"`
}
