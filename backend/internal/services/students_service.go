package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type StudentListItem struct {
	StudentID       string `json:"student_id"`
	AdmissionNumber string `json:"admission_number"`
	FullName        string `json:"full_name"`
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`
	ClassID         string `json:"class_id"`
	ClassName       string `json:"class_name"`
	GradeLevel      string `json:"grade_level"`
}

type StudentListResponse struct {
	Items []StudentListItem `json:"items"`
	Total int               `json:"total"`
}

type StudentSummary struct {
	TotalStudents          int64 `json:"total_students"`
	MaleCount              int64 `json:"male_count"`
	FemaleCount            int64 `json:"female_count"`
	UnassignedCount        int64 `json:"unassigned_count"`
	UnlinkedGuardiansCount int64 `json:"unlinked_guardians_count"`
}

type StudentProfile struct {
	StudentID             string         `json:"student_id"`
	AdmissionNumber       string         `json:"admission_number"`
	NemisUPN              *string        `json:"nemis_upn,omitempty"`
	FullName              string         `json:"full_name"`
	DateOfBirth           string         `json:"date_of_birth"`
	Gender                string         `json:"gender"`
	Address               *string        `json:"address,omitempty"`
	Phone                 *string        `json:"phone,omitempty"`
	Email                 *string        `json:"email,omitempty"`
	Nationality           *string        `json:"nationality,omitempty"`
	EmergencyContactName  *string        `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string        `json:"emergency_contact_phone,omitempty"`
	SpecialNeeds          *string        `json:"special_needs,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type UpdateStudentRequest struct {
	FullName              *string        `json:"full_name,omitempty"`
	DateOfBirth           *string        `json:"date_of_birth,omitempty"`
	Gender                *string        `json:"gender,omitempty"`
	NemisUPN              *string        `json:"nemis_upn,omitempty"`
	Address               *string        `json:"address,omitempty"`
	Phone                 *string        `json:"phone,omitempty"`
	Email                 *string        `json:"email,omitempty"`
	Nationality           *string        `json:"nationality,omitempty"`
	EmergencyContactName  *string        `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string        `json:"emergency_contact_phone,omitempty"`
	SpecialNeeds          *string        `json:"special_needs,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type StudentsService interface {
	ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, classID *uuid.UUID, withoutClass bool, withoutGuardian bool) (*StudentListResponse, error)
	DeleteStudents(ctx context.Context, schoolID uuid.UUID, studentIDs []uuid.UUID) error
	GetStudentSummary(ctx context.Context, schoolID uuid.UUID) (*StudentSummary, error)
	GetStudentByID(ctx context.Context, schoolID uuid.UUID, studentID uuid.UUID) (*StudentProfile, error)
	UpdateStudent(ctx context.Context, schoolID uuid.UUID, studentID uuid.UUID, req UpdateStudentRequest) error
}

type studentsService struct {
	queries *sqlc.Queries
}

func NewStudentsService(queries *sqlc.Queries) StudentsService {
	return &studentsService{queries: queries}
}

func (s *studentsService) ListStudents(ctx context.Context, schoolID uuid.UUID, page int, limit int, search string, classID *uuid.UUID, withoutClass bool, withoutGuardian bool) (*StudentListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	var classUUID pgtype.UUID
	if classID != nil {
		classUUID = pgtype.UUID{Bytes: *classID, Valid: true}
	}
	rows, err := s.queries.ListStudents(ctx, sqlc.ListStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
		Column5:  classUUID,
		Column6:  withoutClass,
		Column7:  withoutGuardian,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}

	total, err := s.queries.CountStudents(ctx, sqlc.CountStudentsParams{
		SchoolID: schoolUUID,
		Column2:  search,
		Column3:  classUUID,
		Column4:  withoutClass,
		Column5:  withoutGuardian,
	})
	if err != nil {
		return nil, fmt.Errorf("count students: %w", err)
	}

	items := make([]StudentListItem, 0, len(rows))
	for _, r := range rows {
		dobStr := ""
		if r.DateOfBirth.Valid {
			dobStr = r.DateOfBirth.Time.Format("2006-01-02")
		}
		classID := ""
		if r.ClassID.Valid {
			classID = r.ClassID.String()
		}
		className := ""
		if r.ClassName.Valid {
			className = r.ClassName.String
		}
		genderStr := fmt.Sprintf("%v", r.Gender)
		gradeLevel := ""
		if r.GradeLevel.Valid {
			gradeLevel = r.GradeLevel.String
		}
		items = append(items, StudentListItem{
			StudentID:       r.StudentID.String(),
			AdmissionNumber: r.AdmissionNumber,
			FullName:        r.FullName,
			DateOfBirth:     dobStr,
			Gender:          genderStr,
			ClassID:         classID,
			ClassName:       className,
			GradeLevel:      gradeLevel,
		})
	}

	return &StudentListResponse{
		Items: items,
		Total: int(total),
	}, nil
}

func (s *studentsService) DeleteStudents(ctx context.Context, schoolID uuid.UUID, studentIDs []uuid.UUID) error {
	if len(studentIDs) == 0 {
		return BadRequest("student_ids must be provided and non-empty")
	}

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	uuids := make([]pgtype.UUID, 0, len(studentIDs))
	for _, id := range studentIDs {
		uuids = append(uuids, pgtype.UUID{Bytes: id, Valid: true})
	}

	if err := s.queries.DeleteStudents(ctx, sqlc.DeleteStudentsParams{
		SchoolID: schoolUUID,
		Column2:  uuids,
	}); err != nil {
		return fmt.Errorf("delete students: %w", err)
	}
	return nil
}

func (s *studentsService) GetStudentSummary(ctx context.Context, schoolID uuid.UUID) (*StudentSummary, error) {
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}

	// Resolve current academic term strictly on backend; never pass from client
	resolver := NewAcademicPeriodResolver(s.queries, zap.NewNop())
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve academic period: %w", err)
	}
	if res.ReadTermID == uuid.Nil {
		return nil, fmt.Errorf("no academic term found for school")
	}
	termUUID := pgtype.UUID{Bytes: res.ReadTermID, Valid: true}

	row, err := s.queries.GetStudentSummary(ctx, sqlc.GetStudentSummaryParams{
		SchoolID:       schoolUUID,
		AcademicTermID: termUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("get student summary: %w", err)
	}

	return &StudentSummary{
		TotalStudents:          int64(row.TotalStudents),
		MaleCount:              int64(row.MaleCount),
		FemaleCount:            int64(row.FemaleCount),
		UnassignedCount:        int64(row.UnassignedCount),
		UnlinkedGuardiansCount: int64(row.UnlinkedGuardiansCount),
	}, nil
}

func (s *studentsService) GetStudentByID(ctx context.Context, schoolID uuid.UUID, studentID uuid.UUID) (*StudentProfile, error) {
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	studentUUID := pgtype.UUID{Bytes: studentID, Valid: true}

	row, err := s.queries.GetStudentByID(ctx, sqlc.GetStudentByIDParams{
		StudentID: studentUUID,
		SchoolID:  schoolUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("get student by id: %w", err)
	}

	dobStr := ""
	if row.DateOfBirth.Valid {
		dobStr = row.DateOfBirth.Time.Format("2006-01-02")
	}

	profile := &StudentProfile{
		StudentID:       row.StudentID.String(),
		AdmissionNumber: row.AdmissionNumber,
		FullName:        row.FullName,
		DateOfBirth:     dobStr,
		Gender:          fmt.Sprintf("%v", row.Gender),
	}
	if row.NemisUpn.Valid {
		upn := row.NemisUpn.String
		profile.NemisUPN = &upn
	}
	if row.Address.Valid {
		addr := row.Address.String
		profile.Address = &addr
	}
	if row.Phone.Valid {
		ph := row.Phone.String
		profile.Phone = &ph
	}
	if row.Email.Valid {
		em := row.Email.String
		profile.Email = &em
	}
	if row.Nationality.Valid {
		nat := row.Nationality.String
		profile.Nationality = &nat
	}
	if row.EmergencyContactName.Valid {
		ecName := row.EmergencyContactName.String
		profile.EmergencyContactName = &ecName
	}
	if row.EmergencyContactPhone.Valid {
		ecPhone := row.EmergencyContactPhone.String
		profile.EmergencyContactPhone = &ecPhone
	}
	if row.SpecialNeeds.Valid {
		sn := row.SpecialNeeds.String
		profile.SpecialNeeds = &sn
	}
	// metadata parsing omitted for brevity; return empty map if needed
	return profile, nil
}

func (s *studentsService) UpdateStudent(ctx context.Context, schoolID uuid.UUID, studentID uuid.UUID, req UpdateStudentRequest) error {
	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	studentUUID := pgtype.UUID{Bytes: studentID, Valid: true}

	var dob pgtype.Date
	if req.DateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			return BadRequest("date_of_birth must be YYYY-MM-DD")
		}
		dob = pgtype.Date{Time: parsed, Valid: true}
	}

	fullName := ""
	if req.FullName != nil {
		fullName = *req.FullName
	}

	genderStr := ""
	if req.Gender != nil {
		genderStr = *req.Gender
	}

	var nemisUpn pgtype.Text
	if req.NemisUPN != nil {
		nemisUpn = pgtype.Text{String: *req.NemisUPN, Valid: true}
	}

	var address, phone, email, nationality pgtype.Text
	var emergencyContactName, emergencyContactPhone, specialNeeds pgtype.Text
	if req.Address != nil {
		address = pgtype.Text{String: *req.Address, Valid: true}
	}
	if req.Phone != nil {
		phone = pgtype.Text{String: *req.Phone, Valid: true}
	}
	if req.Email != nil {
		email = pgtype.Text{String: *req.Email, Valid: true}
	}
	if req.Nationality != nil {
		nationality = pgtype.Text{String: *req.Nationality, Valid: true}
	}
	if req.EmergencyContactName != nil {
		emergencyContactName = pgtype.Text{String: *req.EmergencyContactName, Valid: true}
	}
	if req.EmergencyContactPhone != nil {
		emergencyContactPhone = pgtype.Text{String: *req.EmergencyContactPhone, Valid: true}
	}
	if req.SpecialNeeds != nil {
		specialNeeds = pgtype.Text{String: *req.SpecialNeeds, Valid: true}
	}

	var metadataBytes []byte
	if req.Metadata != nil {
		b, err := json.Marshal(req.Metadata)
		if err != nil {
			return BadRequest("metadata must be valid JSON")
		}
		metadataBytes = b
	}

	if err := s.queries.UpdateStudent(ctx, sqlc.UpdateStudentParams{
		StudentID:    studentUUID,
		SchoolID:     schoolUUID,
		Column3:      fullName,
		DateOfBirth:  dob,
		Column5:      genderStr,
		Column6:      nemisUpn,
		Metadata:     metadataBytes,
		Address:      address,
		Column9:      phone,
		Column10:     email,
		Column11:     nationality,
		Column12:     emergencyContactName,
		Column13:     emergencyContactPhone,
		SpecialNeeds: specialNeeds,
	}); err != nil {
		return fmt.Errorf("update student: %w", err)
	}
	return nil
}
