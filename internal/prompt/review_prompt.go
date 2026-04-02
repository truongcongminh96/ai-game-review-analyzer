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
- Return JSON only. Do not write headings, bullet points, explanations, or markdown.
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

func BuildReviewAnalysisRetryPrompt(reviews []string) string {
	return BuildReviewAnalysisPrompt(reviews) + `

Important:
- The previous attempt returned invalid or incomplete JSON.
- Retry from scratch.
- Return one complete JSON object only.
- Ensure every string is properly closed and the JSON object is fully closed.
`
}

func BuildReviewAnalysisPromptV2(reviews []string) string {
	var reviewLines []string
	for i, review := range reviews {
		reviewLines = append(reviewLines, fmt.Sprintf("%d. %s", i+1, review))
	}

	return fmt.Sprintf(`You are a game review intelligence AI.

Analyze the player reviews and return ONLY valid JSON.
Do not add markdown.
Do not wrap the JSON in backticks.
Do not include any text before or after the JSON.

Rules:
- Return JSON only. Do not write headings, bullet points, explanations, or markdown.
- Count sentiment across all reviews.
- Extract up to 4 praises, up to 4 issues, and up to 5 topics.
- Each item must have a short label and a concise summary.
- For issues only, include severity from 1 to 5.
- Confidence must be a decimal between 0 and 1.
- Each evidence entry must reference the numbered review via review_ref.
- Never return evidence as strings like "Review 2: ..."; every evidence entry must be a JSON object.
- Each quote must be copied from the review text and be short, usually under 180 characters.
- Each item should include exactly 1 evidence entry whenever the reviews support that item.
- Prefer the single most representative review instead of repeating the same review for every item.
- If you cannot point to at least one supporting review, do not include that item.
- Do not invent evidence.
- Keep the JSON compact and avoid redundant items.
- If a category has no strong signals, return an empty array instead of filler items.
- Topics should be short noun phrases like combat, progression, performance, story, matchmaking, exploration, ui/ux.

Return this exact JSON shape:
{
  "summary": "",
  "sentiment": {
    "positive": 0,
    "neutral": 0,
    "negative": 0
  },
  "praises": [
    {
      "label": "",
      "summary": "",
      "confidence": 0.0,
      "evidence": [
        { "review_ref": 1, "quote": "" }
      ]
    }
  ],
  "issues": [
    {
      "label": "",
      "summary": "",
      "severity": 1,
      "confidence": 0.0,
      "evidence": [
        { "review_ref": 1, "quote": "" }
      ]
    }
  ],
  "topics": [
    {
      "label": "",
      "summary": "",
      "confidence": 0.0,
      "evidence": [
        { "review_ref": 1, "quote": "" }
      ]
    }
  ]
}

Reviews:
%s
`, strings.Join(reviewLines, "\n"))
}

func BuildReviewAnalysisPromptV2Retry(reviews []string) string {
	return BuildReviewAnalysisPromptV2(reviews) + `

Important:
- The previous attempt returned invalid or incomplete JSON.
- Retry from scratch.
- Return one complete JSON object only.
- Ensure every string is properly closed and the JSON object is fully closed.
`
}
