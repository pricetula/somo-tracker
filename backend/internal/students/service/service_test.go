package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	storegen "somotracker/internal/students/store/gen"
	"somotracker/internal/platform/apperr"
)

// fakeStore implements the Store interface for service unit tests.
type fakeStore struct {
	students       map[uuid.UUID]*storegen.Student
	admissionIndex map[string]uuid.UUID // admission_number -> student_id (within a school)
	guardianLinks  map[uuid.UUID][]storegen.GuardianStudentLink // student_id -> links
	err            error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		students:       make(map[uuid.UUID]*storegen.Student),
		admissionIndex: make(map[string]uuid.UUID),
		guardianLinks:  make(map[uuid.UUID][]storegen.GuardianStudentLink),
	}
}

func (f *fakeStore) GetStudent(ctx context.Context, id uuid.UUID) (*storegen.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.students[id]
	if !ok {
		return nil, &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
	}
	return s, nil
}

func (f *fakeStore) ListStudents(ctx context.Context, limit int32, cursorFullName string, cursorStudentID uuid.UUID) ([]storegen.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	var result []storegen.Student
	count := int32(0)
	for _, s := range f.students {
		if count >= limit {
			break
		}
		result = append(result, *s)
		count++
	}
	return result, nil
}

func (f *fakeStore) CreateStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	// Check duplicate admission_number within school
	if _, exists := f.admissionIndex[admissionNumber]; exists {
		return nil, &apperr.Error{Code: apperr.CONFLICT, Message: "duplicate key violation"}
	}
	id := uuid.New()
	student := &storegen.Student{
		StudentID:       id,
		SchoolID:        schoolID,
		AdmissionNumber: admissionNumber,
		NationalNumber:  nationalNumber,
		FullName:        fullName,
		DateOfBirth:     dateOfBirth,
		Gender:          gender,
		Metadata:        metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	f.students[id] = student
	f.admissionIndex[admissionNumber] = id
	return student, nil
}

