package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	storegen "somotracker/internal/students/store/gen"
	"somotracker/internal/platform/apperr"
)

// Store implements the students domain data layer.
// SQL lives in queries/*.sql, generated to gen/ by sqlc (BE-009).
type Store struct {
	q *storegen.Queries
}

// New creates store.
func New(q *storegen.Queries) *Store {
	return &Store{q: q}
}

// mapPGError converts Postgres errors to apperr errors.
func mapPGError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.New(apperr.NOT_FOUND, "record not found")
	}
	// Check for unique violation (23505)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CONFLICT, "duplicate key violation")
		}
	}
	return apperr.New(apperr.INTERNAL_ERROR, "database error")
}

// GetStudent returns a single student by ID, tenant-scoped.
func (s *Store) GetStudent(ctx context.Context, id uuid.UUID) (*storegen.Student, error) {
	row, err := s.q.GetStudent(ctx, id)
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// ListStudents returns students with keyset pagination, tenant-scoped.
// cursorFullName and cursorStudentID encode the last item from the previous page.
func (s *Store) ListStudents(ctx context.Context, limit int32, cursorFullName string, cursorStudentID uuid.UUID) ([]storegen.Student, error) {
	rows, err := s.q.ListStudents(ctx, storegen.ListStudentsParams{
		Limit:   limit,
		Column2: cursorFullName,
		Column3: cursorStudentID,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return rows, nil
}

// CreateStudent inserts a new student.
func (s *Store) CreateStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error) {
	row, err := s.q.CreateStudent(ctx, storegen.CreateStudentParams{
		SchoolID:        schoolID,
		AdmissionNumber: admissionNumber,
		NationalNumber:  nationalNumber,
		FullName:        fullName,
		DateOfBirth:     dateOfBirth,
		Gender:          gender,
		Metadata:        metadata,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// UpdateStudent updates an existing student (full body).
func (s *Store) UpdateStudent(ctx context.Context, id uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error) {
	row, err := s.q.UpdateStudent(ctx, storegen.UpdateStudentParams{
		StudentID:       id,
		AdmissionNumber: admissionNumber,
		NationalNumber:  nationalNumber,
		FullName:        fullName,
		DateOfBirth:     dateOfBirth,
		Gender:          gender,
		Metadata:        metadata,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// DeleteStudent soft-deletes a student.
func (s *Store) DeleteStudent(ctx context.Context, id uuid.UUID) error {
	err := s.q.DeleteStudent(ctx, id)
	return mapPGError(err)
}

// ListStudentsForGuardian returns students linked to a guardian, tenant-scoped.
func (s *Store) ListStudentsForGuardian(ctx context.Context, guardianID uuid.UUID, limit int32) ([]storegen.Student, error) {
	rows, err := s.q.ListStudentsForGuardian(ctx, storegen.ListStudentsForGuardianParams{
		GuardianID: guardianID,
		Limit:      limit,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return rows, nil
}

// ListGuardianLinks returns guardians linked to a student.
func (s *Store) ListGuardianLinks(ctx context.Context, studentID uuid.UUID) ([]storegen.GuardianStudentLink, error) {
	rows, err := s.q.ListGuardianLinks(ctx, studentID)
	if err != nil {
		return nil, mapPGError(err)
	}
	return rows, nil
}

// CreateGuardianLink links a guardian to a student.
func (s *Store) CreateGuardianLink(ctx context.Context, guardianID uuid.UUID, studentID uuid.UUID, relationshipType string, isEmergencyContact bool) (*storegen.GuardianStudentLink, error) {
	row, err := s.q.CreateGuardianLink(ctx, storegen.CreateGuardianLinkParams{
		GuardianID:         guardianID,
		StudentID:          studentID,
		RelationshipType:   relationshipType,
		IsEmergencyContact: isEmergencyContact,
	})
	if err != nil {
		return nil, mapPGError(err)
	}
	return &row, nil
}

// DeleteGuardianLink unlinks a guardian from a student.
func (s *Store) DeleteGuardianLink(ctx context.Context, studentID uuid.UUID, guardianID uuid.UUID) error {
	err := s.q.DeleteGuardianLink(ctx, storegen.DeleteGuardianLinkParams{
		StudentID:  studentID,
		GuardianID: guardianID,
	})
	return mapPGError(err)
}
