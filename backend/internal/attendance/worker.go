package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
)

// ─── Task Type Constants ──────────────────────────────────────────────────

const (
	// TaskRefreshTeacherDeliverySummaries refreshes teacher_delivery_summaries
	// for a given term. Payload: DeliveryRefreshPayload.
	TaskRefreshTeacherDeliverySummaries = "attendance:refresh_teacher_delivery_summaries"

	// TaskRefreshTeacherWorkloadSummaries refreshes teacher_workload_summaries
	// for a given academic year. Payload: WorkloadRefreshPayload.
	TaskRefreshTeacherWorkloadSummaries = "attendance:refresh_teacher_workload_summaries"

	// TaskRefreshAttendanceTermSummaries refreshes attendance_term_summaries
	// for a given term. Payload: AttendanceTermRefreshPayload.
	TaskRefreshAttendanceTermSummaries = "attendance:refresh_term_summaries"

	// TaskRefreshClassDailySummary refreshes class_daily_attendance_summaries
	// for a specific class+date, resolved from the timetable slot + date.
	// Payload: ClassDailyRefreshPayload.
	TaskRefreshClassDailySummary = "attendance:refresh_class_daily_summary"
)

// ─── Task Payloads ────────────────────────────────────────────────────────

// DeliveryRefreshPayload is the payload for refreshing teacher delivery summaries.
type DeliveryRefreshPayload struct {
	TermID string `json:"term_id"`
}

// WorkloadRefreshPayload is the payload for refreshing teacher workload summaries.
type WorkloadRefreshPayload struct {
	AcademicYearID string `json:"academic_year_id"`
}

// AttendanceTermRefreshPayload is the payload for refreshing attendance term summaries.
type AttendanceTermRefreshPayload struct {
	TenantID string `json:"tenant_id"`
	SchoolID string `json:"school_id"`
	TermID   string `json:"term_id"`
}

// ClassDailyRefreshPayload is the payload for refreshing class daily attendance summaries.
type ClassDailyRefreshPayload struct {
	TenantID        string `json:"tenant_id"`
	SchoolID        string `json:"school_id"`
	TimetableSlotID string `json:"timetable_slot_id"`
	Date            string `json:"date"`
}

// ─── Asynq Client ─────────────────────────────────────────────────────────

// NewAsynqClient creates an Asynq client from the Redis pool.
func NewAsynqClient(pools *database.Pools) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr: pools.Redis.Options().Addr,
	})
}

// ─── Enqueuer ─────────────────────────────────────────────────────────────

// Enqueuer publishes background refresh tasks to Asynq.
type Enqueuer struct {
	client *asynq.Client
}

// NewEnqueuer creates a new Enqueuer.
func NewEnqueuer(client *asynq.Client) *Enqueuer {
	return &Enqueuer{client: client}
}

// EnqueueTeacherDeliveryRefresh enqueues a task to refresh teacher delivery
// summaries for the given term. Best-effort: logs failures but does not
// block the caller.
func (e *Enqueuer) EnqueueTeacherDeliveryRefresh(ctx context.Context, termID string) {
	payload, _ := json.Marshal(DeliveryRefreshPayload{TermID: termID})
	task := asynq.NewTask(TaskRefreshTeacherDeliverySummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "attendance: enqueue teacher delivery refresh failed",
			"term_id", termID, "error", err,
		)
	}
}

// EnqueueTeacherWorkloadRefresh enqueues a task to refresh teacher workload
// summaries for the given academic year.
func (e *Enqueuer) EnqueueTeacherWorkloadRefresh(ctx context.Context, academicYearID string) {
	payload, _ := json.Marshal(WorkloadRefreshPayload{AcademicYearID: academicYearID})
	task := asynq.NewTask(TaskRefreshTeacherWorkloadSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "attendance: enqueue teacher workload refresh failed",
			"academic_year_id", academicYearID, "error", err,
		)
	}
}

// EnqueueAttendanceTermRefresh enqueues a task to refresh attendance term
// summaries for the given term.
func (e *Enqueuer) EnqueueAttendanceTermRefresh(ctx context.Context, tenantID, schoolID, termID string) {
	payload, _ := json.Marshal(AttendanceTermRefreshPayload{
		TenantID: tenantID,
		SchoolID: schoolID,
		TermID:   termID,
	})
	task := asynq.NewTask(TaskRefreshAttendanceTermSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "attendance: enqueue term summary refresh failed",
			"term_id", termID, "error", err,
		)
	}
}