func (f *fakeStore) UpdateStudent(ctx context.Context, id uuid.UUID, admissionNumber string, nationalNumber sql.NullString, fullName string, dateOfBirth time.Time, gender string, metadata json.RawMessage) (*storegen.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.students[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	// Check duplicate admission_number (different student)
	if existingID, exists := f.admissionIndex[admissionNumber]; exists && existingID != id {
		return nil, &apperr.Error{Code: apperr.CONFLICT, Message: "duplicate key violation"}
	}
	s.AdmissionNumber = admissionNumber
	s.NationalNumber = nationalNumber
	s.FullName = fullName
	s.DateOfBirth = dateOfBirth
	s.Gender = gender
	s.Metadata = metadata
	s.UpdatedAt = time.Now()
	return s, nil
}

func (f *fakeStore) DeleteStudent(ctx context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.students[id]; !ok {
		return &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
	}
	delete(f.students, id)
	return nil
}

func (f *fakeStore) ListStudentsForGuardian(ctx context.Context, guardianID uuid.UUID, limit int32) ([]storegen.Student, error) {
	if f.err != nil {
		return nil, f.err
	}
	var result []storegen.Student
	// In the fake, we just return all students (guardian filtering is at store/query level)
	for _, s := range f.students {
		result = append(result, *s)
	}
	return result, nil
}

func (f *fakeStore) ListGuardianLinks(ctx context.Context, studentID uuid.UUID) ([]storegen.GuardianStudentLink, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.guardianLinks[studentID], nil
}

func (f *fakeStore) CreateGuardianLink(ctx context.Context, guardianID uuid.UUID, studentID uuid.UUID, relationshipType string, isEmergencyContact bool) (*storegen.GuardianStudentLink, error) {
	if f.err != nil {
		return nil, f.err
	}
	link := storegen.GuardianStudentLink{
		ID:                 uuid.New(),
		GuardianID:         guardianID,
		StudentID:          studentID,
		RelationshipType:   relationshipType,
		IsEmergencyContact: isEmergencyContact,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	f.guardianLinks[studentID] = append(f.guardianLinks[studentID], link)
	return &link, nil
}

func (f *fakeStore) DeleteGuardianLink(ctx context.Context, studentID uuid.UUID, guardianID uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	links := f.guardianLinks[studentID]
	for i, link := range links {
		if link.GuardianID == guardianID {
			f.guardianLinks[studentID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

// fakeTxManager implements a minimal TxManager that runs fn directly (no actual transaction).
type fakeTxManager struct{}

func (f *fakeTxManager) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// Ensure fakeTxManager satisfies the TxRunner interface
var _ TxRunner = (*fakeTxManager)(nil)

// TestService_CreateStudent
func TestService_CreateStudent(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("Success", func(t *testing.T) {
		dto, err := svc.CreateStudent(ctx, CreateStudentRequest{
			SchoolID:        schoolID,
			AdmissionNumber: "A001",
			FullName:        "Alice Johnson",
			DateOfBirth:     dob,
			Gender:          "F",
			Metadata:        []byte(`{}`),
		})
		require.NoError(t, err)
		require.NotNil(t, dto)
		require.Equal(t, schoolID, dto.SchoolID)
		require.Equal(t, "A001", dto.AdmissionNumber)
		require.Equal(t, "Alice Johnson", dto.FullName)
		require.Equal(t, dob, dto.DateOfBirth)
	})

	t.Run("DuplicateAdmissionNumber", func(t *testing.T) {
		_, err := svc.CreateStudent(ctx, CreateStudentRequest{
			SchoolID:        schoolID,
			AdmissionNumber: "A001",
			FullName:        "Bob Smith",
			DateOfBirth:     dob,
			Gender:          "M",
		})
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok)
		require.Equal(t, apperr.CONFLICT, appErr.Code)
	})

	t.Run("StoreError", func(t *testing.T) {
		store.err = errors.New("db down")
		_, err := svc.CreateStudent(ctx, CreateStudentRequest{
			SchoolID:        schoolID,
			AdmissionNumber: "A002",
			FullName:        "Test",
			DateOfBirth:     dob,
			Gender:          "F",
		})
		require.Error(t, err)
		store.err = nil
	})
}

// TestService_GetStudent
func TestService_GetStudent(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	created, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{}, "Alice Johnson", dob, "F", []byte(`{}`))
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		dto, err := svc.GetStudent(ctx, created.StudentID)
		require.NoError(t, err)
		require.Equal(t, created.StudentID, dto.StudentID)
		require.Equal(t, "Alice Johnson", dto.FullName)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := svc.GetStudent(ctx, uuid.New())
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok)
		require.Equal(t, apperr.NOT_FOUND, appErr.Code)
	})
}

// TestService_UpdateStudent
func TestService_UpdateStudent(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	created, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{}, "Alice Johnson", dob, "F", []byte(`{}`))
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		newDob := time.Date(2010, 2, 20, 0, 0, 0, 0, time.UTC)
		dto, err := svc.UpdateStudent(ctx, UpdateStudentRequest{
			StudentID:       created.StudentID,
			AdmissionNumber: "A001-UPDATED",
			FullName:        "Alice Updated",
			DateOfBirth:     newDob,
			Gender:          "F",
			Metadata:        []byte(`{"updated": true}`),
		})
		require.NoError(t, err)
		require.Equal(t, "A001-UPDATED", dto.AdmissionNumber)
		require.Equal(t, "Alice Updated", dto.FullName)
		require.Equal(t, newDob, dto.DateOfBirth)
	})

	t.Run("DuplicateAdmissionNumber", func(t *testing.T) {
		newDob := time.Date(2010, 2, 20, 0, 0, 0, 0, time.UTC)
		// Create another student with a different admission number
		_, err := store.CreateStudent(ctx, schoolID, "A002", sql.NullString{}, "Bob Smith", dob, "M", []byte(`{}`))
		require.NoError(t, err)

		// Try to update first student to use second student's admission number
		_, err = svc.UpdateStudent(ctx, UpdateStudentRequest{
			StudentID:       created.StudentID,
			AdmissionNumber: "A002",
			FullName:        "Alice Updated",
			DateOfBirth:     newDob,
			Gender:          "F",
		})
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok)
		require.Equal(t, apperr.CONFLICT, appErr.Code)
	})
}

