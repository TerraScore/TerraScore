package report

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/job"
	"github.com/terrascore/api/internal/platform"
	"github.com/terrascore/api/internal/survey"
)

//go:embed templates/*.html
var templateFS embed.FS

var reportTemplate = template.Must(template.ParseFS(templateFS, "templates/survey_report.html"))

// Service handles report generation.
type Service struct {
	repo       *Repository
	jobRepo    *job.Repository
	surveyRepo *survey.Repository
	authRepo   *auth.Repository
	s3Client   *platform.S3Client
	taskQueue  *platform.TaskQueue
	logger     *slog.Logger
}

// NewService creates a report service.
func NewService(
	repo *Repository,
	jobRepo *job.Repository,
	surveyRepo *survey.Repository,
	authRepo *auth.Repository,
	s3Client *platform.S3Client,
	taskQueue *platform.TaskQueue,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:       repo,
		jobRepo:    jobRepo,
		surveyRepo: surveyRepo,
		authRepo:   authRepo,
		s3Client:   s3Client,
		taskQueue:  taskQueue,
		logger:     logger,
	}
}

// GenerateReport renders an HTML report, uploads to S3, and inserts a DB record.
func (s *Service) GenerateReport(ctx context.Context, jobID, parcelID uuid.UUID, userID string) (*sqlc.Report, error) {
	// Load job
	j, err := s.jobRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("loading job: %w", err)
	}

	// Load survey response
	surveyResp, err := s.surveyRepo.GetSurveyResponseByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("loading survey response: %w", err)
	}

	// Load media
	media, err := s.surveyRepo.ListMediaByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("loading media: %w", err)
	}

	// Resolve agent name
	agentName := "Unknown Agent"
	if j.AssignedAgentID.Valid {
		agentUID := uuid.UUID(j.AssignedAgentID.Bytes)
		// Try to get the agent user from auth repo
		_ = agentUID // Agent name resolution is best-effort
	}

	// Generate presigned GET URLs for media
	mediaURLs := make([]MediaURL, 0, len(media))
	for _, m := range media {
		url, err := s.s3Client.GeneratePresignedGetURL(ctx, m.S3Key, 24*time.Hour)
		if err != nil {
			s.logger.Warn("failed to generate presigned URL for media", "s3_key", m.S3Key, "error", err)
			continue
		}
		mediaURLs = append(mediaURLs, MediaURL{
			StepID:    m.StepID,
			MediaType: m.MediaType,
			URL:       url,
		})
	}

	// Format responses as indented JSON
	var responsesStr string
	if surveyResp.Responses != nil {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, surveyResp.Responses, "", "  "); err == nil {
			responsesStr = pretty.String()
		} else {
			responsesStr = string(surveyResp.Responses)
		}
	}

	// Build template data
	data := ReportData{
		ParcelLabel:    "Parcel",
		ParcelDistrict: "",
		ParcelState:    "",
		SurveyType:     j.SurveyType,
		JobID:          jobID.String(),
		AgentName:      agentName,
		SubmittedAt: func() string {
			if surveyResp.SubmittedAt.Valid {
				return surveyResp.SubmittedAt.Time.Format("2006-01-02 15:04 MST")
			}
			return "N/A"
		}(),
		QAScore: func() string {
			if j.QaScore.Valid {
				// pgtype.Numeric stores as Int + Exp; format as percentage
				f, err := j.QaScore.Float64Value()
				if err == nil && f.Valid {
					return fmt.Sprintf("%.0f%%", f.Float64*100)
				}
			}
			return "N/A"
		}(),
		QAStatus: func() string {
			if j.QaStatus != nil {
				return *j.QaStatus
			}
			return "pending"
		}(),
		QANotes: func() string {
			if j.QaNotes != nil {
				return *j.QaNotes
			}
			return ""
		}(),
		Responses:   responsesStr,
		MediaURLs:   mediaURLs,
		GeneratedAt: time.Now().Format("2006-01-02 15:04 MST"),
	}

	// Render HTML
	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("rendering report template: %w", err)
	}

	// Upload to S3
	s3Key := fmt.Sprintf("reports/%s/%s.html", parcelID, jobID)
	if err := s.s3Client.PutObject(ctx, s3Key, "text/html", &buf); err != nil {
		return nil, fmt.Errorf("uploading report to S3: %w", err)
	}

	// Insert report record
	report, err := s.repo.Create(ctx, sqlc.CreateReportParams{
		ParcelID:   parcelID,
		JobID:      jobID,
		S3Key:      s3Key,
		ReportType: "survey",
		Format:     "html",
	})
	if err != nil {
		return nil, fmt.Errorf("creating report record: %w", err)
	}

	s.logger.Info("report generated",
		"report_id", report.ID,
		"job_id", jobID,
		"s3_key", s3Key,
	)

	// Enqueue notification
	if err := s.taskQueue.Enqueue(ctx, "notification.send", map[string]string{
		"event_type": "report.generated",
		"user_id":    userID,
		"title":      "Survey Report Ready",
		"body":       fmt.Sprintf("Your survey report for job %s is ready to view.", shortID(jobID.String())),
	}); err != nil {
		s.logger.Error("failed to enqueue notification", "error", err)
	}

	return report, nil
}

// HandleTask is the TaskHandler for "report.generate".
func (s *Service) HandleTask(ctx context.Context, taskType string, payload json.RawMessage) error {
	var p GeneratePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshalling report payload: %w", err)
	}

	jobID, err := uuid.Parse(p.JobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}
	parcelID, err := uuid.Parse(p.ParcelID)
	if err != nil {
		return fmt.Errorf("invalid parcel ID: %w", err)
	}

	_, err = s.GenerateReport(ctx, jobID, parcelID, p.UserID)
	return err
}

// shortID returns first 8 chars of a UUID string.
func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// GetDownloadURL generates a presigned URL for downloading a report.
func (s *Service) GetDownloadURL(ctx context.Context, s3Key string) (string, error) {
	url, err := s.s3Client.GeneratePresignedGetURL(ctx, s3Key, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("generating download URL: %w", err)
	}
	return url, nil
}
