// Package reports orchestrates data from attendance, assessments, and behavior
// domains to produce compiled CBC term report cards for individual students.
package reports

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ─── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound     = fmt.Errorf("report not found: %w", middleware.ErrNotFound)
	ErrInvalidInput = fmt.Errorf("invalid report input: %w", middleware.ErrInvalidInput)
	ErrForbidden    = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
)

// ─── Report Types ─────────────────────────────────────────────────────────

// TermReport is the complete compiled report card for a student in a term.
type TermReport struct {
	Student     TermReportStudent `json:"student"`
	Term        TermInfo          `json:"term"`
	Attendance  AttendanceSection `json:"attendance"`
	Assessments AssessmentSection `json:"assessments"`
	Behavior    BehaviorSection   `json:"behavior"`
}

// TermReportStudent holds student identity information for the report header.
type TermReportStudent struct {
	ID              string  `json:"id"`
	FullName        string  `json:"full_name"`
	Gender          string  `json:"gender"`
	ClassName       string  `json:"class_name"`
	GradeLevel      string  `json:"grade_level"`
	StreamName      string  `json:"stream_name,omitempty"`
	AdmissionNumber *string `json:"admission_number,omitempty"`
	UPINumber       *string `json:"upi_number,omitempty"`
}

// TermInfo identifies the academic term the report covers.
type TermInfo struct {
	TermID       string `json:"term_id"`
	TermName     string `json:"term_name"`
	TermNumber   int    `json:"term_number"`
	AcademicYear string `json:"academic_year"`
	SchoolName   string `json:"school_name"`
}

// AttendanceSection holds the compiled attendance data for the report.
type AttendanceSection struct {
	OverallPercentage float64                  `json:"overall_percentage"`
	PeriodsTotal      int                      `json:"periods_total"`
	PeriodsPresent    int                      `json:"periods_present"`
	PeriodsAbsent     int                      `json:"periods_absent"`
	PeriodsLate       int                      `json:"periods_late"`
	PeriodsExcused    int                      `json:"periods_excused"`
	ByLearningArea    []LearningAreaAttendance `json:"by_learning_area"`
}

// LearningAreaAttendance is the attendance breakdown for one learning area.
type LearningAreaAttendance struct {
	LearningAreaID   string  `json:"learning_area_id"`
	LearningAreaName string  `json:"learning_area_name"`
	PeriodsTotal     int     `json:"periods_total"`
	PeriodsPresent   int     `json:"periods_present"`
	PeriodsAbsent    int     `json:"periods_absent"`
	PeriodsLate      int     `json:"periods_late"`
	PeriodsExcused   int     `json:"periods_excused"`
	Percentage       float64 `json:"percentage"`
}

// AssessmentSection holds the compiled assessment data for the report.
type AssessmentSection struct {
	LearningAreas []LearningAreaAssessment `json:"learning_areas"`
	Sessions      []AssessmentSessionItem  `json:"sessions"`
}

// LearningAreaAssessment is the compiled term grade for one learning area.
type LearningAreaAssessment struct {
	LearningAreaID   string `json:"learning_area_id"`
	LearningAreaName string `json:"learning_area_name"`
	FinalLevel       string `json:"final_level,omitempty"`
	AssessmentCount  int    `json:"assessment_count"`
}

// AssessmentSessionItem is a single published assessment session shown in the report.
type AssessmentSessionItem struct {
	SessionID        string   `json:"session_id"`
	SessionName      string   `json:"session_name"`
	LearningAreaID   string   `json:"learning_area_id"`
	LearningAreaName string   `json:"learning_area_name,omitempty"`
	EvaluationMethod string   `json:"evaluation_method"`
	ScheduledDate    *string  `json:"scheduled_date,omitempty"`
	RawScore         *float64 `json:"raw_score,omitempty"`
	MaxPoints        *float64 `json:"max_points,omitempty"`
	PerformanceLevel *string  `json:"performance_level,omitempty"`
}

// BehaviorSection holds the compiled behavior data for the report.
type BehaviorSection struct {
	TotalIncidents int                `json:"total_incidents"`
	Notes          []BehaviorNoteItem `json:"notes"`
}

// BehaviorNoteItem is a behavior note included in the term report.
type BehaviorNoteItem struct {
	ID           string `json:"id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	Date         string `json:"date"`
	IsUrgent     bool   `json:"is_urgent"`
}

// ─── Provider Function Types (wired from main.go) ─────────────────────────

// StudentProvider resolves student identity and current enrollment info.
type StudentProvider func(ctx context.Context, studentID, tenantID, schoolID string) (*TermReportStudent, error)

// TermProvider resolves term, academic year, and school name.
type TermProvider func(ctx context.Context, termID, tenantID, schoolID string) (*TermInfo, error)

// AttendanceProvider fetches attendance summaries for the report.
type AttendanceProvider func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceSummaryItem, error)

// AssessmentProvider fetches compiled term grades and published assessment sessions.
type AssessmentProvider func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*AssessmentData, error)

// BehaviorProvider fetches approved behavior notes for the term report.
type BehaviorProvider func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]BehaviorNoteItem, error)

// ─── Intermediate DTOs (from providers to service) ───────────────────────

// AttendanceSummaryItem is what the attendance provider returns per learning area.
type AttendanceSummaryItem struct {
	LearningAreaID   string  `json:"learning_area_id"`
	LearningAreaName string  `json:"learning_area_name"`
	PeriodsTotal     int     `json:"periods_total"`
	PeriodsPresent   int     `json:"periods_present"`
	PeriodsAbsent    int     `json:"periods_absent"`
	PeriodsLate      int     `json:"periods_late"`
	PeriodsExcused   int     `json:"periods_excused"`
	Percentage       float64 `json:"percentage"`
}

// AssessmentData is what the assessment provider returns.
type AssessmentData struct {
	LearningAreas []LearningAreaAssessment `json:"learning_areas"`
	Sessions      []AssessmentSessionItem  `json:"sessions"`
}

// ─── Request / Response Payloads ─────────────────────────────────────────

// TermReportResponse wraps the report data.
type TermReportResponse struct {
	Data TermReport `json:"data"`
}
