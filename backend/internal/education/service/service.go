package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/education/store"
	"somotracker/internal/platform/logger"
)

// Service handles education business logic: countries, education systems.
type Service struct {
	store store.StoreInterface
	cfg   *config.Config
	log   *logger.Logger
}

// New creates an education service.
func New(s store.StoreInterface, cfg *config.Config, log *logger.Logger) *Service {
	return &Service{store: s, cfg: cfg, log: log}
}

// CountryResponse represents a country in the API response.
type CountryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CountryListResponse is the response for listing countries.
type CountryListResponse struct {
	Countries []CountryResponse `json:"countries"`
}

// EducationSystemLevelResponse represents a level within an education system.
type EducationSystemLevelResponse struct {
	ID          string    `json:"id"`
	LevelName   string    `json:"level_name"`
	SortOrder   int32     `json:"sort_order"`
	YearCount   int32     `json:"year_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// EducationSystemGradingBandResponse represents a grading band.
type EducationSystemGradingBandResponse struct {
	ID         string    `json:"id"`
	Grade      string    `json:"grade"`
	ScaleType  string    `json:"scale_type"`
	RubricLevel *string   `json:"rubric_level,omitempty"`
	MinScore   string    `json:"min_score"`
	MaxScore   string    `json:"max_score"`
	SortOrder  int32     `json:"sort_order"`
	ScaleName  string    `json:"scale_name"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
}

