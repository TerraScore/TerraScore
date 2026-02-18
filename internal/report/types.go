package report

import "time"

// GeneratePayload is the task queue payload for report generation.
type GeneratePayload struct {
	JobID    string `json:"job_id"`
	ParcelID string `json:"parcel_id"`
	UserID   string `json:"user_id"`
}

// ReportData holds all data needed to render a report template.
type ReportData struct {
	ParcelLabel    string
	ParcelDistrict string
	ParcelState    string
	SurveyType     string
	JobID          string
	AgentName      string
	SubmittedAt    string
	QAScore        string
	QAStatus       string
	QANotes        string
	Responses      string
	MediaURLs      []MediaURL
	GeneratedAt    string
}

// MediaURL holds a presigned URL for a media item.
type MediaURL struct {
	StepID    string
	MediaType string
	URL       string
}

// ReportResponse is the API representation of a report.
type ReportResponse struct {
	ID          string    `json:"id"`
	ParcelID    string    `json:"parcel_id"`
	JobID       string    `json:"job_id"`
	ReportType  string    `json:"report_type"`
	Format      string    `json:"format"`
	GeneratedAt time.Time `json:"generated_at"`
}

// DownloadResponse is the API response for report download.
type DownloadResponse struct {
	DownloadURL string `json:"download_url"`
}
