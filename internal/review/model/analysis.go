package model

type AnalysisRun struct {
	ID              string
	GameID          string
	Status          AnalysisStatus
	CurrentStage    AnalysisStage
	ProgressPercent int
}

type AnalysisStatus string

const (
	AnalysisStatusPending AnalysisStatus = "pending"
	AnalysisStatusSuccess AnalysisStatus = "success"
	AnalysisStatusFailed  AnalysisStatus = "failed"
)

type CreateAnalysisRunInput struct {
	GameID      string
	ReviewLimit int
	Language    string
}

type CompleteAnalysisRunInput struct {
	RunID       string
	ReviewCount int
	Insight     *Insight
	Report      *StructuredInsight
	ModelName   string
}

type FailAnalysisRunInput struct {
	RunID        string
	ReviewCount  int
	ErrorMessage string
}
