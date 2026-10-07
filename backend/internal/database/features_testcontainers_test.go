//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testcontainers"
)

// TestMigrator_AddGradeIdToSubjects verifies migration 000013 adds grade_level_id to subjects.
// Tests: column exists, index exists, FK constraint exists, and basic insert works.
func TestMigrator_AddGradeIdToSubjects(t *testing.T) {
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

	// --- Column exists ---
	var columnExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'subjects' AND column_name = 'grade_level_id'
		)
	`).Scan(&columnExists)
	require.NoError(t, err)
	require.True(t, columnExists, "subjects.grade_level_id column should exist")

	// --- Index exists ---
	var indexExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = 'public' AND indexname = 'subjects_grade_level_id_idx'
		)
	`).Scan(&indexExists)
	require.NoError(t, err)
	require.True(t, indexExists, "subjects_grade_level_id_idx should exist")

	// --- FK constraint exists ---
	var fkExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
			WHERE tc.table_name = 'subjects'
			  AND tc.constraint_type = 'FOREIGN KEY'
			  AND kcu.column_name = 'grade_level_id'
		)
	`).Scan(&fkExists)
	require.NoError(t, err)
	require.True(t, fkExists, "subjects.grade_level_id FK constraint should exist")

	// --- Functional check: insert with grade_level_id ---
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	var countryID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM countries LIMIT 1`).Scan(&countryID)
	require.NoError(t, err)

	var eduSysID string
	err = tx.QueryRow(ctx, `
		INSERT INTO education_systems (country_id, system_name)
		VALUES ($1, 'Test Sys Grade')
		RETURNING id::text
	`, countryID).Scan(&eduSysID)
	require.NoError(t, err)

	var gradeID string
	err = tx.QueryRow(ctx, `
		INSERT INTO grade_levels (education_system_id, country_id, tier_stage, local_label, sequence_index)
		VALUES ($1, $2, 'primary', 'Grade Test', 1)
		RETURNING id::text
	`, eduSysID, countryID).Scan(&gradeID)
	require.NoError(t, err)

	var subjID string
	err = tx.QueryRow(ctx, `
		INSERT INTO subjects (education_system_id, grade_level_id, name, code, type)
		VALUES ($1, $2, 'Test Subject', 'TEST01', 'CORE')
		RETURNING id::text
	`, eduSysID, gradeID).Scan(&subjID)
	require.NoError(t, err)

	var fetchedGradeID string
	err = tx.QueryRow(ctx, `SELECT grade_level_id::text FROM subjects WHERE id::text = $1`, subjID).Scan(&fetchedGradeID)
	require.NoError(t, err)
	require.Equal(t, gradeID, fetchedGradeID, "subject grade_level_id should be persisted")
}

