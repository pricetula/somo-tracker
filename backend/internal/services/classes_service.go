package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type ClassListItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	GradeID      string `json:"gradeId"`
	Grade        string `json:"grade"`
	Stream       string `json:"stream"`
	AcademicYear string `json:"academicYear"`
	TeacherName  string `json:"teacherName,omitempty"`
}

type ClassDetail struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	GradeID       string `json:"gradeId"`
	Grade         string `json:"grade"`
	Stream        string `json:"stream"`
	AcademicYear  string `json:"academicYear"`
	TeacherName   string `json:"teacherName,omitempty"`
	Description   string `json:"description,omitempty"`
	StudentsCount int    `json:"studentsCount"`
}

type CreateClassRequest struct {
	Name                  string `json:"name"`
	GradeID               string `json:"gradeId"`
	StreamID              string `json:"streamId"`
	BypassFutureTermModal bool   `json:"bypassFutureTermModal,omitempty"`
}

type ClassesService interface {
	ListClasses(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, grades []string, streams []string) ([]ClassListItem, int, error)
	GetClass(ctx context.Context, schoolID uuid.UUID, id uuid.UUID) (*ClassDetail, error)
	CreateClass(ctx context.Context, schoolID uuid.UUID, req CreateClassRequest) (*ClassListItem, error)
}

type classesService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewClassesService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) ClassesService {
	return &classesService{pool: pool, queries: queries, logger: logger.With(zap.String("service", "classes"))}
}

func (s *classesService) ListClasses(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, grades []string, streams []string) ([]ClassListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Resolve current academic period for reads
	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		s.logger.Error("academic period resolve failed", zap.Error(err))
		return []ClassListItem{}, 0, fmt.Errorf("internal_error: failed to resolve academic period")
	}
	if res.ReadYearID == uuid.Nil {
		return []ClassListItem{}, 0, nil
	}
	academicYearID := pgtype.UUID{Bytes: res.ReadYearID, Valid: true}

	args := []interface{}{schoolID, academicYearID}
	query := `
		SELECT cr.id, cr.name, gl.id, gl.local_label, COALESCE(strm.name, ''), ay.name, ''
		FROM class_rooms cr
		JOIN grade_levels gl ON gl.id = cr.grade_level_id
		JOIN academic_years ay ON ay.id = cr.academic_year_id
		LEFT JOIN streams strm ON strm.id = cr.stream_id
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
		query += fmt.Sprintf(" AND cr.stream_id IN (%s)", strings.Join(placeholders, ","))
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
		if err := rows.Scan(&id, &it.Name, &it.GradeID, &it.Grade, &it.Stream, &it.AcademicYear, &teacher); err != nil {
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
	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		s.logger.Error("academic period resolve failed", zap.Error(err))
		return nil, fmt.Errorf("internal_error: failed to resolve academic period")
	}
	if res.ReadYearID == uuid.Nil {
		return nil, fmt.Errorf("not_found: class not found")
	}
	query := `
		SELECT cr.id, cr.name, gl.id, gl.local_label, COALESCE(strm.name, ''), ay.name, ''
		FROM class_rooms cr
		JOIN grade_levels gl ON gl.id = cr.grade_level_id
		JOIN academic_years ay ON ay.id = cr.academic_year_id
		LEFT JOIN streams strm ON strm.id = cr.stream_id
		WHERE cr.school_id = $1 AND cr.id = $2 AND cr.academic_year_id = $3
	`
	var d ClassDetail
	var teacher string
	var gradeID uuid.UUID
	err = s.pool.QueryRow(ctx, query, schoolID, id, res.ReadYearID).Scan(&d.ID, &d.Name, &gradeID, &d.Grade, &d.Stream, &d.AcademicYear, &teacher)
	if err != nil {
		return nil, fmt.Errorf("not_found: class not found")
	}
	d.GradeID = gradeID.String()
	d.TeacherName = teacher
	// Students count for current academic term
	var count int
	countQry := `SELECT COUNT(*) FROM student_class_enrollments sce WHERE sce.class_room_id = $1 AND sce.academic_term_id IN (SELECT id FROM academic_terms WHERE academic_year_id = $2)`
	if err := s.pool.QueryRow(ctx, countQry, id, res.ReadYearID).Scan(&count); err == nil {
		d.StudentsCount = count
	}
	return &d, nil
}

func (s *classesService) CreateClass(ctx context.Context, schoolID uuid.UUID, req CreateClassRequest) (*ClassListItem, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, BadRequest("name is required")
	}
	gradeUUID, err := uuid.Parse(req.GradeID)
	if err != nil {
		return nil, BadRequest("invalid gradeId")
	}
	// Resolve grade label
	var gradeLabel string
	gradeRow, gradeErr := s.queries.GetGradeLevelByID(ctx, pgtype.UUID{Bytes: gradeUUID, Valid: true})
	if gradeErr == nil && gradeRow.LocalLabel != "" {
		gradeLabel = gradeRow.LocalLabel
	} else if gradeErr != nil {
		return nil, BadRequest("grade not found")
	}
	// Resolve stream name - required
	if strings.TrimSpace(req.StreamID) == "" {
		return nil, BadRequest("streamId is required")
	}
	streamUUID, err := uuid.Parse(req.StreamID)
	if err != nil {
		return nil, BadRequest("invalid streamId")
	}
	var streamName string
	streamRow, err := s.queries.GetStream(ctx, pgtype.UUID{Bytes: streamUUID, Valid: true})
	if err != nil || !streamRow.ID.Valid {
		return nil, BadRequest("stream not found")
	}
	expectedSchool := pgtype.UUID{Bytes: schoolID, Valid: true}
	if streamRow.SchoolID != expectedSchool {
		return nil, BadRequest("stream not found")
	}
	streamName = streamRow.Name
	// Resolve current academic period for writes
	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		s.logger.Error("academic period resolve failed", zap.Error(err))
		return nil, fmt.Errorf("internal_error: failed to resolve academic period")
	}
	if res.Status == PeriodStatusBreak && !req.BypassFutureTermModal {
		return nil, fmt.Errorf("no_active_term_future_exists: cannot create class for current period, future term exists")
	}
	if !res.IsWriteAllowed || res.WriteYearID == uuid.Nil {
		return nil, fmt.Errorf("no_next_academic_term: cannot create class for current period")
	}
	ayID := res.WriteYearID
	newID := uuid.New()
	_, err = s.pool.Exec(ctx, `INSERT INTO class_rooms (id, school_id, academic_year_id, grade_level_id, name, stream_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6, now(), now())`,
		newID, schoolID, ayID, gradeUUID, req.Name, streamUUID)
	if err != nil {
		s.logger.Error("create class failed", zap.Error(err))
		return nil, fmt.Errorf("internal_error: failed to create class")
	}
	var ayName string
	if err := s.pool.QueryRow(ctx, `SELECT name FROM academic_years WHERE id = $1`, ayID).Scan(&ayName); err != nil {
		s.logger.Warn("failed to fetch academic year name", zap.Error(err), zap.String("academic_year_id", ayID.String()))
	}
	return &ClassListItem{
		ID:           newID.String(),
		Name:         req.Name,
		Grade:        gradeLabel,
		Stream:       streamName,
		AcademicYear: ayName,
	}, nil
}
