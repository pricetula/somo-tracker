package attendance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestPgRepository_ListRecordsBySlotDate_NoForcedPresent verifies:
//  1. Enrolled students with NO attendance record return status = "" (blank), NOT "PRESENT"
//  2. Students with an existing record return the correct status
//  3. Record IDs are real when marked, empty string when unmarked
func TestPgRepository_ListRecordsBySlotDate_NoForcedPresent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)
	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)

	// Seed academic year + term
	academicYearID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	academicTermID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-03-31', true, false, $5, $5)`,
		academicTermID, tenantID, schoolID, academicYearID, userID)
	require.NoError(t, err)

	// Seed class + learning area + teacher (for the allocation)
	streamID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	classID := uuid.New().String()
	laID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G1', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Early_Years', 'G1')`,
		laID, tenantID, schoolID)
	require.NoError(t, err)

	// Seed timetable track + block + allocation
	trackID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_tracks (id, tenant_id, school_id, academic_year_id, academic_term_id, name, description, is_default, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, 'Main Track', '', true, NOW(), NOW())`,
		trackID, tenantID, schoolID, academicYearID, academicTermID)
	require.NoError(t, err)

	blockID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '09:00', false, 1)`,
		blockID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	allocationID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		allocationID, tenantID, schoolID, blockID, classID, laID, userID)
	require.NoError(t, err)

	// Seed 2 students enrolled in the class
	student1ID := uuid.New().String()
	student2ID := uuid.New().String()
	for i, sid := range []string{student1ID, student2ID} {
		name := []string{"Alice Smith", "Bob Jones"}[i]
		_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, $4, 'F', 'Age_Based')`,
			sid, tenantID, schoolID, name)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New().String(), tenantID, schoolID, sid, academicTermID, academicYearID, classID)
		require.NoError(t, err)
	}

	repo := newRepo(pool)
	dateStr := "2026-09-01"

	// Alice gets a PRESENT record
	_, err = pool.Exec(ctx, `INSERT INTO attendance_records (id, tenant_id, school_id, student_id, timetable_allocation_id, date, status, academic_term_id, marked_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New().String(), tenantID, schoolID, student1ID, allocationID, dateStr, "PRESENT", academicTermID, userID)
	require.NoError(t, err)

	// Bob has NO attendance record on this date

	items, err := repo.ListRecordsBySlotDate(ctx, tenantID, schoolID, allocationID, dateStr)
	require.NoError(t, err)

	alice := findByStudentID(items, student1ID)
	require.NotNil(t, alice, "Alice should appear in results")
	require.Equal(t, AttendanceStatus("PRESENT"), alice.Status, "Alice must be PRESENT")
	require.NotEmpty(t, alice.ID, "Alice's record id must not be empty")

	bob := findByStudentID(items, student2ID)
	require.NotNil(t, bob, "Bob should appear in results")
	require.Equal(t, AttendanceStatus(""), bob.Status, "unmarked student must have blank status, not forced PRESENT")
	require.Empty(t, bob.ID, "unmarked student must have empty record id")
}

func findByStudentID(items []RecordWithEnrichedData, studentID string) *RecordWithEnrichedData {
	for i := range items {
		if items[i].StudentID == studentID {
			return &items[i]
		}
	}
	return nil
}