// EnqueueClassDailyRefresh enqueues a task to refresh the class daily
// attendance summary for the given timetable slot + date.
func (e *Enqueuer) EnqueueClassDailyRefresh(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) {
	payload, _ := json.Marshal(ClassDailyRefreshPayload{
		TenantID:        tenantID,
		SchoolID:        schoolID,
		TimetableSlotID: timetableSlotID,
		Date:            date,
	})
	task := asynq.NewTask(TaskRefreshClassDailySummary, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		slog.WarnContext(ctx, "attendance: enqueue class daily refresh failed",
			"timetable_slot_id", timetableSlotID, "date", date, "error", err,
		)
	}
}

// ─── Worker ───────────────────────────────────────────────────────────────

// Worker processes background attendance summary refresh tasks.
type Worker struct {
	pools  *database.Pools
	server *asynq.Server
}

// NewWorker creates a new background attendance summary refresh worker.
func NewWorker(pools *database.Pools) *Worker {
	return &Worker{pools: pools}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	w.server = asynq.NewServer(
		asynq.RedisClientOpt{Addr: w.pools.Redis.Options().Addr},
		asynq.Config{
			Concurrency: 1,
			Queues:      map[string]int{"summaries": 10},
			Logger:      asynqLogger{},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRefreshTeacherDeliverySummaries, w.handleTeacherDeliveryRefresh)
	mux.HandleFunc(TaskRefreshTeacherWorkloadSummaries, w.handleTeacherWorkloadRefresh)
	mux.HandleFunc(TaskRefreshAttendanceTermSummaries, w.handleAttendanceTermRefresh)
	mux.HandleFunc(TaskRefreshClassDailySummary, w.handleClassDailyRefresh)

	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("attendance.Worker.Start: %w", err)
	}
	slog.InfoContext(ctx, "attendance.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	if w.server != nil {
		w.server.Shutdown()
	}
	slog.InfoContext(ctx, "attendance.Worker: asynq server stopped")
	return nil
}

// ─── Task Handlers ────────────────────────────────────────────────────────

