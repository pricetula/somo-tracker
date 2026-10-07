//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testcontainers"
)

// TestMigrator_SISSchema verifies the SIS schema migration (000004).
// Tests: tables, enum, indexes, constraints, triggers, comments.
func TestMigrator_SISSchema(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	sisTables := []string{
		"countries", "education_systems", "grade_levels",
		"schools", "school_memberships", "students", "guardian_student_links",
	}
	for _, table := range sisTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "table %q should exist", table)
	}

	// --- user_role enum exists with correct values ---
	var enumValues []string
	rows, err := pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'user_role'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		enumValues = append(enumValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"ADMIN", "TEACHER", "GUARDIAN", "FINANCE"}, enumValues)

	// --- Indexes exist ---
	sisIndexes := []string{
		"countries_country_code_idx", "countries_created_at_idx",
		"education_systems_system_name_idx",
		"grade_levels_education_system_id_idx", "grade_levels_country_id_idx", "grade_levels_sys_country_seq_idx",
		"schools_tenant_id_idx", "schools_country_id_idx", "schools_education_system_id_idx",
		"school_memberships_school_id_idx", "school_memberships_user_id_idx", "school_memberships_role_idx",
		"students_school_id_idx", "students_admission_number_idx", "students_full_name_idx",
		"guardian_student_links_school_membership_id_idx", "guardian_student_links_student_id_idx",
	}
	for _, idx := range sisIndexes {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, idx).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "index %q should exist", idx)
	}

	// --- updated_at triggers exist ---
	triggerTables := []string{
		"countries", "education_systems", "grade_levels",
		"schools", "school_memberships", "students", "guardian_student_links",
	}
	for _, table := range triggerTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.triggers
				WHERE event_object_table = $1
				  AND trigger_name = $2 || '_updated_at_trg'
				  AND trigger_schema = 'public'
			)
		`, table, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "trigger %s_updated_at_trg should exist", table)
	}
}

// TestMigrator_SubjectHierarchy verifies the subject hierarchy migration (000005).
// Tests: tables, indexes, triggers.
func TestMigrator_SubjectHierarchy(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	subjectTables := []string{"subjects", "topics", "sub_topics"}
	for _, table := range subjectTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "table %q should exist", table)
	}

	// --- Indexes exist ---
	subjectIndexes := []string{
		"subjects_education_system_id_idx", "subjects_code_idx",
		"topics_subject_id_idx", "topics_subject_seq_idx",
		"sub_topics_topic_id_idx", "sub_topics_topic_seq_idx",
	}
	for _, idx := range subjectIndexes {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, idx).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "index %q should exist", idx)
	}

	// --- updated_at triggers exist ---
	for _, table := range subjectTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.triggers
				WHERE event_object_table = $1
				  AND trigger_name = $2 || '_updated_at_trg'
				  AND trigger_schema = 'public'
			)
		`, table, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "trigger %s_updated_at_trg should exist", table)
	}
}

