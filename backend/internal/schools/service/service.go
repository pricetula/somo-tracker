package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	schoolStore "somotracker/internal/schools/store"
	storegen "somotracker/internal/schools/store/gen"
)

// TxRunner is the transaction runner interface used by Service.
// db.TxManager satisfies this interface.
type TxRunner interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store is the data access interface used by Service.
// The real storegen-backed store satisfies this interface.
type Store interface {
	GetSchool(ctx context.Context, id uuid.UUID) (*storegen.School, error)
	ListSchools(ctx context.Context, limit int32, cursorName string, cursorID uuid.UUID) ([]storegen.School, error)
	CreateSchool(ctx context.Context, tenantID uuid.UUID, name string, code string, countryID uuid.UUID, educationSystemID uuid.UUID) (*storegen.School, error)
	UpdateSchool(ctx context.Context, id uuid.UUID, name string, code string, countryID uuid.UUID, educationSystemID uuid.UUID) (*storegen.School, error)
	DeactivateSchools(ctx context.Context, ids []uuid.UUID) error
}

// Service implements the schools domain business logic.
// Depends only on Store interface; never imports handler or sqlc directly (BE-002).
type Service struct {
	store Store
	txm   TxRunner
}

// New creates service.
func New(store Store, txm TxRunner) *Service {
	return &Service{store: store, txm: txm}
}

// Ensure the real storegen-backed store satisfies the Store interface.
var _ Store = (*schoolStore.Store)(nil)

// SchoolDTO is the domain type returned to handlers (never sqlc rows, BE-007).
type SchoolDTO struct {
	ID                uuid.UUID    `json:"id"`
	TenantID          uuid.UUID    `json:"tenant_id"`
	Name              string       `json:"name"`
	Code              string       `json:"code"`
	CountryID         uuid.UUID    `json:"country_id"`
	EducationSystemID uuid.UUID    `json:"education_system_id"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	DeletedAt         *time.Time   `json:"deleted_at,omitempty"`
}

// ListResponse is the paginated list response.
type ListResponse struct {
	Items      []SchoolDTO `json:"items"`
	NextCursor *string     `json:"nextCursor,omitempty"`
}

// mapSchoolDTO converts a sqlc School row to domain DTO.
func mapSchoolDTO(row storegen.School) SchoolDTO {
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}
	return SchoolDTO{
		ID:                row.ID,
		TenantID:          row.TenantID,
		Name:              row.Name,
		Code:              row.Code,
		CountryID:         row.CountryID,
		EducationSystemID: row.EducationSystemID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		DeletedAt:         deletedAt,
	}
}

// ListSchools returns paginated schools for the current tenant.
func (s *Service) ListSchools(ctx context.Context, limit int32, cursorName string, cursorID uuid.UUID) (*ListResponse, error) {
	rows, err := s.store.ListSchools(ctx, limit, cursorName, cursorID)
	if err != nil {
		return nil, err
	}

	items := make([]SchoolDTO, len(rows))
	for i, row := range rows {
		items[i] = mapSchoolDTO(row)
	}

	var nextCursor *string
	if len(items) == int(limit) {
		last := items[len(items)-1]
		cursor := last.Name + "|" + last.ID.String()
		nextCursor = &cursor
	}

	return &ListResponse{Items: items, NextCursor: nextCursor}, nil
}

// GetSchool returns a single school by ID.
func (s *Service) GetSchool(ctx context.Context, id uuid.UUID) (*SchoolDTO, error) {
	row, err := s.store.GetSchool(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := mapSchoolDTO(*row)
	return &dto, nil
}

// CreateSchoolRequest is the input for creating a school.
type CreateSchoolRequest struct {
	TenantID          uuid.UUID `json:"tenant_id"`
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	CountryID         uuid.UUID `json:"country_id"`
	EducationSystemID uuid.UUID `json:"education_system_id"`
}

// CreateSchool creates a new school in a transaction.
func (s *Service) CreateSchool(ctx context.Context, req CreateSchoolRequest) (*SchoolDTO, error) {
	var result *SchoolDTO
	err := s.txm.Run(ctx, func(ctx context.Context) error {
		row, err := s.store.CreateSchool(ctx, req.TenantID, req.Name, req.Code, req.CountryID, req.EducationSystemID)
		if err != nil {
			return err
		}
		dto := mapSchoolDTO(*row)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateSchoolRequest is the input for updating a school (full body).
type UpdateSchoolRequest struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	CountryID         uuid.UUID `json:"country_id"`
	EducationSystemID uuid.UUID `json:"education_system_id"`
}

// UpdateSchool updates a school in a transaction.
func (s *Service) UpdateSchool(ctx context.Context, req UpdateSchoolRequest) (*SchoolDTO, error) {
	var result *SchoolDTO
	err := s.txm.Run(ctx, func(ctx context.Context) error {
		row, err := s.store.UpdateSchool(ctx, req.ID, req.Name, req.Code, req.CountryID, req.EducationSystemID)
		if err != nil {
			return err
		}
		dto := mapSchoolDTO(*row)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeactivateSchoolsRequest is the input for deactivating schools.
type DeactivateSchoolsRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

// DeactivateSchools soft-deletes schools in a transaction.
func (s *Service) DeactivateSchools(ctx context.Context, ids []uuid.UUID) error {
	return s.txm.Run(ctx, func(ctx context.Context) error {
		return s.store.DeactivateSchools(ctx, ids)
	})
}
