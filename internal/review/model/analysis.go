package model

type AnalysisRun struct {
	ID     string
	GameID string
}

type CreateAnalysisRunInput struct {
	GameID      string
	ReviewLimit int
	Language    string
	Genre       *string
}

type CompleteAnalysisRunInput struct {
	RunID       string
	ReviewCount int
	Insight     *Insight
	ModelName   string
}

type FailAnalysisRunInput struct {
	RunID        string
	ReviewCount  int
	ErrorMessage string
}
