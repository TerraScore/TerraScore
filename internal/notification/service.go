package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// Service handles notification dispatch across channels.
type Service struct {
	repo    *Repository
	pusher  Pusher
	emailer Emailer
	sms     SMSSender
	logger  *slog.Logger
}

// NewService creates a notification service with the given channel implementations.
func NewService(repo *Repository, pusher Pusher, emailer Emailer, sms SMSSender, logger *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		pusher:  pusher,
		emailer: emailer,
		sms:     sms,
		logger:  logger,
	}
}

// Notify dispatches notifications based on event type.
func (s *Service) Notify(ctx context.Context, eventType string, userID uuid.UUID, title, body string, data map[string]string) error {
	// Always create an in-app alert
	bodyPtr := &body
	dataJSON, _ := json.Marshal(data)
	if _, err := s.repo.CreateAlert(ctx, userID, eventType, title, bodyPtr, dataJSON); err != nil {
		s.logger.Error("failed to create in-app alert", "error", err, "event", eventType)
	}

	// Route to channels based on event type
	switch eventType {
	case "report.generated":
		// Email + push + in-app
		if err := s.emailer.Send(ctx, data["email"], title, body); err != nil {
			s.logger.Error("failed to send email", "error", err)
		}
		if token := data["fcm_token"]; token != "" {
			if err := s.pusher.Send(ctx, token, title, body, data); err != nil {
				s.logger.Error("failed to send push", "error", err)
			}
		}

	case "survey.submitted", "qa.completed", "job.assigned":
		// In-app only (already created above)

	default:
		s.logger.Debug("unhandled notification event", "type", eventType)
	}

	return nil
}

// HandleTask is the TaskHandler for "notification.send".
func (s *Service) HandleTask(ctx context.Context, taskType string, payload json.RawMessage) error {
	var p NotificationPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshalling notification payload: %w", err)
	}

	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	s.logger.Info("sending notification",
		"event_type", p.EventType,
		"user_id", userID,
		"title", p.Title,
	)

	return s.Notify(ctx, p.EventType, userID, p.Title, p.Body, p.Data)
}