// TestService_DeleteStudent
func TestService_DeleteStudent(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	created, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{}, "Alice Johnson", dob, "F", []byte(`{}`))
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		require.NoError(t, svc.DeleteStudent(ctx, created.StudentID))
		_, err := svc.GetStudent(ctx, created.StudentID)
		require.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		require.Error(t, svc.DeleteStudent(ctx, uuid.New()))
	})
}

// TestService_CreateGuardianLink
func TestService_CreateGuardianLink(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	guardianID := uuid.New()
	studentID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		dto, err := svc.CreateGuardianLink(ctx, CreateGuardianLinkRequest{
			GuardianID:         guardianID,
			StudentID:          studentID,
			RelationshipType:   "Parent",
			IsEmergencyContact: true,
		})
		require.NoError(t, err)
		require.Equal(t, guardianID, dto.GuardianID)
		require.Equal(t, studentID, dto.StudentID)
		require.True(t, dto.IsEmergencyContact)
	})

	t.Run("StoreError", func(t *testing.T) {
		store.err = errors.New("db down")
		_, err := svc.CreateGuardianLink(ctx, CreateGuardianLinkRequest{
			GuardianID:         guardianID,
			StudentID:          studentID,
			RelationshipType:   "Parent",
			IsEmergencyContact: false,
		})
		require.Error(t, err)
		store.err = nil
	})
}

// TestService_DeleteGuardianLink
func TestService_DeleteGuardianLink(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	guardianID := uuid.New()
	studentID := uuid.New()

	// Create link first
	_, err := store.CreateGuardianLink(ctx, guardianID, studentID, "Parent", true)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		require.NoError(t, svc.DeleteGuardianLink(ctx, studentID, guardianID))
		links, err := store.ListGuardianLinks(ctx, studentID)
		require.NoError(t, err)
		require.Empty(t, links)
	})

	t.Run("NotFound", func(t *testing.T) {
		require.Error(t, svc.DeleteGuardianLink(ctx, studentID, uuid.New()))
	})
}

// TestService_ListStudentsForGuardian
func TestService_ListStudentsForGuardian(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	// Create students
	_, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{}, "Alice", dob, "F", []byte(`{}`))
	require.NoError(t, err)
	_, err = store.CreateStudent(ctx, schoolID, "A002", sql.NullString{}, "Bob", dob, "M", []byte(`{}`))
	require.NoError(t, err)

	guardianID := uuid.New()

	t.Run("ReturnsAllStudents", func(t *testing.T) {
		// In the fake, ListStudentsForGuardian returns all students
		resp, err := svc.ListStudents(ctx, &guardianID, 10, "", uuid.Nil)
		require.NoError(t, err)
		require.Len(t, resp.Items, 2)
	})
}

// TestService_ListStudents
func TestService_ListStudents(t *testing.T) {
	store := newFakeStore()
	svc := New(store, &fakeTxManager{})

	ctx := context.Background()
	schoolID := uuid.New()
	dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)

	_, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{}, "Alice", dob, "F", []byte(`{}`))
	require.NoError(t, err)
	_, err = store.CreateStudent(ctx, schoolID, "A002", sql.NullString{}, "Bob", dob, "M", []byte(`{}`))
	require.NoError(t, err)

	t.Run("Pagination", func(t *testing.T) {
		resp, err := svc.ListStudents(ctx, nil, 1, "", uuid.Nil)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		require.NotNil(t, resp.NextCursor)
	})
}
