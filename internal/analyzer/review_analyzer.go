package analyzer

import "strings"

type ReviewAnalyzer struct{}

func NewReviewAnalyzer() *ReviewAnalyzer {
	return &ReviewAnalyzer{}
}

func (a *ReviewAnalyzer) ExtractPraisedFeatures(reviews []string) []string {
	var features []string

	for _, review := range reviews {
		text := strings.ToLower(review)

		if strings.Contains(text, "combat") {
			features = append(features, "combat")
		}
		if strings.Contains(text, "world") || strings.Contains(text, "open world") {
			features = append(features, "world design")
		}
		if strings.Contains(text, "story") {
			features = append(features, "story")
		}
		if strings.Contains(text, "visual") || strings.Contains(text, "graphics") || strings.Contains(text, "beautiful") {
			features = append(features, "visual design")
		}
	}

	return unique(features)
}

func (a *ReviewAnalyzer) ExtractCommonIssues(reviews []string) []string {
	var issues []string

	for _, review := range reviews {
		text := strings.ToLower(review)

		if strings.Contains(text, "crash") || strings.Contains(text, "crashes") {
			issues = append(issues, "crashes")
		}
		if strings.Contains(text, "performance") || strings.Contains(text, "lag") || strings.Contains(text, "fps") {
			issues = append(issues, "performance issues")
		}
		if strings.Contains(text, "bug") || strings.Contains(text, "bugs") {
			issues = append(issues, "bugs")
		}
		if strings.Contains(text, "balance") || strings.Contains(text, "unbalanced") {
			issues = append(issues, "balance issues")
		}
	}

	return unique(issues)
}

func unique(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