// TestMigrator_AcademicCalendar verifies the academic calendar migration (000006).
// Tests: tables, indexes, triggers.
func TestMigrator_AcademicCalendar(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	calendarTables := []string{
		"academic_years", "academic_terms", "public_holidays", "school_events",
	}
	for _, table := range calendarTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "table %q should exist", table)
	}

	// --- Indexes exist ---
	calendarIndexes := []string{
		"academic_years_school_id_idx", "academic_years_name_idx",
		"academic_terms_academic_year_id_idx", "academic_terms_name_idx",
		"public_holidays_country_id_idx", "public_holidays_month_day_idx", "public_holidays_country_month_day_idx",
		"school_events_school_id_idx", "school_events_event_type_idx", "school_events_date_range_idx",
	}
	for _, idx := range calendarIndexes {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, idx).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "index %q should exist", idx)
	}

	// --- updated_at triggers exist ---
	for _, table := range calendarTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.triggers
				WHERE event_object_table = $1
				  AND trigger_name = $2 || '_updated_at_trg'
				  AND trigger_schema = 'public'
			)
		`, table, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "trigger %s_updated_at_trg should exist", table)
	}
}

// TestMigrator_ClassRoomsAndEnrollments verifies the class_rooms and
// student_class_enrollments migration (000007). Tests: tables, enum, indexes,
// triggers, RLS policy, unique constraints.
func TestMigrator_ClassRoomsAndEnrollments(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	enrollmentTables := []string{"class_rooms", "student_class_enrollments"}
	for _, table := range enrollmentTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "table %q should exist", table)
	}

	// --- enrollment_status enum exists with correct values ---
	var enumValues []string
	rows, err := pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'enrollment_status'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		enumValues = append(enumValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"ACTIVE", "PROMOTED", "REPEATING", "GRADUATED"}, enumValues)

	// --- Indexes exist ---
	enrollmentIndexes := []string{
		"class_rooms_school_id_idx", "class_rooms_academic_year_id_idx",
		"class_rooms_grade_level_id_idx", "class_rooms_name_idx",
		"student_class_enrollments_school_id_idx", "student_class_enrollments_student_id_idx",
		"student_class_enrollments_class_room_id_idx", "student_class_enrollments_academic_year_id_idx",
		"student_class_enrollments_academic_term_id_idx", "student_class_enrollments_status_idx",
	}
	for _, idx := range enrollmentIndexes {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, idx).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "index %q should exist", idx)
	}

	// --- Unique constraints/indexes exist ---
	uniqueConstraints := []struct {
		table   string
		name    string
		isIndex bool
	}{
		{"class_rooms", "class_rooms_school_year_grade_stream_uniq", true},
		{"student_class_enrollments", "student_class_enrollments_student_year_uniq", false},
		{"student_class_enrollments", "student_class_enrollments_student_term_uniq", true},
	}
	for _, uc := range uniqueConstraints {
		var exists bool
		if uc.isIndex {
			err = pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM pg_indexes
					WHERE schemaname = 'public'
					  AND tablename = $1
					  AND indexname = $2
				)
			`, uc.table, uc.name).Scan(&exists)
		} else {
			err = pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_schema = 'public'
					  AND table_name = $1
					  AND constraint_name = $2
					  AND constraint_type = 'UNIQUE'
				)
			`, uc.table, uc.name).Scan(&exists)
		}
		require.NoError(t, err)
		require.Truef(t, exists, "unique %s %q on %q should exist",
			map[bool]string{true: "index", false: "constraint"}[uc.isIndex], uc.name, uc.table)
	}

	// --- updated_at triggers exist ---
	for _, table := range enrollmentTables {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.triggers
				WHERE event_object_table = $1
				  AND trigger_name = $2 || '_updated_at_trg'
				  AND trigger_schema = 'public'
			)
		`, table, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "trigger %s_updated_at_trg should exist", table)
	}

	// --- RLS is enabled on student_class_enrollments ---
	var rlsEnabled, rlsForced bool
	err = pool.QueryRow(ctx, `
		SELECT relrowsecurity, relforcerowsecurity
		FROM pg_class
		WHERE relname = 'student_class_enrollments'
		  AND relnamespace = 'public'::regnamespace
	`).Scan(&rlsEnabled, &rlsForced)
	require.NoError(t, err)
	require.True(t, rlsEnabled, "RLS should be enabled on student_class_enrollments")
	require.True(t, rlsForced, "RLS should be FORCED on student_class_enrollments")

	// --- RLS policy exists and references app.current_tenant_id ---
	var policyName string
	err = pool.QueryRow(ctx, `
		SELECT policyname FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'student_class_enrollments'
		  AND policyname = 'student_class_enrollments_tenant_isolation'
	`).Scan(&policyName)
	require.NoError(t, err)
	require.Equal(t, "student_class_enrollments_tenant_isolation", policyName,
		"student_class_enrollments_tenant_isolation policy should be installed")

	var policyQual string
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(qual, '') FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'student_class_enrollments'
		  AND policyname = 'student_class_enrollments_tenant_isolation'
	`).Scan(&policyQual)
	require.NoError(t, err)
	require.Contains(t, policyQual, "current_setting",
		"policy qual should reference current_setting()")
	require.Contains(t, policyQual, "app.current_tenant_id",
		"policy qual should reference app.current_tenant_id")
}
