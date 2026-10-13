package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type AcademicPeriodData struct {
	Exists    bool   `json:"exists"`
	Status    string `json:"status"`
	YearID    string `json:"yearId"`
	YearName  string `json:"yearName"`
	TermID    string `json:"termId"`
	TermName  string `json:"termName"`
	TermStart string `json:"termStart"`
	TermEnd   string `json:"termEnd"`
}

type ClassroomItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Grade  string `json:"grade"`
	Stream string `json:"stream"`
}

type ClassroomsData struct {
	Count int             `json:"count"`
	Items []ClassroomItem `json:"items"`
}

type EnrollmentItem struct {
	ClassID      string `json:"classId"`
	ClassName    string `json:"className"`
	Grade        string `json:"grade"`
	Stream       string `json:"stream"`
	StudentCount int64  `json:"studentCount"`
}

type EnrollmentsData struct {
	Items []EnrollmentItem `json:"items"`
}

type StreamsData struct {
	Count int `json:"count"`
}

type SetupSummaryResponse struct {
	AcademicPeriod *AcademicPeriodData `json:"academicPeriod"`
	Streams        *StreamsData        `json:"streams"`
	Classrooms     *ClassroomsData     `json:"classrooms"`
	Enrollments    *EnrollmentsData    `json:"enrollments"`
}

type SetupSummaryService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewSetupSummaryService(queries *sqlc.Queries, logger *zap.Logger) *SetupSummaryService {
	return &SetupSummaryService{queries: queries, logger: logger}
}

func (s *SetupSummaryService) GetSetupSummary(ctx context.Context, schoolID string) (*SetupSummaryResponse, error) {
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, BadRequest("invalid school_id")
	}

	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolUUID, time.Now(), "current")
	if err != nil {
		return nil, err
	}

	resp := &SetupSummaryResponse{}

	// Academic Period
	if res.ReadYearID == uuid.Nil {
		resp.AcademicPeriod = &AcademicPeriodData{
			Exists: false,
			Status: string(res.Status),
		}
		resp.Classrooms = nil
		resp.Enrollments = nil
		return resp, nil
	}

	// Get latest term names
	termRow, err := s.queries.GetLatestTermBySchool(ctx, pgtype.UUID{Bytes: schoolUUID, Valid: true})
	// fallback if query not generated
	yearName := ""
	termName := ""
	termStart := ""
	termEnd := ""
	if err == nil && termRow.ID.Valid {
		yearName = termRow.AcademicYearName
		termName = termRow.TermName
		if termRow.StartDate.Valid {
			termStart = termRow.StartDate.Time.Format("2006-01-02")
		}
		if termRow.EndDate.Valid {
			termEnd = termRow.EndDate.Time.Format("2006-01-02")
		}
	} else {
		// fallback using ListAcademicTermsBySchool
		terms, _ := s.queries.ListAcademicTermsBySchool(ctx, pgtype.UUID{Bytes: schoolUUID, Valid: true})
		for _, t := range terms {
			if t.AcademicYearID.Bytes == res.ReadYearID && t.ID.Bytes == res.ReadTermID {
				yearName = t.AcademicYearName
				termName = t.Name
				if t.StartDate.Valid {
					termStart = t.StartDate.Time.Format("2006-01-02")
				}
				if t.EndDate.Valid {
					termEnd = t.EndDate.Time.Format("2006-01-02")
				}
				break
			}
		}
	}

	resp.AcademicPeriod = &AcademicPeriodData{
		Exists:    true,
		Status:    string(res.Status),
		YearID:    res.ReadYearID.String(),
		YearName:  yearName,
		TermID:    res.ReadTermID.String(),
		TermName:  termName,
		TermStart: termStart,
		TermEnd:   termEnd,
	}

	// Streams
	streamRows, err := s.queries.ListStreamsBySchool(ctx, pgtype.UUID{Bytes: schoolUUID, Valid: true})
	if err == nil {
		resp.Streams = &StreamsData{Count: len(streamRows)}
	} else {
		resp.Streams = &StreamsData{Count: 0}
	}

	// Classrooms
	classCount, err := s.queries.CountClassRoomsBySchoolAndYear(ctx, sqlc.CountClassRoomsBySchoolAndYearParams{
		SchoolID:       pgtype.UUID{Bytes: schoolUUID, Valid: true},
		AcademicYearID: pgtype.UUID{Bytes: res.ReadYearID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	if classCount == 0 {
		resp.Classrooms = &ClassroomsData{Count: 0, Items: []ClassroomItem{}}
		resp.Enrollments = nil
		return resp, nil
	}

	rooms, err := s.queries.ListClassRoomsBySchoolAndYear(ctx, sqlc.ListClassRoomsBySchoolAndYearParams{
		SchoolID:       pgtype.UUID{Bytes: schoolUUID, Valid: true},
		AcademicYearID: pgtype.UUID{Bytes: res.ReadYearID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	items := make([]ClassroomItem, 0, len(rooms))
	for _, r := range rooms {
		items = append(items, ClassroomItem{
			ID:     r.ID.String(),
			Name:   r.Name,
			Grade:  r.Grade,
			Stream: r.Stream,
		})
	}
	resp.Classrooms = &ClassroomsData{Count: int(classCount), Items: items}

	// Enrollments summary
	summary, err := s.queries.ListClassEnrollmentsSummaryBySchoolAndYear(ctx, sqlc.ListClassEnrollmentsSummaryBySchoolAndYearParams{
		SchoolID:       pgtype.UUID{Bytes: schoolUUID, Valid: true},
		AcademicYearID: pgtype.UUID{Bytes: res.ReadYearID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	enrollmentItems := make([]EnrollmentItem, 0, len(summary))
	for _, srow := range summary {
		enrollmentItems = append(enrollmentItems, EnrollmentItem{
			ClassID:      srow.ClassID.String(),
			ClassName:    srow.ClassName,
			Grade:        srow.Grade,
			Stream:       srow.Stream,
			StudentCount: srow.StudentCount,
		})
	}
	resp.Enrollments = &EnrollmentsData{Items: enrollmentItems}
	return resp, nil
}
