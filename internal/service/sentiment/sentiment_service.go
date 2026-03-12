package sentiment

import "strings"

type ServiceSentiment struct{}

func NewSentimentService() *ServiceSentiment {
	return &ServiceSentiment{}
}

func (s *ServiceSentiment) Analyze(reviews []string) (positive, neutral, negative int) {

	positiveWords := []string{"amazing", "great", "beautiful", "excellent", "love"}
	negativeWords := []string{"crash", "bug", "terrible", "lag", "bad", "performance"}

	for _, review := range reviews {

		text := strings.ToLower(review)

		posScore := 0
		negScore := 0

		for _, w := range positiveWords {
			if strings.Contains(text, w) {
				posScore++
			}
		}

		for _, w := range negativeWords {
			if strings.Contains(text, w) {
				negScore++
			}
		}

		if posScore > negScore {
			positive++
		} else if negScore > posScore {
			negative++
		} else {
			neutral++
		}
	}

	return
}
