package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	studentStore "somotracker/internal/students/store"
	storegen "somotracker/internal/students/store/gen"
)

// TxRunner is the transaction runner interface used by Service.
// db.TxManager satisfies this interface.
type TxRunner interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store is the data access interface used by Service.
// The real storegen-backed store satisfies this interface.
type Store interface {
	GetStudent(ctx context.Context, id uuid.UUID) (*storegen.Student, error)
	ListStudents(ctx context.Context, limit int32, cursorFullName string, cursorStudentID uuid.UUID) ([]storegen.Student, error)
	CreateStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error)
	UpdateStudent(ctx context.Context, id uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error)
	DeleteStudent(ctx context.Context, id uuid.UUID) error
	ListStudentsForGuardian(ctx context.Context, guardianID uuid.UUID, limit int32) ([]storegen.Student, error)
	ListGuardianLinks(ctx context.Context, studentID uuid.UUID) ([]storegen.GuardianStudentLink, error)
	CreateGuardianLink(ctx context.Context, guardianID uuid.UUID, studentID uuid.UUID, relationshipType string, isEmergencyContact bool) (*storegen.GuardianStudentLink, error)
	DeleteGuardianLink(ctx context.Context, studentID uuid.UUID, guardianID uuid.UUID) error
}

// Service implements the students domain business logic.
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
var _ Store = (*studentStore.Store)(nil)

