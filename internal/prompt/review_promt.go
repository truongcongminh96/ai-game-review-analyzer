package prompt

import (
	"fmt"
	"strings"
)

func BuildReviewAnalysisPrompt(reviews []string) string {
	var reviewLines []string
	for i, review := range reviews {
		reviewLines = append(reviewLines, fmt.Sprintf("%d. %s", i+1, review))
	}

	return fmt.Sprintf(`You are a game analytics AI.

Analyze the following player reviews and return ONLY valid JSON.
Do not add markdown.
Do not wrap the JSON in backticks.

Rules:
- Count sentiment across all reviews.
- Extract up to 5 praised features.
- Extract up to 5 common issues.
- Extract up to 6 key topics about gameplay systems or player experience.
- Topics should be short noun phrases like: combat, progression, performance, story, quest design, balance, UI/UX, monetization, matchmaking, exploration.
- Write a professional 2-3 sentence summary.

Return this exact JSON shape:
{
  "praised_features": [],
  "common_issues": [],
  "topics": [],
  "sentiment": {
    "positive": 0,
    "neutral": 0,
    "negative": 0
  },
  "summary": ""
}

Reviews:
%s
`, strings.Join(reviewLines, "\n"))
}
