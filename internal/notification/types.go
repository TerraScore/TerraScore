package notification

import "context"

// Pusher sends push notifications (FCM).
type Pusher interface {
	Send(ctx context.Context, token, title, body string, data map[string]string) error
}

// Emailer sends email notifications.
type Emailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// SMSSender sends SMS notifications.
type SMSSender interface {
	Send(ctx context.Context, phone, message string) error
}

// NotificationPayload is the task queue payload for sending notifications.
type NotificationPayload struct {
	EventType string            `json:"event_type"`
	UserID    string            `json:"user_id"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data,omitempty"`
}

// AlertResponse is the API representation of an alert.
type AlertResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Body      *string `json:"body,omitempty"`
	IsRead    bool    `json:"is_read"`
	CreatedAt string  `json:"created_at"`
}
