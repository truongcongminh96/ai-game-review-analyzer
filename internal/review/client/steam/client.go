package steam

type Client interface {
	GetReviews(appID string, limit int, language string) ([]string, error)
}
