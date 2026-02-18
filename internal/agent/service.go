package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Service orchestrates agent business logic.
type Service struct {
	repo     *Repository
	rdb      *redis.Client
	keycloak *auth.KeycloakClient
	otp      *auth.OTPService
	logger   *slog.Logger
}

// NewService creates an agent service.
func NewService(repo *Repository, rdb *redis.Client, kc *auth.KeycloakClient, otp *auth.OTPService, logger *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		rdb:      rdb,
		keycloak: kc,
		otp:      otp,
		logger:   logger,
	}
}

// NewServiceForTest creates a Service with nil dependencies for validation-only tests.
func NewServiceForTest() *Service {
	return &Service{}
}

// RegisterRequest is the payload for agent registration.
type RegisterRequest struct {
	Phone     string  `json:"phone"`
	FullName  string  `json:"full_name"`
	Email     string  `json:"email,omitempty"`
	HomeLng   float64 `json:"home_lng,omitempty"`
	HomeLat   float64 `json:"home_lat,omitempty"`
	StateCode string  `json:"state_code,omitempty"`
	District  string  `json:"district_code,omitempty"`
}

// RegisterResponse after agent registration.
type RegisterResponse struct {
	Message string `json:"message"`
	Phone   string `json:"phone"`
}

// AgentProfileResponse is the agent's profile.
type AgentProfileResponse struct {
	ID                string   `json:"id"`
	FullName          string   `json:"full_name"`
	Phone             string   `json:"phone"`
	Email             *string  `json:"email,omitempty"`
	VehicleType       *string  `json:"vehicle_type,omitempty"`
	PreferredRadiusKm *int32   `json:"preferred_radius_km,omitempty"`
	Status            *string  `json:"status"`
	Tier              *string  `json:"tier,omitempty"`
	IsOnline          *bool    `json:"is_online"`
	AvailableDays     []string `json:"available_days,omitempty"`
	TotalJobsCompleted *int32  `json:"total_jobs_completed,omitempty"`
}

// UpdateProfileRequest for updating agent profile.
type UpdateProfileRequest struct {
	FullName          string   `json:"full_name,omitempty"`
	Email             *string  `json:"email,omitempty"`
	VehicleType       *string  `json:"vehicle_type,omitempty"`
	PreferredRadiusKm *int32   `json:"preferred_radius_km,omitempty"`
	BankAccountEnc    *string  `json:"bank_account_enc,omitempty"`
	BankIfsc          *string  `json:"bank_ifsc,omitempty"`
	UpiID             *string  `json:"upi_id,omitempty"`
	AvailableDays     []string `json:"available_days,omitempty"`
	AvailableStart    string   `json:"available_start,omitempty"`
	AvailableEnd      string   `json:"available_end,omitempty"`
}

// LocationRequest for updating agent location.
type LocationRequest struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
}

// AvailabilityRequest for toggling availability.
type AvailabilityRequest struct {
	IsOnline bool `json:"is_online"`
}

