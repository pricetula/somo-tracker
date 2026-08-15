package members

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Integration Tests: member_counts table + triggers
// ============================================================================
//
// These tests verify the database triggers that maintain the single-row
// member_counts table stay in sync with cbc_students and memberships:
//
//   trg_student_count     — INSERT / UPDATE (is_active) / DELETE on cbc_students
//   trg_membership_count  — INSERT / role change / activation toggle / DELETE
//                           on memberships
//
// The repository method GetMemberCounts is exercised as the read path so the
// full contract (write via trigger → read via repo) is covered.

func TestMemberCounts_TriggersStayInSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	// 000001 provides cbc_students/memberships (and their FKs); 000003 adds
	// the member_counts table + triggers.
	applyMigration(t, pool, "000001_initial_schema.up.sql")
	applyMigration(t, pool, "000003_create_member_counts.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	read := func(t *testing.T) *MemberCounts {
		t.Helper()
		counts, err := repo.GetMemberCounts(ctx, tenantID, schoolID)
		require.NoError(t, err)
		return counts
	}

	// Seed data must leave counts at zero (fresh-DB assumption).
	require.Equal(t, &MemberCounts{}, read(t), "fresh DB should start at zero counts")

	// ── Students ──────────────────────────────────────────────────────────
	insertStudent := func(active bool) string {
		studentID := uuid.New().String()
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, is_active)
			VALUES ($1, $2, $3, $4, 'M', $5)`,
			studentID, tenantID, schoolID, "Test Student", active)
		require.NoError(t, err)
		return studentID
	}

	studentID := insertStudent(true)
	require.Equal(t, 1, read(t).Students, "active student should be counted")

	inactiveID := insertStudent(false)
	require.Equal(t, 1, read(t).Students, "inactive student must not be counted")

	// Deactivate the active student → count drops to zero.
	_, err := pool.Exec(ctx, `UPDATE cbc_students SET is_active = false WHERE id = $1`, studentID)
	require.NoError(t, err)
	require.Equal(t, 0, read(t).Students, "deactivating a student should decrement the count")

	// Reactivate → back to one.
	_, err = pool.Exec(ctx, `UPDATE cbc_students SET is_active = true WHERE id = $1`, studentID)
	require.NoError(t, err)
	require.Equal(t, 1, read(t).Students, "reactivating a student should increment the count")

	// Hard delete of an active student → count drops.
	_, err = pool.Exec(ctx, `DELETE FROM cbc_students WHERE id = $1`, studentID)
	require.NoError(t, err)
	require.Equal(t, 0, read(t).Students, "deleting an active student should decrement the count")

	// Delete of an inactive student is a no-op for the counter.
	_, err = pool.Exec(ctx, `DELETE FROM cbc_students WHERE id = $1`, inactiveID)
	require.NoError(t, err)
	require.Equal(t, 0, read(t).Students, "deleting an inactive student must not change the count")

	// ── Memberships ───────────────────────────────────────────────────────
	insertMembership := func(role string) {
		_, err := pool.Exec(ctx, `
			INSERT INTO memberships (user_id, tenant_id, school_id, role, is_active)
			VALUES ($1, $2, $3, $4, true)`,
			userID, tenantID, schoolID, role)
		require.NoError(t, err)
	}

	insertMembership("TEACHER")
	counts := read(t)
	require.Equal(t, 1, counts.Teachers, "inserting a TEACHER membership should count it")

	// Role change while remaining active — the case that regressed in the
	// migration before it was fixed.
	_, err = pool.Exec(ctx, `UPDATE memberships SET role = 'NURSE' WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID)
	require.NoError(t, err)
	counts = read(t)
	require.Equal(t, 0, counts.Teachers, "old role must be decremented on role change")
	require.Equal(t, 1, counts.Nurses, "new role must be incremented on role change")

	// Deactivate → nurse count drops.
	_, err = pool.Exec(ctx, `UPDATE memberships SET is_active = false WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID)
	require.NoError(t, err)
	require.Equal(t, 0, read(t).Nurses, "deactivating a membership should decrement its role count")

	// Reactivate → back to one.
	_, err = pool.Exec(ctx, `UPDATE memberships SET is_active = true WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID)
	require.NoError(t, err)
	require.Equal(t, 1, read(t).Nurses, "reactivating a membership should increment its role count")

	// Hard delete → count drops.
	_, err = pool.Exec(ctx, `DELETE FROM memberships WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID)
	require.NoError(t, err)
	require.Equal(t, 0, read(t).Nurses, "deleting an active membership should decrement its role count")

	// ── All six roles counted independently ───────────────────────────────
	roles := []string{"SCHOOL_ADMIN", "NURSE", "TEACHER", "PARENT", "FINANCE"}
	for i, role := range roles {
		uid := uuid.New().String()
		_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
			uid, "role-"+role+"@test.com", tenantID, "Role User "+role)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO memberships (user_id, tenant_id, school_id, role, is_active)
			VALUES ($1, $2, $3, $4, true)`,
			uid, tenantID, schoolID, role)
		require.NoError(t, err)
		counts := read(t)
		require.Equal(t, i+1, counts.Admins+counts.Nurses+counts.Teachers+counts.Parents+counts.Finance,
			"each role membership should add to the aggregate")
	}
}
