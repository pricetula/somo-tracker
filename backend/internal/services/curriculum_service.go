package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type CurriculumService interface {
	ListSubjects(ctx context.Context, page int, limit int, search string, grade string) ([]CurriculumSubject, int, error)
	GetSubjectDetail(ctx context.Context, id string) (*CurriculumSubjectDetail, error)
	ListTopics(ctx context.Context, subjectID string, page int, limit int) ([]CurriculumTopic, int, error)
	GetTopicDetail(ctx context.Context, id string) (*CurriculumTopicDetail, error)
	ListSubTopics(ctx context.Context, topicID string, page int, limit int) ([]CurriculumSubTopic, int, error)
	GetSubTopic(ctx context.Context, id string) (*CurriculumSubTopic, error)
}

type CurriculumSubject struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Color   string `json:"color"`
	Grade   string `json:"grade"`
	GradeID string `json:"gradeId"`
}

type CurriculumSubjectDetail struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Code   string            `json:"code"`
	Color  string            `json:"color"`
	Grade  string            `json:"grade"`
	Topics []CurriculumTopic `json:"topics"`
}

type CurriculumTopic struct {
	ID          string `json:"id"`
	SubjectID   string `json:"subjectId"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type CurriculumTopicDetail struct {
	ID          string               `json:"id"`
	SubjectID   string               `json:"subjectId"`
	Name        string               `json:"name"`
	Code        string               `json:"code"`
	Description string               `json:"description"`
	SubTopics   []CurriculumSubTopic `json:"subTopics"`
}

type CurriculumSubTopic struct {
	ID        string `json:"id"`
	TopicID   string `json:"topicId"`
	SubjectID string `json:"subjectId"`
	Name      string `json:"name"`
	Code      string `json:"code"`
}

type curriculumService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewCurriculumService(queries *sqlc.Queries, logger *zap.Logger) CurriculumService {
	return &curriculumService{queries: queries, logger: logger.With(zap.String("service", "curriculum"))}
}

func (s *curriculumService) ListSubjects(ctx context.Context, page int, limit int, search string, grade string) ([]CurriculumSubject, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int

	items := make([]CurriculumSubject, 0)

	if strings.TrimSpace(search) != "" {
		totalInt, err2 := s.queries.CountSearchSubjects(ctx, "%"+search+"%")
		if err2 != nil {
			return nil, 0, fmt.Errorf("internal_error: failed to count subjects: %w", err2)
		}
		total = int(totalInt)
		searchRows, err2 := s.queries.SearchSubjects(ctx, sqlc.SearchSubjectsParams{
			Name:   "%" + search + "%",
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err2 != nil {
			return nil, 0, fmt.Errorf("internal_error: failed to list subjects: %w", err2)
		}
		for _, r := range searchRows {
			gradeID := ""
			gradeLabel := ""
			if r.GradeLevelID.Valid {
				gradeID = r.GradeLevelID.String()
				gl, glErr := s.queries.GetGradeLevelByID(ctx, r.GradeLevelID)
				if glErr == nil {
					gradeLabel = gl.LocalLabel
				}
			}
			items = append(items, CurriculumSubject{
				ID:      r.ID.String(),
				Name:    r.Name,
				Code:    r.Code,
				Color:   r.Color,
				Grade:   gradeLabel,
				GradeID: gradeID,
			})
		}
	} else {
		totalInt, err2 := s.queries.CountSubjects(ctx)
		if err2 != nil {
			return nil, 0, fmt.Errorf("internal_error: failed to count subjects: %w", err2)
		}
		total = int(totalInt)
		listRows, err2 := s.queries.ListSubjects(ctx, sqlc.ListSubjectsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err2 != nil {
			return nil, 0, fmt.Errorf("internal_error: failed to list subjects: %w", err2)
		}
		for _, r := range listRows {
			gradeID := ""
			gradeLabel := ""
			if r.GradeLevelID.Valid {
				gradeID = r.GradeLevelID.String()
				gl, glErr := s.queries.GetGradeLevelByID(ctx, r.GradeLevelID)
				if glErr == nil {
					gradeLabel = gl.LocalLabel
				}
			}
			items = append(items, CurriculumSubject{
				ID:      r.ID.String(),
				Name:    r.Name,
				Code:    r.Code,
				Color:   r.Color,
				Grade:   gradeLabel,
				GradeID: gradeID,
			})
		}
	}
	return items, total, nil
}

func (s *curriculumService) GetSubjectDetail(ctx context.Context, id string) (*CurriculumSubjectDetail, error) {
	subjectID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid subject id")
	}

	row, err := s.queries.GetSubjectByID(ctx, pgtype.UUID{Bytes: subjectID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to get subject: %w", err)
	}

	topics, err := s.queries.ListTopicsBySubject(ctx, pgtype.UUID{Bytes: subjectID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to list topics: %w", err)
	}

	grade := ""
	if row.GradeLevelID.Valid {
		glRow, err := s.queries.GetGradeLevelByID(ctx, row.GradeLevelID)
		if err == nil {
			grade = glRow.LocalLabel
		}
	}

	result := &CurriculumSubjectDetail{
		ID:     row.ID.String(),
		Name:   row.Name,
		Code:   row.Code,
		Color:  row.Color.String,
		Grade:  grade,
		Topics: make([]CurriculumTopic, 0, len(topics)),
	}
	for _, t := range topics {
		result.Topics = append(result.Topics, CurriculumTopic{
			ID:        t.ID.String(),
			SubjectID: t.SubjectID.String(),
			Name:      t.Name,
		})
	}

	return result, nil
}

func (s *curriculumService) ListTopics(ctx context.Context, subjectID string, page int, limit int) ([]CurriculumTopic, int, error) {
	_ = page
	_ = limit

	sid, err := uuid.Parse(subjectID)
	if err != nil {
		return nil, 0, fmt.Errorf("bad_request: invalid subject_id")
	}

	total, err := s.queries.CountTopicsBySubject(ctx, pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		return nil, 0, fmt.Errorf("internal_error: failed to count topics: %w", err)
	}

	rows, err := s.queries.ListTopicsBySubject(ctx, pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		return nil, 0, fmt.Errorf("internal_error: failed to list topics: %w", err)
	}

	items := make([]CurriculumTopic, 0, len(rows))
	for _, t := range rows {
		items = append(items, CurriculumTopic{
			ID:        t.ID.String(),
			SubjectID: t.SubjectID.String(),
			Name:      t.Name,
		})
	}

	return items, int(total), nil
}

func (s *curriculumService) GetTopicDetail(ctx context.Context, id string) (*CurriculumTopicDetail, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid topic id")
	}

	row, err := s.queries.GetTopicByID(ctx, pgtype.UUID{Bytes: tid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to get topic: %w", err)
	}

	subTopics, err := s.queries.ListSubTopicsByTopic(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to list sub topics: %w", err)
	}

	result := &CurriculumTopicDetail{
		ID:        row.ID.String(),
		SubjectID: row.SubjectID.String(),
		Name:      row.Name,
		SubTopics: make([]CurriculumSubTopic, 0, len(subTopics)),
	}
	for _, st := range subTopics {
		result.SubTopics = append(result.SubTopics, CurriculumSubTopic{
			ID:        st.ID.String(),
			TopicID:   st.TopicID.String(),
			SubjectID: row.SubjectID.String(),
			Name:      st.Name,
		})
	}

	return result, nil
}

func (s *curriculumService) ListSubTopics(ctx context.Context, topicID string, page int, limit int) ([]CurriculumSubTopic, int, error) {
	_ = page
	_ = limit

	tid, err := uuid.Parse(topicID)
	if err != nil {
		return nil, 0, fmt.Errorf("bad_request: invalid topic_id")
	}

	total, err := s.queries.CountSubTopicsByTopic(ctx, pgtype.UUID{Bytes: tid, Valid: true})
	if err != nil {
		return nil, 0, fmt.Errorf("internal_error: failed to count sub topics: %w", err)
	}

	rows, err := s.queries.ListSubTopicsByTopic(ctx, pgtype.UUID{Bytes: tid, Valid: true})
	if err != nil {
		return nil, 0, fmt.Errorf("internal_error: failed to list sub topics: %w", err)
	}

	items := make([]CurriculumSubTopic, 0, len(rows))
	for _, st := range rows {
		items = append(items, CurriculumSubTopic{
			ID:      st.ID.String(),
			TopicID: st.TopicID.String(),
			Name:    st.Name,
		})
	}

	return items, int(total), nil
}

func (s *curriculumService) GetSubTopic(ctx context.Context, id string) (*CurriculumSubTopic, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid sub_topic id")
	}

	row, err := s.queries.GetSubTopicByID(ctx, pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to get sub topic: %w", err)
	}

	return &CurriculumSubTopic{
		ID:      row.ID.String(),
		TopicID: row.TopicID.String(),
		Name:    row.Name,
	}, nil
}
