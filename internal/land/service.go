package land

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Service orchestrates parcel business logic.
type Service struct {
	repo     *Repository
	authRepo *auth.Repository
	eventBus *platform.EventBus
	logger   *slog.Logger
}

// NewService creates a land service.
func NewService(repo *Repository, authRepo *auth.Repository, eventBus *platform.EventBus, logger *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		authRepo: authRepo,
		eventBus: eventBus,
		logger:   logger,
	}
}

// NewServiceForTest creates a Service with nil dependencies for validation-only tests.
func NewServiceForTest() *Service {
	return &Service{}
}

// CreateParcelRequest is the payload for creating a parcel.
type CreateParcelRequest struct {
	Label             string   `json:"label"`
	SurveyNumber      string   `json:"survey_number,omitempty"`
	Village           string   `json:"village,omitempty"`
	Taluk             string   `json:"taluk,omitempty"`
	District          string   `json:"district"`
	State             string   `json:"state"`
	StateCode         string   `json:"state_code"`
	PinCode           string   `json:"pin_code,omitempty"`
	Boundary          string   `json:"boundary"` // GeoJSON string
	LandType          string   `json:"land_type,omitempty"`
	RegisteredAreaSqm *float32 `json:"registered_area_sqm,omitempty"`
	TitleDeedS3Key    string   `json:"title_deed_s3_key,omitempty"`
}

// ParcelResponse is returned after creating or getting a parcel.
type ParcelResponse struct {
	ID                uuid.UUID `json:"id"`
	Label             *string   `json:"label"`
	SurveyNumber      *string   `json:"survey_number,omitempty"`
	Village           *string   `json:"village,omitempty"`
	Taluk             *string   `json:"taluk,omitempty"`
	District          string    `json:"district"`
	State             string    `json:"state"`
	StateCode         string    `json:"state_code"`
	PinCode           *string   `json:"pin_code,omitempty"`
	BoundaryGeoJSON   any       `json:"boundary_geojson,omitempty"`
	AreaSqm           *float32  `json:"area_sqm,omitempty"`
	LandType          *string   `json:"land_type,omitempty"`
	RegisteredAreaSqm *float32  `json:"registered_area_sqm,omitempty"`
	Status            *string   `json:"status"`
}

// UpdateBoundaryRequest is the payload for updating a parcel boundary.
type UpdateBoundaryRequest struct {
	Boundary string `json:"boundary"` // GeoJSON string
}

// CreateParcel creates a new parcel for the authenticated landowner.
func (s *Service) CreateParcel(ctx context.Context, userCtx *auth.UserContext, req CreateParcelRequest) (*ParcelResponse, error) {
	if req.District == "" || req.State == "" || req.StateCode == "" {
		return nil, platform.NewValidation("district, state, and state_code are required")
	}
	if req.Boundary == "" {
		return nil, platform.NewValidation("boundary is required")
	}
	if err := ValidateBoundaryGeoJSON(req.Boundary); err != nil {
		return nil, err
	}

	// Resolve keycloak_id to local user
	user, err := s.authRepo.GetUserByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return nil, err
	}

	params := sqlc.CreateParcelParams{
		UserID:           user.ID,
		District:         req.District,
		State:            req.State,
		StateCode:        req.StateCode,
		StGeomfromgeojson: req.Boundary,
	}
	if req.Label != "" {
		params.Label = &req.Label
	}
	if req.SurveyNumber != "" {
		params.SurveyNumber = &req.SurveyNumber
	}
	if req.Village != "" {
		params.Village = &req.Village
	}
	if req.Taluk != "" {
		params.Taluk = &req.Taluk
	}
	if req.PinCode != "" {
		params.PinCode = &req.PinCode
	}
	if req.LandType != "" {
		params.LandType = &req.LandType
	}
	if req.RegisteredAreaSqm != nil {
		params.RegisteredAreaSqm = req.RegisteredAreaSqm
	}
	if req.TitleDeedS3Key != "" {
		params.TitleDeedS3Key = &req.TitleDeedS3Key
	}

	parcel, err := s.repo.CreateParcel(ctx, params)
	if err != nil {
		return nil, err
	}

	// Publish event
	s.eventBus.Publish(platform.Event{
		Type:    "parcel.registered",
		Payload: parcel,
	})

	s.logger.Info("parcel created", "parcel_id", parcel.ID, "user_id", user.ID)

	return &ParcelResponse{
		ID:                parcel.ID,
		Label:             parcel.Label,
		SurveyNumber:      parcel.SurveyNumber,
		Village:           parcel.Village,
		Taluk:             parcel.Taluk,
		District:          parcel.District,
		State:             parcel.State,
		StateCode:         parcel.StateCode,
		PinCode:           parcel.PinCode,
		AreaSqm:           parcel.AreaSqm,
		LandType:          parcel.LandType,
		RegisteredAreaSqm: parcel.RegisteredAreaSqm,
		Status:            parcel.Status,
	}, nil
}

