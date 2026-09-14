package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
			return fmt.Errorf("internal_error: failed to update user full_name: %w", err)
		}

		// Step 2: Create new school — default Kenya / CBE
		var countryID, educationSystemID string
		if err := tx.QueryRow(ctx, `SELECT id FROM countries WHERE country_name = 'Kenya' LIMIT 1`).Scan(&countryID); err != nil {
			return fmt.Errorf("internal_error: Kenya country not found: %w", err)
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM education_systems WHERE country_id = $1 AND system_name ILIKE '%CBE%' LIMIT 1`, countryID).Scan(&educationSystemID); err != nil {
			return fmt.Errorf("internal_error: CBE education system for Kenya not found: %w", err)
		}

		// Insert school
		insertSchoolSQL := `INSERT INTO schools (tenant_id, school_name, country_id, education_system_id)
			VALUES ($1, $2, $3, $4) RETURNING id`
		if err := tx.QueryRow(ctx, insertSchoolSQL, tenantID, schoolName, countryID, educationSystemID).
			Scan(&schoolID); err != nil {
			return fmt.Errorf("internal_error: failed to create school: %w", err)
		}

		// Step 3: Create school membership with role=ADMIN
		insertMembershipSQL := `INSERT INTO school_memberships (school_id, user_id, role, is_active)
			VALUES ($1, $2, 'ADMIN', TRUE)`
		if _, err := tx.Exec(ctx, insertMembershipSQL, schoolID, userID); err != nil {
			return fmt.Errorf("internal_error: failed to create school membership: %w", err)
		}

		return nil
	})

	if txErr != nil {
		return "", txErr
	}

	return schoolID, nil
}

// CreateSchoolWithSetup creates a new school with admin check, academic periods, and CBE curriculum.
func (s *SchoolRegistrationService) CreateSchoolWithSetup(
	ctx context.Context,
	userID, tenantID, schoolName string,
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

	// 1. Verify user is admin (any active ADMIN membership in tenant)
	var adminCheck int
	checkErr := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM school_memberships sm JOIN users u ON sm.user_id = u.id WHERE u.tenant_id = $1 AND sm.user_id = $2 AND sm.role = 'ADMIN' AND sm.is_active = true`, tenantID, userID).Scan(&adminCheck)
	if checkErr != nil {
		return "", fmt.Errorf("internal_error: admin verification failed: %w", checkErr)
	}
	if adminCheck == 0 {
		return "", errors.New("forbidden: user must be admin to create school")
	}

	txErr := database.WithTenantTx(ctx, s.pool, s.logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		// Single CTE: get user's active school education system + country, or fall back to Kenyan CBE
		var countryID, educationSystemID string
		cteSQL := `
		WITH active_school AS (
			SELECT s.id AS sid, s.education_system_id AS esid, s.country_id AS cid
			FROM school_memberships sm
			JOIN schools s ON sm.school_id = s.id
			WHERE sm.user_id = $1 AND sm.is_active = TRUE
			LIMIT 1
		),
		default_cbe AS (
			SELECT c.id AS cid, es.id AS esid
			FROM countries c
			JOIN education_systems es ON es.country_id = c.id
			WHERE c.country_name = 'Kenya' AND es.system_name ILIKE '%CBE%'
			LIMIT 1
		)
		SELECT COALESCE(a.esid, d.esid) AS education_system_id,
			   COALESCE(a.cid, d.cid) AS country_id
		FROM active_school a
		FULL OUTER JOIN default_cbe d ON TRUE`
		if err := tx.QueryRow(ctx, cteSQL, userID).Scan(&educationSystemID, &countryID); err != nil {
			return fmt.Errorf("internal_error: failed to resolve education system/country: %w", err)
		}
		if educationSystemID == "" || countryID == "" {
			return fmt.Errorf("internal_error: education system or country not found")
		}

		// Insert school
		insertSQL := `INSERT INTO schools (tenant_id, school_name, country_id, education_system_id) VALUES ($1, $2, $3, $4) RETURNING id`
		if err := tx.QueryRow(ctx, insertSQL, tenantID, schoolName, countryID, educationSystemID).Scan(&schoolID); err != nil {
			return fmt.Errorf("internal_error: failed to create school: %w", err)
		}

		// Create ADMIN membership
		if _, err := tx.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1, $2, 'ADMIN', TRUE)`, schoolID, userID); err != nil {
			return fmt.Errorf("internal_error: failed to create school membership: %w", err)
		}

		// Create academic year + 3 terms for current year
		yearName := fmt.Sprintf("%d", 2026) // hard-code current year
		var yearID string
		if err := tx.QueryRow(ctx, `INSERT INTO academic_years (school_id, name, start_date, end_date) VALUES ($1, $2, $3, $4) RETURNING id`, schoolID, yearName, "2026-01-01", "2026-12-31").Scan(&yearID); err != nil {
			return fmt.Errorf("internal_error: failed to create academic year: %w", err)
		}

		// 3 terms — bulk insert
		_, err = tx.Exec(ctx, `
			INSERT INTO academic_terms (academic_year_id, name, start_date, end_date) VALUES
			($1, 'Term 1', '2026-01-05', '2026-05-21'),
			($1, 'Term 2', '2026-05-25', '2026-09-11'),
			($1, 'Term 3', '2026-09-15', '2026-12-19')
		`, yearID)
		if err != nil {
			return fmt.Errorf("internal_error: failed to create academic terms bulk: %w", err)
		}

		// Bulk insert CBE curriculum using Go-generated UUIDs (3 DB round-trips max)
		type subjRow struct {
			id   string
			es   string
			name string
			code string
		}
		type topicRow struct {
			id   string
			sid  string
			name string
			seq  int
		}
		type subRow struct {
			id   string
			tid  string
			name string
			seq  int
		}
		subjectsBulk := make([]subjRow, 0)
		topicsBulk := make([]topicRow, 0)
		subsBulk := make([]subRow, 0)
		files, _ := filepath.Glob("../../docs/cbc/*.json")
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			var subjects []map[string]interface{}
			if err := json.Unmarshal(data, &subjects); err != nil {
				continue
			}
			for _, sub := range subjects {
				name, _ := sub["name"].(string)
				code, _ := sub["code"].(string)
				if name == "" || code == "" {
					continue
				}
				sid := uuid.New().String()
				subjectsBulk = append(subjectsBulk, subjRow{id: sid, es: educationSystemID, name: name, code: code})
				strands, ok := sub["strands"].([]interface{})
				if !ok {
					continue
				}
				for idx, s := range strands {
					strand, ok := s.(map[string]interface{})
					if !ok {
						continue
					}
					sName, _ := strand["name"].(string)
					if sName == "" {
						continue
					}
					tid := uuid.New().String()
					topicsBulk = append(topicsBulk, topicRow{id: tid, sid: sid, name: sName, seq: idx + 1})
					subStrands, ok := strand["sub_strands"].([]interface{})
					if !ok {
						continue
					}
					for sIdx, ss := range subStrands {
						sSub, ok := ss.(map[string]interface{})
						if !ok {
							continue
						}
						subName, _ := sSub["name"].(string)
						if subName == "" {
							continue
						}
						ssId := uuid.New().String()
						subsBulk = append(subsBulk, subRow{id: ssId, tid: tid, name: subName, seq: sIdx + 1})
					}
				}
			}
		}
		if len(subjectsBulk) > 0 {
			values := make([]string, 0, len(subjectsBulk))
			args := make([]interface{}, 0, len(subjectsBulk)*4)
			argIdx := 1
			for _, r := range subjectsBulk {
				values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,'CORE')", argIdx, argIdx+1, argIdx+2, argIdx+3))
				args = append(args, r.id, r.es, r.name, r.code)
				argIdx += 4
			}
			sql := fmt.Sprintf(`INSERT INTO subjects (id, education_system_id, name, code, type) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				return fmt.Errorf("internal_error: bulk insert subjects failed: %w", err)
			}
		}
		if len(topicsBulk) > 0 {
			values := make([]string, 0, len(topicsBulk))
			args := make([]interface{}, 0, len(topicsBulk)*4)
			argIdx := 1
			for _, r := range topicsBulk {
				values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
				args = append(args, r.id, r.sid, r.name, r.seq)
				argIdx += 4
			}
			sql := fmt.Sprintf(`INSERT INTO topics (id, subject_id, name, sequence_index) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				return fmt.Errorf("internal_error: bulk insert topics failed: %w", err)
			}
		}
		if len(subsBulk) > 0 {
			values := make([]string, 0, len(subsBulk))
			args := make([]interface{}, 0, len(subsBulk)*4)
			argIdx := 1
			for _, r := range subsBulk {
				values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
				args = append(args, r.id, r.tid, r.name, r.seq)
				argIdx += 4
			}
			sql := fmt.Sprintf(`INSERT INTO sub_topics (id, topic_id, name, sequence_index) VALUES %s ON CONFLICT DO NOTHING`, strings.Join(values, ","))
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				return fmt.Errorf("internal_error: bulk insert sub_topics failed: %w", err)
			}
		}

		return nil
	})

	if txErr != nil {
		return "", txErr
	}

	return schoolID, nil
}
