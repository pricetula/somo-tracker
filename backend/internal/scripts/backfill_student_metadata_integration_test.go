//go:build integration

package scripts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/testcontainers"
)

// TestBackfillStudentMetadata validates that metadata JSON can be backfilled into structured columns.
func TestBackfillStudentMetadata(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err)

	// Create minimal school and student with metadata
	schoolID := uuid.New()
	studentID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO schools (id, tenant_id, name) VALUES ($1, $2, $3)`, schoolID, uuid.New(), "Test School")
	require.NoError(t, err)

	metadata := map[string]string{
		"address":                 "123 Main St",
		"phone":                   "+254700000000",
		"email":                   "student@example.com",
		"nationality":             "Kenyan",
		"emergency_contact_name":  "Jane Doe",
		"emergency_contact_phone": "+254711111111",
		"special_needs":           "None",
	}
	metaBytes, _ := json.Marshal(metadata)

	_, err = pool.Exec(ctx, `
		INSERT INTO students (student_id, school_id, admission_number, full_name, date_of_birth, gender, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, studentID, schoolID, "2026-001", "Test Student", "2010-01-01", "M", metaBytes)
	require.NoError(t, err)

	// Simulate backfill logic: copy from metadata JSON to columns
	// This mirrors backend/scripts/backfill_student_metadata.go
	rows, err := pool.Query(ctx, `SELECT student_id, metadata FROM students WHERE metadata IS NOT NULL`)
	require.NoError(t, err)
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var sid string
		var meta []byte
		require.NoError(t, rows.Scan(&sid, &meta))
		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(meta, &data))

		address, _ := data["address"].(string)
		phone, _ := data["phone"].(string)
		email, _ := data["email"].(string)
		nationality, _ := data["nationality"].(string)
		ecName, _ := data["emergency_contact_name"].(string)
		ecPhone, _ := data["emergency_contact_phone"].(string)
		specialNeeds, _ := data["special_needs"].(string)

		_, err = pool.Exec(ctx, `
			UPDATE students SET
				address = COALESCE(NULLIF($1, ''), address),
				phone = COALESCE(NULLIF($2, ''), phone),
				email = COALESCE(NULLIF($3, ''), email),
				nationality = COALESCE(NULLIF($4, ''), nationality),
				emergency_contact_name = COALESCE(NULLIF($5, ''), emergency_contact_name),
				emergency_contact_phone = COALESCE(NULLIF($6, ''), emergency_contact_phone),
				special_needs = COALESCE(NULLIF($7, ''), special_needs)
			WHERE student_id = $8
		`, address, phone, email, nationality, ecName, ecPhone, specialNeeds, sid)
		require.NoError(t, err)
		updated++
	}
	require.Equal(t, 1, updated)

	// Verify columns populated
	var addr, phone, email, nat, ecName, ecPhone, sn string
	err = pool.QueryRow(ctx, `
		SELECT address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs
		FROM students WHERE student_id = $1
	`, studentID).Scan(&addr, &phone, &email, &nat, &ecName, &ecPhone, &sn)
	require.NoError(t, err)
	require.Equal(t, "123 Main St", addr)
	require.Equal(t, "+254700000000", phone)
	require.Equal(t, "student@example.com", email)
	require.Equal(t, "Kenyan", nat)
	require.Equal(t, "Jane Doe", ecName)
	require.Equal(t, "+254711111111", ecPhone)
	require.Equal(t, "None", sn)
}
