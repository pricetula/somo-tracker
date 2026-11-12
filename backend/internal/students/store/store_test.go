//go:build integration

package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	storegen "somotracker/internal/students/store/gen"
	"somotracker/internal/platform/apperr"
)

func TestStore_Integration_StudentCRUD(t *testing.T) {
	ctx := context.Background()

	postgresC, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:18"),
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "start postgres container")
	defer func() {
		require.NoError(t, postgresC.Terminate(ctx))
	}()

	dsn, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer sqlDB.Close()

	// Apply migrations 01-07
	goose.SetTableName("goose_db_version")
	require.NoError(t, goose.UpContext(ctx, sqlDB, "../../migrations"), "apply migrations")

	// Create a tenant and school first (needed for FK constraints)
	var tenantID uuid.UUID
	require.NoError(t, sqlDB.QueryRowContext(ctx, "INSERT INTO tenants (name) VALUES ('test-tenant') RETURNING id").Scan(&tenantID))

	var schoolID uuid.UUID
	require.NoError(t, sqlDB.QueryRowContext(ctx, "INSERT INTO schools (tenant_id, name, code, country_id, education_system_id) VALUES ($1, 'Test School', 'TS', NULL, NULL) RETURNING id", tenantID).Scan(&schoolID))

	queries := storegen.New(sqlDB)
	store := New(queries)

	// Test CreateStudent
	t.Run("CreateStudent", func(t *testing.T) {
		dob := time.Date(2010, 1, 15, 0, 0, 0, 0, time.UTC)
		student, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{String: "N123456", Valid: true}, "Alice Johnson", dob, "F", []byte(`{}`))
		require.NoError(t, err)
		require.NotNil(t, student)
		require.NotEqual(t, uuid.Nil, student.StudentID)
		require.Equal(t, schoolID, student.SchoolID)
		require.Equal(t, "A001", student.AdmissionNumber)
		require.Equal(t, "Alice Johnson", student.FullName)
	})

	// Test CreateStudent duplicate admission_number -> CONFLICT
	t.Run("CreateStudent_DuplicateAdmissionNumber", func(t *testing.T) {
		dob := time.Date(2010, 2, 20, 0, 0, 0, 0, time.UTC)
		_, err := store.CreateStudent(ctx, schoolID, "A001", sql.NullString{String: "N654321", Valid: true}, "Bob Smith", dob, "M", []byte(`{}`))
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok, "expected *apperr.Error, got %T", err)
		require.Equal(t, "CONFLICT", appErr.Code.String())
	})

	// Test GetStudent
	t.Run("GetStudent", func(t *testing.T) {
		// First create a student to retrieve
		dob := time.Date(2010, 3, 10, 0, 0, 0, 0, time.UTC)
		created, err := store.CreateStudent(ctx, schoolID, "A002", sql.NullString{}, "Charlie Brown", dob, "M", []byte(`{}`))
		require.NoError(t, err)

		student, err := store.GetStudent(ctx, created.StudentID)
		require.NoError(t, err)
		require.Equal(t, created.StudentID, student.StudentID)
		require.Equal(t, "Charlie Brown", student.FullName)
	})

	// Test GetStudent not found
	t.Run("GetStudent_NotFound", func(t *testing.T) {
		_, err := store.GetStudent(ctx, uuid.New())
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok, "expected *apperr.Error, got %T", err)
		require.Equal(t, "NOT_FOUND", appErr.Code.String())
	})

	// Test ListStudents with pagination
	t.Run("ListStudents", func(t *testing.T) {
		students, err := store.ListStudents(ctx, 10, "", uuid.Nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(students), 3) // at least the 3 we created
	})

	// Test UpdateStudent
	t.Run("UpdateStudent", func(t *testing.T) {
		dob := time.Date(2010, 4, 5, 0, 0, 0, 0, time.UTC)
		created, err := store.CreateStudent(ctx, schoolID, "A003", sql.NullString{}, "Diana Prince", dob, "F", []byte(`{}`))
		require.NoError(t, err)

		updatedDob := time.Date(2010, 4, 5, 0, 0, 0, 0, time.UTC)
		updated, err := store.UpdateStudent(ctx, created.StudentID, "A003-UPDATED", sql.NullString{}, "Diana Prince Updated", updatedDob, "F", []byte(`{"updated": true}`))
		require.NoError(t, err)
		require.Equal(t, "A003-UPDATED", updated.AdmissionNumber)
		require.Equal(t, "Diana Prince Updated", updated.FullName)
	})

	// Test DeleteStudent (soft delete)
	t.Run("DeleteStudent", func(t *testing.T) {
		dob := time.Date(2010, 5, 1, 0, 0, 0, 0, time.UTC)
		created, err := store.CreateStudent(ctx, schoolID, "A004", sql.NullString{}, "Eve Wilson", dob, "F", []byte(`{}`))
		require.NoError(t, err)

		require.NoError(t, store.DeleteStudent(ctx, created.StudentID))

		// Should not be found after soft delete
		_, err = store.GetStudent(ctx, created.StudentID)
		require.Error(t, err)
		appErr, ok := err.(*apperr.Error)
		require.True(t, ok, "expected *apperr.Error, got %T", err)
		require.Equal(t, "NOT_FOUND", appErr.Code.String())
	})

	// Test Guardian link CRUD
	t.Run("GuardianLink", func(t *testing.T) {
		// Create a student
		dob := time.Date(2010, 6, 1, 0, 0, 0, 0, time.UTC)
		student, err := store.CreateStudent(ctx, schoolID, "A005", sql.NullString{}, "Frank Miller", dob, "M", []byte(`{}`))
		require.NoError(t, err)

		// Create a guardian membership first
		var guardianMembershipID uuid.UUID
		require.NoError(t, sqlDB.QueryRowContext(ctx, "INSERT INTO memberships (user_id, school_id, role) VALUES ($1, $2, 'guardian') RETURNING id", uuid.New(), schoolID).Scan(&guardianMembershipID))

		// Create guardian link
		link, err := store.CreateGuardianLink(ctx, guardianMembershipID, student.StudentID, "Parent", true)
		require.NoError(t, err)
		require.Equal(t, guardianMembershipID, link.GuardianID)
		require.Equal(t, student.StudentID, link.StudentID)
		require.True(t, link.IsEmergencyContact)

		// List guardian links
		links, err := store.ListGuardianLinks(ctx, student.StudentID)
		require.NoError(t, err)
		require.Len(t, links, 1)
		require.Equal(t, guardianMembershipID, links[0].GuardianID)

		// Delete guardian link
		require.NoError(t, store.DeleteGuardianLink(ctx, student.StudentID, guardianMembershipID))

		// Should be gone
		links, err = store.ListGuardianLinks(ctx, student.StudentID)
		require.NoError(t, err)
		require.Empty(t, links)
	})

	// Test ListStudentsForGuardian
	t.Run("ListStudentsForGuardian", func(t *testing.T) {
		// Create two students
		dob1 := time.Date(2010, 7, 1, 0, 0, 0, 0, time.UTC)
		student1, err := store.CreateStudent(ctx, schoolID, "A006", sql.NullString{}, "Grace Hopper", dob1, "F", []byte(`{}`))
		require.NoError(t, err)

		dob2 := time.Date(2010, 8, 1, 0, 0, 0, 0, time.UTC)
		student2, err := store.CreateStudent(ctx, schoolID, "A007", sql.NullString{}, "Henry Ford", dob2, "M", []byte(`{}`))
		require.NoError(t, err)

		// Create guardian membership
		var guardianMembershipID uuid.UUID
		require.NoError(t, sqlDB.QueryRowContext(ctx, "INSERT INTO memberships (user_id, school_id, role) VALUES ($1, $2, 'guardian') RETURNING id", uuid.New(), schoolID).Scan(&guardianMembershipID))

		// Link only student1
		_, err = store.CreateGuardianLink(ctx, guardianMembershipID, student1.StudentID, "Parent", true)
		require.NoError(t, err)

		// Guardian should only see student1
		students, err := store.ListStudentsForGuardian(ctx, guardianMembershipID, 10)
		require.NoError(t, err)
		require.Len(t, students, 1)
		require.Equal(t, student1.StudentID, students[0].StudentID)
	})
}
