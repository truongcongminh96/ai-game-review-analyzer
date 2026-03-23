package model

type Game struct {
	ID          string
	SteamAppID  string
	Title       string
	CoverURL    string
	Genre       string
	ReleaseYear *int
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
