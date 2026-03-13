package steam

import (
	"net/http"
	"time"
)

type ClientSteam struct {
	httpClient *http.Client
}

func NewClient() ClientSteam {
	return ClientSteam{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}