// FCMTokenRequest for updating FCM token.
type FCMTokenRequest struct {
	FCMToken   string `json:"fcm_token"`
	DeviceID   string `json:"device_id,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}

// Register creates an agent in Keycloak and local DB, then sends OTP.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if req.Phone == "" || req.FullName == "" {
		return nil, platform.NewValidation("phone and full_name are required")
	}

	// Create user in Keycloak
	kcUser := auth.KeycloakUser{
		Username:  req.Phone,
		Email:     req.Email,
		FirstName: req.FullName,
		Enabled:   true,
		Attributes: map[string][]string{
			"phone_number": {req.Phone},
		},
	}

	keycloakID, err := s.keycloak.CreateUser(ctx, kcUser)
	if err != nil {
		return nil, err
	}

	// Assign agent role
	if err := s.keycloak.AssignRealmRole(ctx, keycloakID, "agent"); err != nil {
		s.logger.Error("failed to assign agent role", "keycloak_id", keycloakID, "error", err)
	}

	// Create agent in local DB
	params := sqlc.CreateAgentParams{
		FullName:      req.FullName,
		Phone:         req.Phone,
		StMakepoint:   req.HomeLng,
		StMakepoint_2: req.HomeLat,
		KeycloakID:    &keycloakID,
	}
	if req.Email != "" {
		params.Email = &req.Email
	}
	if req.StateCode != "" {
		params.StateCode = &req.StateCode
	}
	if req.District != "" {
		params.DistrictCode = &req.District
	}

	if _, err := s.repo.CreateAgent(ctx, params); err != nil {
		return nil, err
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

// GetProfile returns the agent's own profile.
func (s *Service) GetProfile(ctx context.Context, userCtx *auth.UserContext) (*AgentProfileResponse, error) {
	agent, err := s.repo.GetAgentByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return nil, err
	}

	return &AgentProfileResponse{
		ID:                 agent.ID.String(),
		FullName:           agent.FullName,
		Phone:              agent.Phone,
		Email:              agent.Email,
		VehicleType:        agent.VehicleType,
		PreferredRadiusKm:  agent.PreferredRadiusKm,
		Status:             agent.Status,
		Tier:               agent.Tier,
		IsOnline:           agent.IsOnline,
		AvailableDays:      agent.AvailableDays,
		TotalJobsCompleted: agent.TotalJobsCompleted,
	}, nil
}

// UpdateProfile updates the agent's profile fields.
func (s *Service) UpdateProfile(ctx context.Context, userCtx *auth.UserContext, req UpdateProfileRequest) error {
	agent, err := s.repo.GetAgentByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	params := sqlc.UpdateAgentProfileParams{
		ID:                agent.ID,
		FullName:          req.FullName,
		Email:             req.Email,
		VehicleType:       req.VehicleType,
		PreferredRadiusKm: req.PreferredRadiusKm,
		BankAccountEnc:    req.BankAccountEnc,
		BankIfsc:          req.BankIfsc,
		UpiID:             req.UpiID,
		AvailableDays:     req.AvailableDays,
	}

	// Parse time fields if provided
	if req.AvailableStart != "" {
		params.AvailableStart = pgtype.Time{Valid: true}
		// Parse HH:MM format
		params.AvailableStart.Microseconds = parseTimeMicroseconds(req.AvailableStart)
	}
	if req.AvailableEnd != "" {
		params.AvailableEnd = pgtype.Time{Valid: true}
		params.AvailableEnd.Microseconds = parseTimeMicroseconds(req.AvailableEnd)
	}

	return s.repo.UpdateAgentProfile(ctx, params)
}

// UpdateLocation writes agent location to Redis (hot path).
func (s *Service) UpdateLocation(ctx context.Context, userCtx *auth.UserContext, req LocationRequest) error {
	if err := ValidateLocation(req.Lat, req.Lng, req.Accuracy); err != nil {
		return err
	}

	agent, err := s.repo.GetAgentByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	return UpdateLocationRedis(ctx, s.rdb, agent.ID, req.Lat, req.Lng, req.Accuracy)
}

// UpdateAvailability toggles the agent's is_online status.
func (s *Service) UpdateAvailability(ctx context.Context, userCtx *auth.UserContext, req AvailabilityRequest) error {
	agent, err := s.repo.GetAgentByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	return s.repo.UpdateAgentOnlineStatus(ctx, agent.ID, req.IsOnline)
}

// UpdateFCMToken registers the agent's FCM push token.
func (s *Service) UpdateFCMToken(ctx context.Context, userCtx *auth.UserContext, req FCMTokenRequest) error {
	if req.FCMToken == "" {
		return platform.NewValidation("fcm_token is required")
	}

	agent, err := s.repo.GetAgentByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	params := sqlc.UpdateAgentFCMTokenParams{
		ID:       agent.ID,
		FcmToken: &req.FCMToken,
	}
	if req.DeviceID != "" {
		params.DeviceID = &req.DeviceID
	}
	if req.AppVersion != "" {
		params.AppVersion = &req.AppVersion
	}

	return s.repo.UpdateAgentFCMToken(ctx, params)
}

// parseTimeMicroseconds converts "HH:MM" to microseconds since midnight.
func parseTimeMicroseconds(t string) int64 {
	if len(t) < 5 {
		return 0
	}
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return int64(h)*3600000000 + int64(m)*60000000
}