func (w *Worker) handleTeacherDeliveryRefresh(ctx context.Context, t *asynq.Task) error {
	var p DeliveryRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherDeliveryRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "attendance: refreshing teacher delivery summaries", "term_id", p.TermID)

	_, err := w.pools.PG.Exec(ctx, `SELECT fn_compute_teacher_delivery_summaries($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherDeliveryRefresh: %w", err)
	}
	slog.InfoContext(ctx, "attendance: teacher delivery summaries refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleAttendanceTermRefresh(ctx context.Context, t *asynq.Task) error {
	var p AttendanceTermRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleAttendanceTermRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "attendance: refreshing term summaries", "term_id", p.TermID)

	_, err := w.pools.PG.Exec(ctx, `
		INSERT INTO attendance_term_summaries (
			tenant_id, school_id, student_id, academic_term_id, academic_year_id,
			learning_area_id,
			periods_total, periods_present, periods_absent, periods_late, periods_excused,
			attendance_percentage, last_refreshed_at
		)
		SELECT
			$1 AS tenant_id,
			ar.school_id,
			ar.student_id,
			ar.academic_term_id,
			t.academic_year_id,
			ts.learning_area_id,
			COUNT(*)::INT AS periods_total,
			COUNT(*) FILTER (WHERE ar.status = 'PRESENT')::INT AS periods_present,
			COUNT(*) FILTER (WHERE ar.status = 'ABSENT')::INT AS periods_absent,
			COUNT(*) FILTER (WHERE ar.status = 'LATE')::INT AS periods_late,
			COUNT(*) FILTER (WHERE ar.status = 'EXCUSED')::INT AS periods_excused,
			ROUND(
				(COUNT(*) FILTER (WHERE ar.status = 'PRESENT') * 100.0 / NULLIF(COUNT(*), 0)),
				2
			) AS attendance_percentage,
			NOW() AS last_refreshed_at
		FROM attendance_records ar
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN academic_terms t ON t.id = ar.academic_term_id
		LEFT JOIN cbc_attendance_sessions s
			ON s.timetable_slot_id = ar.timetable_slot_id
			AND s.date = ar.date
			AND s.tenant_id = ar.tenant_id
		WHERE ar.tenant_id = $1 AND ar.school_id = $2 AND ar.academic_term_id = $3
		  AND (s.status IS NULL OR s.status != 'SKIPPED')
		GROUP BY ar.school_id, ar.student_id, ar.academic_term_id, t.academic_year_id, ts.learning_area_id
		ON CONFLICT (student_id, academic_term_id, learning_area_id)
		DO UPDATE SET
			academic_year_id     = EXCLUDED.academic_year_id,
			periods_total        = EXCLUDED.periods_total,
			periods_present      = EXCLUDED.periods_present,
			periods_absent       = EXCLUDED.periods_absent,
			periods_late         = EXCLUDED.periods_late,
			periods_excused      = EXCLUDED.periods_excused,
			attendance_percentage = EXCLUDED.attendance_percentage,
			last_refreshed_at    = EXCLUDED.last_refreshed_at
	`, p.TenantID, p.SchoolID, p.TermID)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleAttendanceTermRefresh: %w", err)
	}
	slog.InfoContext(ctx, "attendance: term summaries refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleClassDailyRefresh(ctx context.Context, t *asynq.Task) error {
	var p ClassDailyRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleClassDailyRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "attendance: refreshing class daily summary",
		"timetable_slot_id", p.TimetableSlotID, "date", p.Date,
	)

	// Resolve class_id from timetable slot and recompute daily summary
	_, err := w.pools.PG.Exec(ctx, `
		INSERT INTO class_daily_attendance_summaries (
			tenant_id, school_id, class_id, academic_term_id, date,
			total_enrolled, present_count, absent_count, late_count, excused_count,
			daily_attendance_rate, last_refreshed_at
		)
		SELECT
			$1 AS tenant_id,
			$2 AS school_id,
			c.id AS class_id,
			ar.academic_term_id,
			ar.date,
			COUNT(DISTINCT ar.student_id)::INT AS total_enrolled,
			COUNT(*) FILTER (WHERE ar.status = 'PRESENT')::INT AS present_count,
			COUNT(*) FILTER (WHERE ar.status = 'ABSENT')::INT AS absent_count,
			COUNT(*) FILTER (WHERE ar.status = 'LATE')::INT AS late_count,
			COUNT(*) FILTER (WHERE ar.status = 'EXCUSED')::INT AS excused_count,
			ROUND(
				(COUNT(*) FILTER (WHERE ar.status = 'PRESENT') * 100.0 / NULLIF(COUNT(*), 0)),
				2
			) AS daily_attendance_rate,
			NOW() AS last_refreshed_at
		FROM attendance_records ar
		JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
		JOIN cbc_classes c ON c.id = ts.class_id AND c.tenant_id = $1
		LEFT JOIN cbc_attendance_sessions s
			ON s.timetable_slot_id = ar.timetable_slot_id
			AND s.date = ar.date
			AND s.tenant_id = ar.tenant_id
		WHERE ar.tenant_id = $1 AND ar.school_id = $2
		  AND ar.timetable_slot_id = $3 AND ar.date = $4::DATE
		  AND (s.status IS NULL OR s.status != 'SKIPPED')
		GROUP BY c.id, ar.academic_term_id, ar.date
		ON CONFLICT (class_id, date)
		DO UPDATE SET
			academic_term_id    = EXCLUDED.academic_term_id,
			total_enrolled      = EXCLUDED.total_enrolled,
			present_count       = EXCLUDED.present_count,
			absent_count        = EXCLUDED.absent_count,
			late_count          = EXCLUDED.late_count,
			excused_count       = EXCLUDED.excused_count,
			daily_attendance_rate = EXCLUDED.daily_attendance_rate,
			last_refreshed_at   = NOW(),
			updated_at          = NOW()
	`, p.TenantID, p.SchoolID, p.TimetableSlotID, p.Date)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleClassDailyRefresh: %w", err)
	}
	slog.InfoContext(ctx, "attendance: class daily summary refreshed",
		"timetable_slot_id", p.TimetableSlotID, "date", p.Date,
		"duration", time.Since(start).String(),
	)
	return nil
}

func (w *Worker) handleTeacherWorkloadRefresh(ctx context.Context, t *asynq.Task) error {
	var p WorkloadRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherWorkloadRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	slog.InfoContext(ctx, "attendance: refreshing teacher workload summaries", "academic_year_id", p.AcademicYearID)

	_, err := w.pools.PG.Exec(ctx, `SELECT fn_compute_teacher_workload_summaries($1)`, p.AcademicYearID)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherWorkloadRefresh: %w", err)
	}
	slog.InfoContext(ctx, "attendance: teacher workload summaries refreshed",
		"academic_year_id", p.AcademicYearID, "duration", time.Since(start).String(),
	)
	return nil
}

// ─── Lifecycle Hooks ──────────────────────────────────────────────────────

// RegisterWorkerHooks registers lifecycle hooks for the background worker.
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: worker.Start,
		OnStop:  worker.Stop,
	})
}

// asynqLogger implements asynq.Logger via slog.
type asynqLogger struct{}

func (asynqLogger) Debug(args ...interface{}) { slog.Debug(fmt.Sprint(args...)) }
func (asynqLogger) Info(args ...interface{})  { slog.Info(fmt.Sprint(args...)) }
func (asynqLogger) Warn(args ...interface{})  { slog.Warn(fmt.Sprint(args...)) }
func (asynqLogger) Error(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
func (asynqLogger) Fatal(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