// ListParcels returns paginated parcels for the authenticated landowner.
func (s *Service) ListParcels(ctx context.Context, userCtx *auth.UserContext, page, perPage int) ([]ParcelResponse, int64, error) {
	user, err := s.authRepo.GetUserByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return nil, 0, err
	}

	offset := int32((page - 1) * perPage)
	parcels, err := s.repo.ListParcelsByUser(ctx, user.ID, int32(perPage), offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountParcelsByUser(ctx, user.ID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]ParcelResponse, len(parcels))
	for i, p := range parcels {
		result[i] = ParcelResponse{
			ID:                p.ID,
			Label:             p.Label,
			SurveyNumber:      p.SurveyNumber,
			Village:           p.Village,
			Taluk:             p.Taluk,
			District:          p.District,
			State:             p.State,
			StateCode:         p.StateCode,
			PinCode:           p.PinCode,
			AreaSqm:           p.AreaSqm,
			LandType:          p.LandType,
			RegisteredAreaSqm: p.RegisteredAreaSqm,
			Status:            p.Status,
		}
	}

	return result, total, nil
}

// GetParcel returns a single parcel detail with boundary as GeoJSON.
func (s *Service) GetParcel(ctx context.Context, userCtx *auth.UserContext, parcelID uuid.UUID) (*ParcelResponse, error) {
	user, err := s.authRepo.GetUserByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return nil, err
	}

	row, err := s.repo.GetParcelWithGeoJSON(ctx, parcelID)
	if err != nil {
		return nil, err
	}

	// Owner check
	if row.UserID != user.ID {
		return nil, platform.NewForbidden("you do not own this parcel")
	}

	return &ParcelResponse{
		ID:                row.ID,
		Label:             row.Label,
		SurveyNumber:      row.SurveyNumber,
		Village:           row.Village,
		Taluk:             row.Taluk,
		District:          row.District,
		State:             row.State,
		StateCode:         row.StateCode,
		PinCode:           row.PinCode,
		BoundaryGeoJSON:   row.BoundaryGeojson,
		AreaSqm:           row.AreaSqm,
		LandType:          row.LandType,
		RegisteredAreaSqm: row.RegisteredAreaSqm,
		Status:            row.Status,
	}, nil
}

// UpdateBoundary updates the parcel boundary geometry.
func (s *Service) UpdateBoundary(ctx context.Context, userCtx *auth.UserContext, parcelID uuid.UUID, req UpdateBoundaryRequest) error {
	if req.Boundary == "" {
		return platform.NewValidation("boundary is required")
	}
	if err := ValidateBoundaryGeoJSON(req.Boundary); err != nil {
		return err
	}

	user, err := s.authRepo.GetUserByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	// Verify ownership
	parcel, err := s.repo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return err
	}
	if parcel.UserID != user.ID {
		return platform.NewForbidden("you do not own this parcel")
	}

	return s.repo.UpdateParcelBoundary(ctx, parcelID, req.Boundary)
}

// DeleteParcel soft-deletes a parcel.
func (s *Service) DeleteParcel(ctx context.Context, userCtx *auth.UserContext, parcelID uuid.UUID) error {
	user, err := s.authRepo.GetUserByKeycloakID(ctx, userCtx.KeycloakID)
	if err != nil {
		return err
	}

	parcel, err := s.repo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return err
	}
	if parcel.UserID != user.ID {
		return platform.NewForbidden("you do not own this parcel")
	}

	return s.repo.DeleteParcel(ctx, parcelID)
}