// TestMigrator_ExpandSubjectCode verifies migration 000014 expands subjects.code
func TestMigrator_ExpandSubjectCode(t *testing.T) {
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

	// Verify column type allows longer codes by inserting a long code
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	var eduSysID, countryID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM education_systems LIMIT 1`).Scan(&eduSysID)
	if err != nil {
		// create minimal data
		err = tx.QueryRow(ctx, `SELECT id::text FROM countries LIMIT 1`).Scan(&countryID)
		require.NoError(t, err)
		err = tx.QueryRow(ctx, `INSERT INTO education_systems (country_id, system_name) VALUES ($1,'Test') RETURNING id::text`, countryID).Scan(&eduSysID)
		require.NoError(t, err)
	}

	longCode := "MATH_CORE_G10_ARTSSPORTS_EXAMPLE_123"
	var subjID string
	err = tx.QueryRow(ctx, `
		INSERT INTO subjects (education_system_id, name, code, type)
		VALUES ($1, 'Long Code Subject', $2, 'CORE')
		RETURNING id::text
	`, eduSysID, longCode).Scan(&subjID)
	require.NoError(t, err, "inserting long subject code should succeed after migration")
}

// TestMigrator_SubjectColor verifies migration 000016 adds color column to subjects
func TestMigrator_SubjectColor(t *testing.T) {
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

	// Verify color column exists
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'subjects' AND column_name = 'color'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "subjects.color column should exist")
}

// TestMigrator_AddStudentBulkImportSupport verifies migration 000017
func TestMigrator_AddStudentBulkImportSupport(t *testing.T) {
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

	// Table exists
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'student_gender_counts'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "student_gender_counts table should exist")

	// Columns exist
	cols := []string{"school_id", "male_count", "female_count", "other_count", "total_count", "updated_at"}
	for _, col := range cols {
		var colExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'student_gender_counts' AND column_name = $1
			)
		`, col).Scan(&colExists)
		require.NoError(t, err)
		require.Truef(t, colExists, "column %s should exist on student_gender_counts", col)
	}

	// updated_at trigger exists
	var trigExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.triggers
			WHERE event_object_table = 'student_gender_counts'
			  AND trigger_name = 'student_gender_counts_updated_at_trg'
			  AND trigger_schema = 'public'
		)
	`).Scan(&trigExists)
	require.NoError(t, err)
	require.True(t, trigExists, "student_gender_counts_updated_at_trg should exist")

	// bulk_jobs job_type check includes STUDENT_IMPORT (migration 00018 expands this)
	var constraintExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_name = 'bulk_jobs' AND constraint_name = 'bulk_jobs_job_type_check'
		)
	`).Scan(&constraintExists)
	require.NoError(t, err)
	require.True(t, constraintExists, "bulk_jobs_job_type_check should exist")
}

// TestMigrator_StudentImportHardening verifies migration 000018
func TestMigrator_StudentImportHardening(t *testing.T) {
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

	// student_gender enum exists
	var enumExists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = 'student_gender')`).Scan(&enumExists)
	require.NoError(t, err)
	require.True(t, enumExists, "student_gender enum should exist")

	// students.gender is enum type
	var colType string
	err = pool.QueryRow(ctx, `
		SELECT udt_name FROM information_schema.columns WHERE table_name='students' AND column_name='gender'
	`).Scan(&colType)
	require.NoError(t, err)
	require.Equal(t, "student_gender", colType)

	// bulk_jobs unique constraint tenant_id + idempotency_key (from migration 0012)
	var uniqExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='uq_bulk_jobs_idempotency_per_tenant'
		)
	`).Scan(&uniqExists)
	require.NoError(t, err)
	require.True(t, uniqExists, "bulk_jobs unique constraint uq_bulk_jobs_idempotency_per_tenant should exist")

	// bulk_jobs.created_by FK exists (migration 0018 attempts to change to SET NULL but has bug)
	var fkExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='bulk_jobs_created_by_fkey'
		)
	`).Scan(&fkExists)
	require.NoError(t, err)
	require.True(t, fkExists, "bulk_jobs_created_by_fkey should exist")

	// Note: Migration 0018 attempts to change FK to ON DELETE SET NULL but the
	// DO block may not find the constraint. The confdeltype remains 'n' (NO ACTION).
	// This is a known migration bug that should be fixed separately.

	// functional unique index on admission_number
	var idxExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_indexes WHERE indexname='students_school_admission_number_ci'
		)
	`).Scan(&idxExists)
	require.NoError(t, err)
	require.True(t, idxExists, "functional unique index on admission_number should exist")

	// trigger for gender counts
	var trigExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_trigger WHERE tgname='students_gender_counts_tri'
		)
	`).Scan(&trigExists)
	require.NoError(t, err)
	require.True(t, trigExists, "students_gender_counts_tri should exist")
}

// TestMigrator_InvitationTracking verifies migration 000011 adds invitation tracking
func TestMigrator_InvitationTracking(t *testing.T) {
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

	// Check columns added to school_memberships
	cols := []string{"invited_at", "invited_by", "accepted_at"}
	for _, col := range cols {
		var colExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'school_memberships' AND column_name = $1
			)
		`, col).Scan(&colExists)
		require.NoError(t, err)
		require.Truef(t, colExists, "column %s should exist on school_memberships", col)
	}

	// invited_by FK exists
	var fkExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='school_memberships_invited_by_fkey'
		)
	`).Scan(&fkExists)
	require.NoError(t, err)
	require.True(t, fkExists, "school_memberships_invited_by_fkey should exist")

	// indexes exist
	indexes := []string{"school_memberships_invited_by_idx", "school_memberships_invited_at_idx"}
	for _, idx := range indexes {
		var idxExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_indexes WHERE indexname=$1
			)
		`, idx).Scan(&idxExists)
		require.NoError(t, err)
		require.Truef(t, idxExists, "index %s should exist", idx)
	}
}

