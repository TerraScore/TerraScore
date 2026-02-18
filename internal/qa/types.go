package qa

// Weight constants for QA scoring checks.
const (
	WeightGeo          = 0.25
	WeightCompleteness = 0.25
	WeightBoundaryWalk = 0.20
	WeightTimestamps   = 0.15
	WeightDuplicate    = 0.15
)

// Threshold constants for QA status determination.
const (
	ThresholdAutoPass  = 0.70
	ThresholdFlagged   = 0.50
	ThresholdGeoReject = 0.50
	ThresholdRandomFlag = 0.20
)

// QA status values.
const (
	StatusPassed  = "passed"
	StatusFlagged = "flagged"
	StatusFailed  = "failed"
)

// CheckScore represents a single QA check result.
type CheckScore struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail"`
}

// ScoreResult is the full QA scoring output for a survey.
type ScoreResult struct {
	OverallScore float64      `json:"overall_score"`
	Status       string       `json:"status"`
	Notes        string       `json:"notes"`
	Checks       []CheckScore `json:"checks"`
}

// SurveyQAPayload is the task queue payload for scoring a survey.
type SurveyQAPayload struct {
	JobID    string `json:"job_id"`
	ParcelID string `json:"parcel_id"`
	UserID   string `json:"user_id"`
}
