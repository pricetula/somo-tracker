package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type PeriodStatus string

const (
	PeriodStatusActive  PeriodStatus = "active"
	PeriodStatusBreak   PeriodStatus = "break"
	PeriodStatusExpired PeriodStatus = "expired"
	PeriodStatusNone    PeriodStatus = "none"
)

type PeriodResolution struct {
	Status              PeriodStatus
	ReadTermID          uuid.UUID
	ReadYearID          uuid.UUID
	WriteTermID         uuid.UUID
	WriteYearID         uuid.UUID
	NextTermID          uuid.UUID
	NextYearID          uuid.UUID
	IsWriteAllowed      bool
	IsAttendanceAllowed bool
}

type AcademicPeriodResolver interface {
	Resolve(ctx context.Context, schoolID uuid.UUID, now time.Time, periodMode string) (*PeriodResolution, error)
}

type academicPeriodResolver struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewAcademicPeriodResolver(queries *sqlc.Queries, logger *zap.Logger) AcademicPeriodResolver {
	return &academicPeriodResolver{queries: queries, logger: logger.With(zap.String("service", "academic_period_resolver"))}
}

func (r *academicPeriodResolver) Resolve(ctx context.Context, schoolID uuid.UUID, now time.Time, periodMode string) (*PeriodResolution, error) {
	if periodMode != "" && periodMode != "current" && periodMode != "upcoming" {
		return nil, fmt.Errorf("bad_request: period must be current or upcoming")
	}

	loc, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)

	schoolUUID := pgtype.UUID{Bytes: schoolID, Valid: true}
	terms, err := r.queries.ListAcademicTermsBySchool(ctx, schoolUUID)
	if err != nil {
		r.logger.Error("failed to list academic terms", zap.Error(err), zap.String("school_id", schoolID.String()))
		return nil, fmt.Errorf("internal_error: failed to load academic terms")
	}

	if len(terms) == 0 {
		return &PeriodResolution{
			Status:              PeriodStatusNone,
			IsWriteAllowed:      false,
			IsAttendanceAllowed: false,
		}, nil
	}

	type termInfo struct {
		ID             uuid.UUID
		AcademicYearID uuid.UUID
		Start          time.Time
		End            time.Time
	}
	var infos []termInfo
	for _, t := range terms {
		if !t.StartDate.Valid || !t.EndDate.Valid {
			continue
		}
		start := t.StartDate.Time.In(loc)
		startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end := t.EndDate.Time.In(loc)
		endDate := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
		infos = append(infos, termInfo{
			ID:             uuid.UUID(t.ID.Bytes),
			AcademicYearID: uuid.UUID(t.AcademicYearID.Bytes),
			Start:          startDate,
			End:            endDate,
		})
	}

	// Find active term
	var active *termInfo
	var started []termInfo
	var future []termInfo
	for i := range infos {
		ti := infos[i]
		if !ti.Start.After(today) && !ti.End.Before(today) {
			// inclusive
			active = &ti
		}
		if !ti.Start.After(today) {
			started = append(started, ti)
		}
		if ti.Start.After(today) {
			future = append(future, ti)
		}
	}

	res := &PeriodResolution{}
	// Ensure deterministic order
	var lastStarted *termInfo
	if len(started) > 0 {
		lastStarted = &started[0]
		for i := 1; i < len(started); i++ {
			s := started[i]
			if s.End.After(lastStarted.End) || (s.End.Equal(lastStarted.End) && s.Start.After(lastStarted.Start)) {
				lastStarted = &started[i]
			}
		}
	}

	if active != nil {
		res.Status = PeriodStatusActive
		res.ReadTermID = active.ID
		res.ReadYearID = active.AcademicYearID
		// find next term
		var next *termInfo
		for _, f := range future {
			next = &f
			break
		}
		if next != nil {
			res.NextTermID = next.ID
			res.NextYearID = next.AcademicYearID
		}
		// write term depends on periodMode
		if periodMode == "upcoming" {
			if next != nil {
				res.WriteTermID = next.ID
				res.WriteYearID = next.AcademicYearID
				res.IsWriteAllowed = true
			} else {
				res.IsWriteAllowed = false
			}
		} else {
			res.WriteTermID = active.ID
			res.WriteYearID = active.AcademicYearID
			res.IsWriteAllowed = true
		}
		res.IsAttendanceAllowed = true
		return res, nil
	}

	// No active term
	if len(future) > 0 {
		if len(started) > 0 {
			// break between terms
			res.Status = PeriodStatusBreak
			res.ReadTermID = lastStarted.ID
			res.ReadYearID = lastStarted.AcademicYearID
			firstFuture := future[0]
			res.WriteTermID = firstFuture.ID
			res.WriteYearID = firstFuture.AcademicYearID
			res.NextTermID = firstFuture.ID
			res.NextYearID = firstFuture.AcademicYearID
			res.IsWriteAllowed = true
			res.IsAttendanceAllowed = false
			return res, nil
		}
		// pre-start
		// reads empty, writes target first term
		res.Status = PeriodStatusBreak
		firstFuture := future[0]
		res.WriteTermID = firstFuture.ID
		res.WriteYearID = firstFuture.AcademicYearID
		res.NextTermID = firstFuture.ID
		res.NextYearID = firstFuture.AcademicYearID
		res.IsWriteAllowed = true
		res.IsAttendanceAllowed = false
		// Read IDs remain zero
		return res, nil
	}

	// No future terms
	if len(started) > 0 {
		res.Status = PeriodStatusExpired
		lastStarted := started[0]
		for _, s := range started[1:] {
			if s.End.After(lastStarted.End) || (s.End.Equal(lastStarted.End) && s.Start.After(lastStarted.Start)) {
				lastStarted = s
			}
		}
		res.ReadTermID = lastStarted.ID
		res.ReadYearID = lastStarted.AcademicYearID
		res.IsWriteAllowed = false
		res.IsAttendanceAllowed = false
		return res, nil
	}

	// No terms
	res.Status = PeriodStatusNone
	res.IsWriteAllowed = false
	res.IsAttendanceAllowed = false
	return res, nil
}
