package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserCounts struct {
	Students       int `json:"students"`
	StudentsMale   int `json:"students_male"`
	StudentsFemale int `json:"students_female"`
	Teachers       int `json:"teachers"`
	Guardians      int `json:"guardians"`
	Finance        int `json:"finance"`
	Admins         int `json:"admins"`
	TotalUsers     int `json:"total_users"`
}

type UserCountsService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewUserCountsService(pool *pgxpool.Pool, logger *zap.Logger) *UserCountsService {
	return &UserCountsService{
		pool:   pool,
		logger: logger.With(zap.String("service", "user_counts")),
	}
}

func (s UserCountsService) GetUserCounts(ctx context.Context, schoolID string) (UserCounts, error) {
	query := `
SELECT
  COALESCE(sgc.total_count, 0) AS students,
  COALESCE(sgc.male_count, 0) AS students_male,
  COALESCE(sgc.female_count, 0) AS students_female,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'TEACHER' AND is_active = true), 0) AS teachers,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'GUARDIAN' AND is_active = true), 0) AS guardians,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'FINANCE' AND is_active = true), 0) AS finance,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'ADMIN' AND is_active = true), 0) AS admins
FROM student_gender_counts sgc
WHERE sgc.school_id = $1
`
	var counts UserCounts
	err := s.pool.QueryRow(ctx, query, schoolID).Scan(
		&counts.Students,
		&counts.StudentsMale,
		&counts.StudentsFemale,
		&counts.Teachers,
		&counts.Guardians,
		&counts.Finance,
		&counts.Admins,
	)
	if err != nil {
		// fallback if no gender counts row exists
		fallbackQuery := `
SELECT
  0 AS students,
  0 AS students_male,
  0 AS students_female,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'TEACHER' AND is_active = true), 0) AS teachers,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'GUARDIAN' AND is_active = true), 0) AS guardians,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'FINANCE' AND is_active = true), 0) AS finance,
  COALESCE((SELECT COUNT(*) FROM school_memberships WHERE school_id = $1 AND role = 'ADMIN' AND is_active = true), 0) AS admins
`
		if err2 := s.pool.QueryRow(ctx, fallbackQuery, schoolID).Scan(
			&counts.Students,
			&counts.StudentsMale,
			&counts.StudentsFemale,
			&counts.Teachers,
			&counts.Guardians,
			&counts.Finance,
			&counts.Admins,
		); err2 != nil {
			s.logger.Error("user counts query failed", zap.Error(err2), zap.String("school_id", schoolID))
			return UserCounts{}, fmt.Errorf("internal_error: failed to get user counts: %w", err2)
		}
	}
	counts.TotalUsers = counts.Students + counts.Teachers + counts.Guardians + counts.Finance + counts.Admins
	return counts, nil
}
