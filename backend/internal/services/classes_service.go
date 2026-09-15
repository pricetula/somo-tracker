package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ClassListItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Grade        string `json:"grade"`
	Stream       string `json:"stream"`
	AcademicYear string `json:"academicYear"`
	TeacherName  string `json:"teacherName,omitempty"`
}

type ClassDetail struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Grade         string `json:"grade"`
	Stream        string `json:"stream"`
	AcademicYear  string `json:"academicYear"`
	TeacherName   string `json:"teacherName,omitempty"`
	Description   string `json:"description,omitempty"`
	StudentsCount int    `json:"studentsCount"`
}

type CreateClassRequest struct {
	Name     string `json:"name"`
	GradeID  string `json:"gradeId"`
	StreamID string `json:"streamId"`
}

type ClassesService interface {
	ListClasses(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, grades []string, streams []string) ([]ClassListItem, int, error)
	GetClass(ctx context.Context, schoolID uuid.UUID, id uuid.UUID) (*ClassDetail, error)
	CreateClass(ctx context.Context, schoolID uuid.UUID, req CreateClassRequest) (*ClassListItem, error)
}

type classesService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewClassesService(pool *pgxpool.Pool, logger *zap.Logger) ClassesService {
	return &classesService{pool: pool, logger: logger.With(zap.String("service", "classes"))}
}

func (s *classesService) ListClasses(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, grades []string, streams []string) ([]ClassListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Resolve current academic year for the school
	var academicYearID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT id FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1`, schoolID).Scan(&academicYearID); err != nil {
		// No academic year yet → empty list
		return []ClassListItem{}, 0, nil
	}

	args := []interface{}{schoolID, academicYearID}
	query := `
		SELECT cr.id, cr.name, gl.local_label, COALESCE(cr.stream, ''), ay.name, ''
		FROM class_rooms cr
		JOIN grade_levels gl ON gl.id = cr.grade_level_id
		JOIN academic_years ay ON ay.id = cr.academic_year_id
		WHERE cr.school_id = $1 AND cr.academic_year_id = $2
	`
	paramIdx := 3
	if strings.TrimSpace(search) != "" {
		query += fmt.Sprintf(" AND cr.name ILIKE $%d", paramIdx)
		args = append(args, "%"+search+"%")
		paramIdx++
	}
	if len(grades) > 0 {
		placeholders := make([]string, len(grades))
		for i := range grades {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, grades[i])
		}
		query += fmt.Sprintf(" AND gl.local_label IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(grades)
	}
	if len(streams) > 0 {
		placeholders := make([]string, len(streams))
		for i := range streams {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args = append(args, streams[i])
		}
		query += fmt.Sprintf(" AND COALESCE(cr.stream, '') IN (%s)", strings.Join(placeholders, ","))
		paramIdx += len(streams)
	}

	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS cnt"
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		s.logger.Error("list classes count failed", zap.Error(err))
		return nil, 0, fmt.Errorf("internal_error: failed to count classes")
	}

	query += fmt.Sprintf(" ORDER BY cr.created_at DESC LIMIT $%d OFFSET $%d", paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error("list classes query failed", zap.Error(err))
		return nil, 0, fmt.Errorf("internal_error: failed to list classes")
	}
	defer rows.Close()

	var items []ClassListItem
	for rows.Next() {
		var it ClassListItem
		var id uuid.UUID
		var teacher string
		if err := rows.Scan(&id, &it.Name, &it.Grade, &it.Stream, &it.AcademicYear, &teacher); err != nil {
			s.logger.Error("list classes scan failed", zap.Error(err))
			continue
		}
		it.ID = id.String()
		it.TeacherName = teacher
		items = append(items, it)
	}
	return items, total, nil
}

func (s *classesService) GetClass(ctx context.Context, schoolID uuid.UUID, id uuid.UUID) (*ClassDetail, error) {
	query := `
		WITH current_year AS (
			SELECT id FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1
		)
		SELECT cr.id, cr.name, gl.local_label, COALESCE(cr.stream, ''), ay.name, ''
		FROM class_rooms cr
		JOIN grade_levels gl ON gl.id = cr.grade_level_id
		JOIN academic_years ay ON ay.id = cr.academic_year_id
		WHERE cr.school_id = $1 AND cr.id = $2 AND cr.academic_year_id = (SELECT id FROM current_year)
	`
	var d ClassDetail
	var teacher string
	err := s.pool.QueryRow(ctx, query, schoolID, id).Scan(&d.ID, &d.Name, &d.Grade, &d.Stream, &d.AcademicYear, &teacher)
	if err != nil {
		return nil, fmt.Errorf("not_found: class not found")
	}
	d.TeacherName = teacher
	d.StudentsCount = 0
	return &d, nil
}

func (s *classesService) CreateClass(ctx context.Context, schoolID uuid.UUID, req CreateClassRequest) (*ClassListItem, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("bad_request: name is required")
	}
	gradeUUID, err := uuid.Parse(req.GradeID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid gradeId")
	}
	// Resolve grade label
	var gradeLabel string
	if err := s.pool.QueryRow(ctx, `SELECT local_label FROM grade_levels WHERE id = $1`, gradeUUID).Scan(&gradeLabel); err != nil {
		return nil, fmt.Errorf("bad_request: grade not found")
	}
	// Resolve stream name - required
	if strings.TrimSpace(req.StreamID) == "" {
		return nil, fmt.Errorf("bad_request: streamId is required")
	}
	streamUUID, err := uuid.Parse(req.StreamID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid streamId")
	}
	var streamName string
	if err := s.pool.QueryRow(ctx, `SELECT name FROM streams WHERE id = $1 AND school_id = $2`, streamUUID, schoolID).Scan(&streamName); err != nil {
		return nil, fmt.Errorf("bad_request: stream not found")
	}
	// Resolve current academic year for the school
	var ayID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT id FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1`, schoolID).Scan(&ayID); err != nil {
		return nil, fmt.Errorf("bad_request: no academic year found for school")
	}
	newID := uuid.New()
	_, err = s.pool.Exec(ctx, `INSERT INTO class_rooms (id, school_id, academic_year_id, grade_level_id, name, stream, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6, now(), now())`,
		newID, schoolID, ayID, gradeUUID, req.Name, streamName)
	if err != nil {
		s.logger.Error("create class failed", zap.Error(err))
		return nil, fmt.Errorf("internal_error: failed to create class")
	}
	var ayName string
	_ = s.pool.QueryRow(ctx, `SELECT name FROM academic_years WHERE id = $1`, ayID).Scan(&ayName)
	return &ClassListItem{
		ID:           newID.String(),
		Name:         req.Name,
		Grade:        gradeLabel,
		Stream:       streamName,
		AcademicYear: ayName,
	}, nil
}