// StudentDTO is the domain type returned to handlers (never sqlc rows, BE-007).
type StudentDTO struct {
	StudentID       uuid.UUID      `json:"student_id"`
	SchoolID        uuid.UUID      `json:"school_id"`
	AdmissionNumber string         `json:"admission_number"`
	NationalNumber  *string        `json:"national_number,omitempty"`
	FullName        string         `json:"full_name"`
	DateOfBirth     time.Time      `json:"date_of_birth"`
	Gender          string         `json:"gender"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// GuardianLinkDTO is the domain type for guardian links.
type GuardianLinkDTO struct {
	GuardianID         uuid.UUID `json:"guardian_id"`
	StudentID          uuid.UUID `json:"student_id"`
	RelationshipType   string    `json:"relationship_type"`
	IsEmergencyContact bool      `json:"is_emergency_contact"`
}

// ListResponse is the paginated list response.
type ListResponse struct {
	Items      []StudentDTO `json:"items"`
	NextCursor *string      `json:"nextCursor,omitempty"`
}

// mapStudentDTO converts a sqlc Student row to domain DTO.
func mapStudentDTO(row storegen.Student) StudentDTO {
	var nationalNumber *string
	if row.NationalNumber.Valid {
		nationalNumber = &row.NationalNumber.String
	}
	return StudentDTO{
		StudentID:       row.StudentID,
		SchoolID:        row.SchoolID,
		AdmissionNumber: row.AdmissionNumber,
		NationalNumber:  nationalNumber,
		FullName:        row.FullName,
		DateOfBirth:     row.DateOfBirth,
		Gender:          row.Gender,
		Metadata:        row.Metadata,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

// mapGuardianLinkDTO converts a sqlc GuardianStudentLink row to domain DTO.
func mapGuardianLinkDTO(row storegen.GuardianStudentLink) GuardianLinkDTO {
	return GuardianLinkDTO{
		GuardianID:         row.GuardianID,
		StudentID:          row.StudentID,
		RelationshipType:   row.RelationshipType,
		IsEmergencyContact: row.IsEmergencyContact,
	}
}

// ListStudents returns paginated students for the current tenant.
// If guardianID is provided, returns only students linked to that guardian.
func (s *Service) ListStudents(ctx context.Context, guardianID *uuid.UUID, limit int32, cursorFullName string, cursorStudentID uuid.UUID) (*ListResponse, error) {
	var rows []storegen.Student
	var err error

	if guardianID != nil {
		// Guardian scope: only linked students
		rows, err = s.store.ListStudentsForGuardian(ctx, *guardianID, limit)
	} else {
		rows, err = s.store.ListStudents(ctx, limit, cursorFullName, cursorStudentID)
	}
	if err != nil {
		return nil, err
	}

	items := make([]StudentDTO, len(rows))
	for i, row := range rows {
		items[i] = mapStudentDTO(row)
	}

	var nextCursor *string
	if len(items) == int(limit) {
		last := items[len(items)-1]
		cursor := last.FullName + "|" + last.StudentID.String()
		nextCursor = &cursor
	}

	return &ListResponse{Items: items, NextCursor: nextCursor}, nil
}

// GetStudent returns a single student by ID.
func (s *Service) GetStudent(ctx context.Context, id uuid.UUID) (*StudentDTO, error) {
	row, err := s.store.GetStudent(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := mapStudentDTO(*row)
	return &dto, nil
}

// CreateStudentRequest is the input for creating a student.
type CreateStudentRequest struct {
	SchoolID        uuid.UUID      `json:"school_id"`
	AdmissionNumber string         `json:"admission_number"`
	NationalNumber  *string        `json:"national_number,omitempty"`
	FullName        string         `json:"full_name"`
	DateOfBirth     time.Time      `json:"date_of_birth"`
	Gender          string         `json:"gender"`
	Metadata        json.RawMessage `json:"metadata"`
}

// CreateStudent creates a new student in a transaction.
func (s *Service) CreateStudent(ctx context.Context, req CreateStudentRequest) (*StudentDTO, error) {
	var nationalNumber sql.NullString
	if req.NationalNumber != nil {
		nationalNumber = sql.NullString{String: *req.NationalNumber, Valid: true}
	}

	var result *StudentDTO
	err := s.txm.Run(ctx, func(ctx context.Context) error {
		row, err := s.store.CreateStudent(ctx, req.SchoolID, req.AdmissionNumber, nationalNumber, req.FullName, req.DateOfBirth, req.Gender, req.Metadata)
		if err != nil {
			return err
		}
		dto := mapStudentDTO(*row)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateStudentRequest is the input for updating a student (full body).
type UpdateStudentRequest struct {
	StudentID       uuid.UUID      `json:"student_id"`
	AdmissionNumber string         `json:"admission_number"`
	NationalNumber  *string        `json:"national_number,omitempty"`
	FullName        string         `json:"full_name"`
	DateOfBirth     time.Time      `json:"date_of_birth"`
	Gender          string         `json:"gender"`
	Metadata        json.RawMessage `json:"metadata"`
}

// UpdateStudent updates a student in a transaction.
func (s *Service) UpdateStudent(ctx context.Context, req UpdateStudentRequest) (*StudentDTO, error) {
	var nationalNumber sql.NullString
	if req.NationalNumber != nil {
		nationalNumber = sql.NullString{String: *req.NationalNumber, Valid: true}
	}

	var result *StudentDTO
	err := s.txm.Run(ctx, func(ctx context.Context) error {
		row, err := s.store.UpdateStudent(ctx, req.StudentID, req.AdmissionNumber, nationalNumber, req.FullName, req.DateOfBirth, req.Gender, req.Metadata)
		if err != nil {
			return err
		}
		dto := mapStudentDTO(*row)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteStudent soft-deletes a student in a transaction.
func (s *Service) DeleteStudent(ctx context.Context, id uuid.UUID) error {
	return s.txm.Run(ctx, func(ctx context.Context) error {
		return s.store.DeleteStudent(ctx, id)
	})
}

// ListGuardianLinks returns guardians linked to a student.
func (s *Service) ListGuardianLinks(ctx context.Context, studentID uuid.UUID) ([]GuardianLinkDTO, error) {
	rows, err := s.store.ListGuardianLinks(ctx, studentID)
	if err != nil {
		return nil, err
	}
	items := make([]GuardianLinkDTO, len(rows))
	for i, row := range rows {
		items[i] = mapGuardianLinkDTO(row)
	}
	return items, nil
}

// CreateGuardianLinkRequest is the input for linking a guardian to a student.
type CreateGuardianLinkRequest struct {
	GuardianID         uuid.UUID `json:"guardian_id"`
	StudentID          uuid.UUID `json:"student_id"`
	RelationshipType   string    `json:"relationship_type"`
	IsEmergencyContact bool      `json:"is_emergency_contact"`
}

// CreateGuardianLink links a guardian to a student in a transaction.
func (s *Service) CreateGuardianLink(ctx context.Context, req CreateGuardianLinkRequest) (*GuardianLinkDTO, error) {
	var result *GuardianLinkDTO
	err := s.txm.Run(ctx, func(ctx context.Context) error {
		row, err := s.store.CreateGuardianLink(ctx, req.GuardianID, req.StudentID, req.RelationshipType, req.IsEmergencyContact)
		if err != nil {
			return err
		}
		dto := mapGuardianLinkDTO(*row)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteGuardianLink unlinks a guardian from a student in a transaction.
func (s *Service) DeleteGuardianLink(ctx context.Context, studentID uuid.UUID, guardianID uuid.UUID) error {
	return s.txm.Run(ctx, func(ctx context.Context) error {
		return s.store.DeleteGuardianLink(ctx, studentID, guardianID)
	})
}
