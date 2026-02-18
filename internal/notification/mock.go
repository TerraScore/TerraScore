package notification

import (
	"context"
	"log/slog"
)

// MockPusher logs push notifications instead of sending them.
type MockPusher struct {
	logger *slog.Logger
}

func NewMockPusher(logger *slog.Logger) *MockPusher {
	return &MockPusher{logger: logger}
}

func (m *MockPusher) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	m.logger.Info("[mock] push notification",
		"token", token,
		"title", title,
		"body", body,
		"data", data,
	)
	return nil
}

// MockEmailer logs emails instead of sending them.
type MockEmailer struct {
	logger *slog.Logger
}

func NewMockEmailer(logger *slog.Logger) *MockEmailer {
	return &MockEmailer{logger: logger}
}

func (m *MockEmailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	m.logger.Info("[mock] email sent",
		"to", to,
		"subject", subject,
		"body_length", len(htmlBody),
	)
	return nil
}

// MockSMSSender logs SMS instead of sending them.
type MockSMSSender struct {
	logger *slog.Logger
}

func NewMockSMSSender(logger *slog.Logger) *MockSMSSender {
	return &MockSMSSender{logger: logger}
}

func (m *MockSMSSender) Send(ctx context.Context, phone, message string) error {
	m.logger.Info("[mock] SMS sent",
		"phone", phone,
		"message", message,
	)
	return nil
}
