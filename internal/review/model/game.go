package model

type Game struct {
	ID          string `json:"id"`
	SteamAppID  string `json:"steam_app_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url"`
	Genre       string `json:"genre"`
	ReleaseYear *int   `json:"release_year,omitempty"`
}

type GameUpsertInput struct {
	SteamAppID          string
	Title               string
	CoverURL            *string
	Genre               *string
	ReleaseYear         *int
	PreferExistingTitle bool
}

type SteamGameDetails struct {
	AppID       string
	Title       string
	CoverURL    string
	Genre       string
	ReleaseYear *int
}

type GameView struct {
	AppID       string `json:"app_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url,omitempty"`
	Genre       string `json:"genre,omitempty"`
	ReleaseYear *int   `json:"release_year,omitempty"`
}
