package steam

import "github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"

type Client interface {
	GetReviews(appID string, limit int, language string) ([]model.ReviewSteam, error)
	GetGameDetails(appID string) (*model.SteamGameDetails, error)
}