// EducationSystemSubjectResponse represents a subject within an education system.
type EducationSystemSubjectResponse struct {
	ID                  string    `json:"id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	IsCompulsoryDefault bool      `json:"is_compulsory_default"`
	LevelName           string    `json:"level_name"`
	LevelSortOrder      int32     `json:"level_sort_order"`
	CreatedAt           time.Time `json:"created_at"`
}

// EducationSystemResponse represents an education system with all its details.
type EducationSystemResponse struct {
	ID          string                                  `json:"id"`
	CountryID   string                                  `json:"country_id"`
	CountryName string                                  `json:"country_name"`
	CountryCode string                                  `json:"country_code"`
	Name        string                                  `json:"name"`
	Levels      []EducationSystemLevelResponse          `json:"levels"`
	GradingBands []EducationSystemGradingBandResponse   `json:"grading_bands"`
	Subjects    []EducationSystemSubjectResponse        `json:"subjects"`
	CreatedAt   time.Time                               `json:"created_at"`
	UpdatedAt   time.Time                               `json:"updated_at"`
}

// EducationSystemListResponse is the response for listing education systems.
type EducationSystemListResponse struct {
	Systems []EducationSystemListItemResponse `json:"systems"`
}

// EducationSystemListItemResponse is a list item for education systems.
type EducationSystemListItemResponse struct {
	ID          string    `json:"id"`
	CountryID   string    `json:"country_id"`
	CountryName string    `json:"country_name"`
	CountryCode string    `json:"country_code"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListCountries returns all countries.
func (s *Service) ListCountries(ctx context.Context) (CountryListResponse, error) {
	countries, err := s.store.ListCountries(ctx)
	if err != nil {
		s.log.Error(ctx, err, "list countries failed")
		return CountryListResponse{}, apperr.New(apperr.INTERNAL_ERROR, "list countries failed")
	}

	resp := CountryListResponse{
		Countries: make([]CountryResponse, len(countries)),
	}
	for i, c := range countries {
		resp.Countries[i] = CountryResponse{
			ID:        c.ID.String(),
			Name:      c.Name,
			Code:      c.Code,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	return resp, nil
}

// GetEducationSystem returns a single education system with all its details.
func (s *Service) GetEducationSystem(ctx context.Context, id string) (EducationSystemResponse, error) {
	systemID, err := uuid.Parse(id)
	if err != nil {
		return EducationSystemResponse{}, apperr.New(apperr.INVALID_REQUEST, "invalid education system ID")
	}

	system, err := s.store.GetEducationSystem(ctx, systemID)
	if err != nil {
		s.log.Error(ctx, err, "get education system failed", "id", id)
		return EducationSystemResponse{}, apperr.New(apperr.NOT_FOUND, "education system not found")
	}

	levels, err := s.store.ListEducationSystemLevels(ctx, systemID)
	if err != nil {
		s.log.Error(ctx, err, "list education system levels failed", "id", id)
		return EducationSystemResponse{}, apperr.New(apperr.INTERNAL_ERROR, "list levels failed")
	}

	bands, err := s.store.ListEducationSystemGradingBands(ctx, systemID)
	if err != nil {
		s.log.Error(ctx, err, "list grading bands failed", "id", id)
		return EducationSystemResponse{}, apperr.New(apperr.INTERNAL_ERROR, "list grading bands failed")
	}

	subjects, err := s.store.ListEducationSystemSubjects(ctx, systemID)
	if err != nil {
		s.log.Error(ctx, err, "list subjects failed", "id", id)
		return EducationSystemResponse{}, apperr.New(apperr.INTERNAL_ERROR, "list subjects failed")
	}

	resp := EducationSystemResponse{
		ID:          system.ID.String(),
		CountryID:   system.CountryID.String(),
		CountryName: system.CountryName,
		CountryCode: system.CountryCode,
		Name:        system.Name,
		Levels:      make([]EducationSystemLevelResponse, len(levels)),
		GradingBands:        make([]EducationSystemGradingBandResponse, len(bands)),
		Subjects:            make([]EducationSystemSubjectResponse, len(subjects)),
		CreatedAt:           system.CreatedAt,
		UpdatedAt:           system.UpdatedAt,
	}

	for i, l := range levels {
		resp.Levels[i] = EducationSystemLevelResponse{
			ID:          l.EducationLevelID1.String(),
			LevelName:   l.LevelName,
			SortOrder:   l.SortOrder,
			YearCount:   l.YearCount,
			CreatedAt:   l.LevelCreatedAt,
		}
	}

	for i, b := range bands {
		var rubricLevel *string
		if b.RubricLevel.Valid {
			v := b.RubricLevel.String
			rubricLevel = &v
		}
		resp.GradingBands[i] = EducationSystemGradingBandResponse{
			ID:         b.ID.String(),
			Grade:      b.Grade,
			ScaleType:  string(b.ScaleType),
			RubricLevel: rubricLevel,
			MinScore:   b.MinScore,
			MaxScore:   b.MaxScore,
			SortOrder:  b.SortOrder,
			ScaleName:  b.ScaleName,
			IsDefault:  b.IsDefault,
			CreatedAt:  b.CreatedAt,
		}
	}

	for i, subj := range subjects {
		resp.Subjects[i] = EducationSystemSubjectResponse{
			ID:                 subj.ID.String(),
			Code:               subj.Code,
			Name:               subj.Name,
			IsCompulsoryDefault: subj.IsCompulsoryDefault,
			LevelName:          subj.LevelName,
			LevelSortOrder:     subj.SortOrder,
			CreatedAt:          subj.CreatedAt,
		}
	}

	return resp, nil
}

// ListEducationSystems returns all education systems.
func (s *Service) ListEducationSystems(ctx context.Context) (EducationSystemListResponse, error) {
	systems, err := s.store.ListEducationSystems(ctx)
	if err != nil {
		s.log.Error(ctx, err, "list education systems failed")
		return EducationSystemListResponse{}, apperr.New(apperr.INTERNAL_ERROR, "list education systems failed")
	}

	resp := EducationSystemListResponse{
		Systems: make([]EducationSystemListItemResponse, len(systems)),
	}
	for i, sys := range systems {
		resp.Systems[i] = EducationSystemListItemResponse{
			ID:          sys.ID.String(),
			CountryID:   sys.CountryID.String(),
			CountryName: sys.CountryName,
			CountryCode: sys.CountryCode,
			Name:        sys.Name,
			CreatedAt:   sys.CreatedAt,
			UpdatedAt:   sys.UpdatedAt,
		}
	}
	return resp, nil
}