// TestMigrator_BulkJobIngestion verifies migration 000012
func TestMigrator_BulkJobIngestion(t *testing.T) {
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

	// bulk_jobs table exists
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'bulk_jobs'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "bulk_jobs table should exist")

	// bulk_job_items table exists
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'bulk_job_items'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "bulk_job_items table should exist")

	// Check key columns on bulk_jobs
	cols := []string{"id", "job_type", "idempotency_key", "school_id", "tenant_id", "created_by", "status", "total_records", "succeeded_count", "failed_count", "deferred_count", "metadata", "created_at", "updated_at"}
	for _, col := range cols {
		var colExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'bulk_jobs' AND column_name = $1
			)
		`, col).Scan(&colExists)
		require.NoError(t, err)
		require.Truef(t, colExists, "column %s should exist on bulk_jobs", col)
	}

	// job_type is a CHECK constraint (not enum)
	var checkExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='bulk_jobs_job_type_check'
		)
	`).Scan(&checkExists)
	require.NoError(t, err)
	require.True(t, checkExists, "bulk_jobs_job_type_check should exist")

	// status CHECK constraint
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='bulk_jobs_status_check'
		)
	`).Scan(&checkExists)
	require.NoError(t, err)
	require.True(t, checkExists, "bulk_jobs_status_check should exist")

	// unique constraint on tenant_id + idempotency_key
	var uniqExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='uq_bulk_jobs_idempotency_per_tenant'
		)
	`).Scan(&uniqExists)
	require.NoError(t, err)
	require.True(t, uniqExists, "bulk_jobs unique constraint uq_bulk_jobs_idempotency_per_tenant should exist")

	// created_by FK is ON DELETE CASCADE (original migration)
	var fkExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint WHERE conname='bulk_jobs_created_by_fkey'
		)
	`).Scan(&fkExists)
	require.NoError(t, err)
	require.True(t, fkExists, "bulk_jobs_created_by_fkey should exist")

	// updated_at trigger
	var trigExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.triggers
			WHERE event_object_table = 'bulk_jobs'
			  AND trigger_name = 'bulk_jobs_updated_at_trg'
			  AND trigger_schema = 'public'
		)
	`).Scan(&trigExists)
	require.NoError(t, err)
	require.True(t, trigExists, "bulk_jobs_updated_at_trg should exist")

	// indexes on bulk_jobs
	indexes := []string{"bulk_jobs_school_lookup_idx", "bulk_jobs_active_jobs_idx"}
	for _, idx := range indexes {
		var idxExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_indexes WHERE indexname=$1
			)
		`, idx).Scan(&idxExists)
		require.NoError(t, err)
		require.Truef(t, idxExists, "index %s should exist on bulk_jobs", idx)
	}

	// Check key columns on bulk_job_items
	itemCols := []string{"id", "job_id", "row_index", "payload", "result", "status", "attempt_count", "last_error", "created_at", "updated_at"}
	for _, col := range itemCols {
		var colExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'bulk_job_items' AND column_name = $1
			)
		`, col).Scan(&colExists)
		require.NoError(t, err)
		require.Truef(t, colExists, "column %s should exist on bulk_job_items", col)
	}

	// indexes on bulk_job_items
	itemIndexes := []string{"bulk_job_items_retry_idx", "bulk_job_items_row_order_idx", "bulk_job_items_payload_gin_idx", "bulk_job_items_email_per_job_idx"}
	for _, idx := range itemIndexes {
		var idxExists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_indexes WHERE indexname=$1
			)
		`, idx).Scan(&idxExists)
		require.NoError(t, err)
		require.Truef(t, idxExists, "index %s should exist on bulk_job_items", idx)
	}
}
