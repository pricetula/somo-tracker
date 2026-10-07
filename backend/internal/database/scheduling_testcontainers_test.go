//go:build integration

package database

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testcontainers"
)

// TestMigrator_TimetableScheduling verifies the timetable & scheduling
// migration (000008). Tests: tables, enums, indexes, unique constraints,
// triggers, RLS policies.
func TestMigrator_TimetableScheduling(t *testing.T) {
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
	timetableTables := []string{
		"timetable_templates", "time_slots", "rooms",
		"class_timetable_slots", "timetable_substitutions",
	}
	for _, table := range timetableTables {
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

	// --- Enums exist with correct values ---
	var subStatusValues []string
	rows, err := pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'substitution_status'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		subStatusValues = append(subStatusValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"PENDING", "ASSIGNED", "COMPLETED", "CANCELLED"}, subStatusValues)

	var roomTypeValues []string
	rows, err = pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'room_type'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		roomTypeValues = append(roomTypeValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"STANDARD", "SCIENCE_LAB", "COMPUTER_LAB", "GYM"}, roomTypeValues)

	// --- Indexes exist ---
	timetableIndexes := []string{
		"timetable_templates_school_id_idx", "timetable_templates_name_idx",
		"time_slots_school_id_idx", "time_slots_timetable_template_id_idx", "time_slots_sequence_idx",
		"rooms_school_id_idx", "rooms_room_type_idx", "rooms_capacity_idx",
		"class_timetable_slots_school_id_idx", "class_timetable_slots_class_room_id_idx",
		"class_timetable_slots_academic_term_id_idx", "class_timetable_slots_teacher_membership_id_idx",
		"class_timetable_slots_room_id_idx", "class_timetable_slots_day_time_idx",
		"timetable_substitutions_school_id_idx", "timetable_substitutions_class_timetable_slot_id_idx",
		"timetable_substitutions_substitution_date_idx", "timetable_substitutions_original_teacher_idx",
		"timetable_substitutions_substitute_teacher_idx", "timetable_substitutions_status_idx",
		"timetable_substitutions_date_status_idx",
	}
	for _, idx := range timetableIndexes {
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

	// --- Unique constraints exist ---
	uniqueConstraints := []struct {
		table string
		name  string
	}{
		{"timetable_templates", "timetable_templates_school_name_uniq"},
		{"time_slots", "time_slots_template_seq_uniq"},
		{"time_slots", "time_slots_time_order"},
		{"rooms", "rooms_school_name_uniq"},
		{"class_timetable_slots", "class_timetable_slots_teacher_no_clash"},
		{"class_timetable_slots", "class_timetable_slots_class_day_time_uniq"},
		{"timetable_substitutions", "timetable_substitutions_slot_date_uniq"},
	}
	for _, uc := range uniqueConstraints {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = 'public'
				  AND table_name = $1
				  AND constraint_name = $2
				  AND constraint_type IN ('UNIQUE', 'CHECK')
			)
		`, uc.table, uc.name).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "constraint %q on %q should exist", uc.name, uc.table)
	}

	// --- updated_at triggers exist ---
	for _, table := range timetableTables {
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

	// --- RLS is enabled and FORCED on all timetable tables ---
	for _, table := range timetableTables {
		var rlsEnabled, rlsForced bool
		err = pool.QueryRow(ctx, `
			SELECT relrowsecurity, relforcerowsecurity
			FROM pg_class
			WHERE relname = $1
			  AND relnamespace = 'public'::regnamespace
		`, table).Scan(&rlsEnabled, &rlsForced)
		require.NoError(t, err)
		require.Truef(t, rlsEnabled, "RLS should be enabled on %s", table)
		require.Truef(t, rlsForced, "RLS should be FORCED on %s", table)
	}

	// --- RLS policies exist and reference app.current_tenant_id ---
	policies := []string{
		"timetable_templates_tenant_isolation",
		"time_slots_tenant_isolation",
		"rooms_tenant_isolation",
		"class_timetable_slots_tenant_isolation",
		"timetable_substitutions_tenant_isolation",
	}
	for _, policyName := range policies {
		tableName := strings.TrimSuffix(policyName, "_tenant_isolation")
		var foundName string
		err = pool.QueryRow(ctx, `
			SELECT policyname FROM pg_policies
			WHERE schemaname = 'public' AND tablename = $1
			  AND policyname = $2
		`, tableName, policyName).Scan(&foundName)
		require.NoError(t, err)
		require.Equal(t, policyName, foundName, "policy %q should be installed", policyName)

		var policyQual string
		err = pool.QueryRow(ctx, `
			SELECT COALESCE(qual, '') FROM pg_policies
			WHERE schemaname = 'public' AND tablename = $1
			  AND policyname = $2
		`, tableName, policyName).Scan(&policyQual)
		require.NoError(t, err)
		require.Contains(t, policyQual, "current_setting",
			"policy %q qual should reference current_setting()", policyName)
		require.Contains(t, policyQual, "app.current_tenant_id",
			"policy %q qual should reference app.current_tenant_id", policyName)
	}
}

// TestMigrator_AttendanceTracking verifies the attendance tracking migration
// (000009). Tests: tables, enums, indexes, unique constraints, triggers,
// and RLS policies.
func TestMigrator_AttendanceTracking(t *testing.T) {
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
	attendanceTables := []string{"timetable_attendance", "event_attendance"}
	for _, table := range attendanceTables {
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

	// --- Enums exist with correct values ---
	var ttStatusValues []string
	rows, err := pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'timetable_attendance_status'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		ttStatusValues = append(ttStatusValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"PRESENT", "ABSENT", "LATE", "EXCUSED"}, ttStatusValues)

	var evStatusValues []string
	rows, err = pool.Query(ctx, `
		SELECT enumlabel FROM pg_enum
		WHERE enumtypid = 'event_attendance_status'::regtype
		ORDER BY enumsortorder
	`)
	require.NoError(t, err)
	for rows.Next() {
		var val string
		require.NoError(t, rows.Scan(&val))
		evStatusValues = append(evStatusValues, val)
	}
	rows.Close()
	require.Equal(t, []string{"PRESENT", "ABSENT", "EXCUSED"}, evStatusValues)

	// --- Indexes exist ---
	attendanceIndexes := []string{
		"timetable_attendance_school_id_idx",
		"timetable_attendance_student_id_idx",
		"timetable_attendance_slot_id_idx",
		"timetable_attendance_date_idx",
		"timetable_attendance_recorded_by_idx",
		"timetable_attendance_slot_date_idx",
		"timetable_attendance_student_date_idx",
		"event_attendance_school_id_idx",
		"event_attendance_student_id_idx",
		"event_attendance_event_id_idx",
		"event_attendance_date_idx",
		"event_attendance_student_event_idx",
	}
	for _, idx := range attendanceIndexes {
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

	// --- Unique constraints exist ---
	uniqueConstraints := []struct {
		table string
		name  string
	}{
		{"timetable_attendance", "timetable_attendance_uniq_student_slot_date"},
		{"event_attendance", "event_attendance_uniq_student_event_date"},
	}
	for _, uc := range uniqueConstraints {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = 'public'
				  AND table_name = $1
				  AND constraint_name = $2
				  AND constraint_type = 'UNIQUE'
			)
		`, uc.table, uc.name).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "unique constraint %q on %q should exist", uc.name, uc.table)
	}

	// --- updated_at triggers exist ---
	for _, table := range attendanceTables {
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

	// --- RLS is enabled and FORCED on both attendance tables ---
	for _, table := range attendanceTables {
		var rlsEnabled, rlsForced bool
		err = pool.QueryRow(ctx, `
			SELECT relrowsecurity, relforcerowsecurity
			FROM pg_class
			WHERE relname = $1
			  AND relnamespace = 'public'::regnamespace
		`, table).Scan(&rlsEnabled, &rlsForced)
		require.NoError(t, err)
		require.Truef(t, rlsEnabled, "RLS should be enabled on %s", table)
		require.Truef(t, rlsForced, "RLS should be FORCED on %s", table)
	}

	// --- RLS policies exist and reference app.current_tenant_id ---
	policies := []string{
		"timetable_attendance_tenant_isolation",
		"event_attendance_tenant_isolation",
	}
	for _, policyName := range policies {
		tableName := strings.TrimSuffix(policyName, "_tenant_isolation")
		var foundName string
		err = pool.QueryRow(ctx, `
			SELECT policyname FROM pg_policies
			WHERE schemaname = 'public' AND tablename = $1
			  AND policyname = $2
		`, tableName, policyName).Scan(&foundName)
		require.NoError(t, err)
		require.Equal(t, policyName, foundName, "policy %q should be installed", policyName)

		var policyQual string
		err = pool.QueryRow(ctx, `
			SELECT COALESCE(qual, '') FROM pg_policies
			WHERE schemaname = 'public' AND tablename = $1
			  AND policyname = $2
		`, tableName, policyName).Scan(&policyQual)
		require.NoError(t, err)
		require.Contains(t, policyQual, "current_setting",
			"policy %q qual should reference current_setting()", policyName)
		require.Contains(t, policyQual, "app.current_tenant_id",
			"policy %q qual should reference app.current_tenant_id", policyName)
	}
}
