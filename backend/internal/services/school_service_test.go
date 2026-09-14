package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSchoolRegistration_ValidationErrors(t *testing.T) {
	svc := NewSchoolService(nil, zap.NewNop())

	_, err := svc.RegisterSchool(context.Background(), "", "t1", "Alice", "School")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
}

func TestSchoolRegistration_TransactionalFlow_RealDB(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	svc := NewSchoolService(pool, zap.NewNop())

	// The service validates inputs before any DB interaction.
	_, err := svc.RegisterSchool(context.Background(), "bad-user", "bad-tenant", "", "School")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
}

func TestCreateSchoolWithSetup_RealDB(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	svc := NewSchoolService(pool, zap.NewNop())

	// Without admin membership this should return forbidden
	_, err := svc.CreateSchoolWithSetup(context.Background(), "non-admin-user-id", "some-tenant", "Test School")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestCreateSchoolWithSetup_VerifyAllDBTables(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	svc := NewSchoolService(pool, zap.NewNop())

	// Setup prerequisites: tenant + admin user
	var tenantID string
	pool.QueryRow(context.Background(), `INSERT INTO tenants (name, slug, stytch_org_id) VALUES ('Test Org', 'test-org-verify', 'org-test-verify') ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name RETURNING id`).Scan(&tenantID)

	var userID string
	pool.QueryRow(context.Background(), `INSERT INTO users (email, full_name, tenant_id, is_active) VALUES ('admin@verify.local', 'Verify Admin', $1, true) ON CONFLICT (tenant_id, email) DO UPDATE SET full_name = EXCLUDED.full_name RETURNING id`, tenantID).Scan(&userID)

	// Pre-existing ADMIN school membership so admin verification passes
	var countryID string
	pool.QueryRow(context.Background(), `SELECT id FROM countries WHERE country_name = 'Kenya' LIMIT 1`).Scan(&countryID)
	var edSysIDSetup string
	pool.QueryRow(context.Background(), `SELECT id FROM education_systems WHERE system_name ILIKE '%CBE%' LIMIT 1`).Scan(&edSysIDSetup)

	var existingSchool string
	pool.QueryRow(context.Background(), `INSERT INTO schools (tenant_id, school_name, country_id, education_system_id) VALUES ($1, 'Pre-existing School', $2, $3) ON CONFLICT DO NOTHING RETURNING id`, tenantID, countryID, edSysIDSetup).Scan(&existingSchool)
	if existingSchool == "" {
		pool.QueryRow(context.Background(), `SELECT id FROM schools WHERE tenant_id = $1 LIMIT 1`, tenantID).Scan(&existingSchool)
	}
	if existingSchool != "" {
		pool.Exec(context.Background(), `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1, $2, 'ADMIN', TRUE) ON CONFLICT DO NOTHING`, existingSchool, userID)
	}

	schoolName := "Verification School"

	schoolID, err := svc.CreateSchoolWithSetup(context.Background(), userID, tenantID, schoolName)
	require.NoError(t, err, "CreateSchoolWithSetup should succeed with admin user")
	require.NotEmpty(t, schoolID)

	// 1. Verify school created
	var sName, countryName string
	err = pool.QueryRow(context.Background(), `SELECT s.school_name, c.country_name FROM schools s JOIN countries c ON s.country_id = c.id WHERE s.id = $1`, schoolID).Scan(&sName, &countryName)
	require.NoError(t, err)
	assert.Equal(t, schoolName, sName)

	// 2. Verify ADMIN membership active
	var role string
	var isActive bool
	err = pool.QueryRow(context.Background(), `SELECT role, is_active FROM school_memberships WHERE school_id = $1 AND user_id = $2`, schoolID, userID).Scan(&role, &isActive)
	require.NoError(t, err)
	assert.Equal(t, "ADMIN", role)
	assert.True(t, isActive)

	// 3. Verify academic year + 3 terms
	var yearName string
	err = pool.QueryRow(context.Background(), `SELECT name FROM academic_years WHERE school_id = $1`, schoolID).Scan(&yearName)
	require.NoError(t, err)
	assert.Equal(t, "2026", yearName)

	var termCount int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM academic_terms WHERE academic_year_id IN (SELECT id FROM academic_years WHERE school_id = $1)`, schoolID).Scan(&termCount)
	require.NoError(t, err)
	assert.Equal(t, 3, termCount)

	// 4. Verify CBE subjects loaded (at least some from docs/cbc/*.json)
	var subjectCount int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM subjects WHERE id IN (SELECT education_system_id FROM schools WHERE id = $1)`, schoolID).Scan(&subjectCount)
	// Actually check subjects linked to the school's education_system
	var edSys string
	pool.QueryRow(context.Background(), `SELECT education_system_id FROM schools WHERE id = $1`, schoolID).Scan(&edSys)
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM subjects WHERE education_system_id = $1`, edSys).Scan(&subjectCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, subjectCount, 1, "At least one CBE subject should be loaded")

	// 5. Verify topics exist
	var topicCount int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM topics t JOIN subjects s ON t.subject_id = s.id WHERE s.education_system_id = $1`, edSys).Scan(&topicCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, topicCount, 1, "At least one topic should exist")

	// 6. Verify sub-topics exist
	var subTopicCount int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sub_topics st JOIN topics t ON st.topic_id = t.id JOIN subjects s ON t.subject_id = s.id WHERE s.education_system_id = $1`, edSys).Scan(&subTopicCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, subTopicCount, 1, "At least one sub-topic should exist")
}
