package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// SchoolRegistrationService handles the school registration flow:
// 1. Update user's full_name
// 2. Create new school with default country and education system
// 3. Create school membership with role=ADMIN
type SchoolRegistrationService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewSchoolRegistrationService(pool *pgxpool.Pool, logger *zap.Logger) SchoolRegistrationService {
	return SchoolRegistrationService{
		pool:   pool,
		logger: logger.With(zap.String("service", "school_registration")),
	}
}

// RegisterSchool performs the atomic registration:
// 1. Update user's full_name
// 2. Create new school with default country and education system
// 3. Create school membership with role=ADMIN
func (s *SchoolRegistrationService) RegisterSchool(
	ctx context.Context,
	userID, tenantID, userName, schoolName string,
) (schoolID string, err error) {
	if userID == "" {
		return "", errors.New("bad_request: user_id is required")
	}
	if tenantID == "" {
		return "", errors.New("bad_request: tenant_id is required")
	}
	if schoolName == "" {
		return "", errors.New("bad_request: school_name is required")
	}
	if userName == "" {
		return "", errors.New("bad_request: user_name is required")
	}

	txErr := database.WithTenantTx(ctx, s.pool, s.logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		// Step 1: Update user's full_name
		updateSQL := `UPDATE users SET full_name = $1 WHERE id = $2`
		if _, err := tx.Exec(ctx, updateSQL, userName, userID); err != nil {
			return errors.New("internal_error: failed to update user full_name")
		}

		// Step 2: Create new school — default Kenya / CBE
		var countryID, educationSystemID string
		if err := tx.QueryRow(ctx, `SELECT id FROM countries WHERE country_name = 'Kenya' LIMIT 1`).Scan(&countryID); err != nil {
			return errors.New("internal_error: Kenya country not found")
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM education_systems WHERE country_id = $1 AND system_name ILIKE '%CBE%' LIMIT 1`, countryID).Scan(&educationSystemID); err != nil {
			return errors.New("internal_error: CBE education system for Kenya not found")
		}

		// Insert school
		insertSchoolSQL := `INSERT INTO schools (tenant_id, school_name, country_id, education_system_id)
			VALUES ($1, $2, $3, $4) RETURNING id`
		if err := tx.QueryRow(ctx, insertSchoolSQL, tenantID, schoolName, countryID, educationSystemID).
			Scan(&schoolID); err != nil {
			return errors.New("internal_error: failed to create school")
		}

		// Step 3: Create school membership with role=ADMIN
		insertMembershipSQL := `INSERT INTO school_memberships (school_id, user_id, role, is_active)
			VALUES ($1, $2, 'ADMIN', TRUE)`
		if _, err := tx.Exec(ctx, insertMembershipSQL, schoolID, userID); err != nil {
			return errors.New("internal_error: failed to create school membership")
		}

		return nil
	})

	if txErr != nil {
		return "", txErr
	}

	return schoolID, nil
}
