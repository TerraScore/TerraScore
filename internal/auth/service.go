package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// NormalizePhone ensures phone is in +91XXXXXXXXXX format.
// Accepts: "9876543210", "91XXXXXXXXXX", "+91XXXXXXXXXX", "+91 98765 43210".
func NormalizePhone(phone string) string {
	// Strip spaces, dashes, parens
	phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(phone)
	if strings.HasPrefix(phone, "+91") {
		return phone
	}
	if strings.HasPrefix(phone, "91") && len(phone) == 12 {
		return "+" + phone
	}
	// 10-digit number
	if len(phone) == 10 {
		return "+91" + phone
	}
	return phone // return as-is if unrecognized
}

// Service orchestrates the auth flow.
type Service struct {
	repo     *Repository
	keycloak *KeycloakClient
	otp      *OTPService
	logger   *slog.Logger
}

// NewService creates an auth service.
func NewService(repo *Repository, kc *KeycloakClient, otp *OTPService, logger *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		keycloak: kc,
		otp:      otp,
		logger:   logger,
	}
}

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Phone    string `json:"phone"`
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"` // "landowner" or "agent"
}

// RegisterResponse after registration.
type RegisterResponse struct {
	Message string `json:"message"`
	Phone   string `json:"phone"`
}

// Register creates a user in Keycloak and local DB, then sends OTP.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if req.Phone == "" || req.FullName == "" {
		return nil, platform.NewValidation("phone and full_name are required")
	}
	req.Phone = NormalizePhone(req.Phone)
	if req.Role == "" {
		req.Role = "landowner"
	}
	if req.Role != "landowner" && req.Role != "agent" {
		return nil, platform.NewValidation("role must be 'landowner' or 'agent'")
	}

	// Create user in Keycloak
	kcUser := KeycloakUser{
		Username:  req.Phone,
		Email:     req.Email,
		FirstName: req.FullName,
		Enabled:   true,
		Attributes: map[string][]string{
			"phone_number": {req.Phone},
		},
		RequiredActions: []string{},
	}

	keycloakID, err := s.keycloak.CreateUser(ctx, kcUser)
	if err != nil {
		return nil, fmt.Errorf("creating keycloak user: %w", err)
	}

	// Assign role
	if err := s.keycloak.AssignRealmRole(ctx, keycloakID, req.Role); err != nil {
		s.logger.Error("failed to assign role", "keycloak_id", keycloakID, "role", req.Role, "error", err)
	}

	// Create user in local DB
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	_, err = s.repo.CreateUser(ctx, sqlc.CreateUserParams{
		Phone:      req.Phone,
		Email:      email,
		FullName:   req.FullName,
		Role:       req.Role,
		KeycloakID: &keycloakID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating local user: %w", err)
	}

	// Send OTP
	if err := s.otp.SendOTP(ctx, req.Phone); err != nil {
		s.logger.Error("failed to send OTP", "phone", req.Phone, "error", err)
	}

	return &RegisterResponse{
		Message: "OTP sent to your phone number",
		Phone:   req.Phone,
	}, nil
}

// VerifyOTPRequest is the payload for OTP verification.
type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

// VerifyOTPResponse after successful OTP verification.
type VerifyOTPResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// VerifyOTP verifies the OTP and returns Keycloak tokens.
func (s *Service) VerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error) {
	if req.Phone == "" || req.OTP == "" {
		return nil, platform.NewValidation("phone and otp are required")
	}
	req.Phone = NormalizePhone(req.Phone)

	valid, err := s.otp.VerifyOTP(ctx, req.Phone, req.OTP)
	if err != nil {
		return nil, fmt.Errorf("verifying OTP: %w", err)
	}
	if !valid {
		return nil, platform.NewUnauthorized("invalid or expired OTP")
	}

	// Set a temporary password in Keycloak and exchange for tokens
	// In production, use a proper OTP SPI in Keycloak
	keycloakID, err := s.repo.GetKeycloakIDByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}

	// Use phone as temp password for token exchange after OTP
	tempPass := "otp-verified-" + req.Phone
	if err := s.keycloak.SetTemporaryPassword(ctx, keycloakID, tempPass); err != nil {
		return nil, fmt.Errorf("setting temp password: %w", err)
	}

	tokenResp, err := s.keycloak.GetToken(ctx, req.Phone, tempPass)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	return &VerifyOTPResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

// LoginRequest for phone + OTP login.
type LoginRequest struct {
	Phone string `json:"phone"`
}

// Login sends an OTP for an existing user (landowner or agent).
func (s *Service) Login(ctx context.Context, req LoginRequest) (*RegisterResponse, error) {
	if req.Phone == "" {
		return nil, platform.NewValidation("phone is required")
	}
	req.Phone = NormalizePhone(req.Phone)

	// Verify user exists in either users or agents table
	_, err := s.repo.GetKeycloakIDByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}

	// Send OTP
	if err := s.otp.SendOTP(ctx, req.Phone); err != nil {
		return nil, fmt.Errorf("sending OTP: %w", err)
	}

	return &RegisterResponse{
		Message: "OTP sent to your phone number",
		Phone:   req.Phone,
	}, nil
}

// RefreshRequest for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh exchanges a refresh token for new tokens.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (*VerifyOTPResponse, error) {
	if req.RefreshToken == "" {
		return nil, platform.NewValidation("refresh_token is required")
	}

	tokenResp, err := s.keycloak.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, platform.NewUnauthorized("invalid refresh token")
	}

	return &VerifyOTPResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}
