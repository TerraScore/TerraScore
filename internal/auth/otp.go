package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	otpTTL    = 5 * time.Minute
	otpPrefix = "otp:"
)

// OTPService handles OTP generation and verification.
type OTPService struct {
	redis    *redis.Client
	provider string // "mock" or "msg91"
	authKey  string
	logger   *slog.Logger
}

// NewOTPService creates a new OTP service.
func NewOTPService(rdb *redis.Client, provider, authKey string, logger *slog.Logger) *OTPService {
	return &OTPService{
		redis:    rdb,
		provider: provider,
		authKey:  authKey,
		logger:   logger,
	}
}

// SendOTP generates and sends an OTP to the given phone number.
func (s *OTPService) SendOTP(ctx context.Context, phone string) error {
	otp, err := generateOTP(6)
	if err != nil {
		return fmt.Errorf("generating OTP: %w", err)
	}

	// Store in Redis with TTL
	key := otpPrefix + phone
	if err := s.redis.Set(ctx, key, otp, otpTTL).Err(); err != nil {
		return fmt.Errorf("storing OTP: %w", err)
	}

	if s.provider == "mock" {
		s.logger.Info("MOCK OTP", "phone", phone, "otp", otp)
		return nil
	}

	// MSG91 integration would go here
	// For now, only mock is implemented
	s.logger.Warn("non-mock OTP provider not yet implemented", "provider", s.provider)
	return nil
}

// VerifyOTP checks if the given OTP is valid for the phone number.
func (s *OTPService) VerifyOTP(ctx context.Context, phone, otp string) (bool, error) {
	key := otpPrefix + phone
	stored, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("getting OTP: %w", err)
	}

	if stored != otp {
		return false, nil
	}

	// Delete OTP after successful verification
	s.redis.Del(ctx, key)
	return true, nil
}

// generateOTP generates a cryptographically secure numeric OTP.
func generateOTP(length int) (string, error) {
	otp := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		otp += fmt.Sprintf("%d", n.Int64())
	}
	return otp, nil
}
