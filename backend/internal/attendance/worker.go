package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
	"go.uber.org/zap"

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

	// TaskRefreshClassLearningAreaTermSummary refreshes
	// class_learning_area_term_summaries for a given (tenant, school, term)
	// after attendance_term_summaries is current. Chained from
	// handleAttendanceTermRefresh so the rollup runs *after* its source
	// table is up-to-date, not on an independent timer. Payload:
	// AttendanceTermRefreshPayload.
	TaskRefreshClassLearningAreaTermSummary = "attendance:refresh_class_learning_area_term_summary"

	// TaskRefreshClassTermSummary refreshes class_term_attendance_summaries
	// for a given (tenant, school, term) after class_daily_attendance_summaries
	// is current. Chained from handleClassDailyRefresh for the same reason.
	// Payload: AttendanceTermRefreshPayload.
	TaskRefreshClassTermSummary = "attendance:refresh_class_term_summary"
)

// ─── Task Payloads ────────────────────────────────────────────────────────

// DeliveryRefreshPayload is the payload for refreshing teacher delivery summaries.
type DeliveryRefreshPayload struct {
	TenantID string `json:"tenant_id"`
	TermID   string `json:"term_id"`
}

// WorkloadRefreshPayload is the payload for refreshing teacher workload summaries.
type WorkloadRefreshPayload struct {
	TenantID       string `json:"tenant_id"`
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

// ClassLearningAreaTermRefreshPayload is the payload for refreshing the
// class_learning_area_term_summaries rollup for a given (tenant, school,
// term). ClassID is optional; when empty, the rollup recomputes every class
// in the school for that term.
type ClassLearningAreaTermRefreshPayload struct {
	TenantID string `json:"tenant_id"`
	SchoolID string `json:"school_id"`
	TermID   string `json:"term_id"`
	ClassID  string `json:"class_id,omitempty"`
}

// ClassTermRefreshPayload is the payload for refreshing the
// class_term_attendance_summaries rollup for a given (tenant, school, term).
// ClassID is optional; when empty, the rollup recomputes every class in the
// school for that term.
type ClassTermRefreshPayload struct {
	TenantID string `json:"tenant_id"`
	SchoolID string `json:"school_id"`
	TermID   string `json:"term_id"`
	ClassID  string `json:"class_id,omitempty"`
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
	logger *zap.SugaredLogger
}

// NewEnqueuer creates a new Enqueuer.
func NewEnqueuer(client *asynq.Client, logger *zap.SugaredLogger) *Enqueuer {
	return &Enqueuer{client: client, logger: logger}
}

// EnqueueTeacherDeliveryRefresh enqueues a task to refresh teacher delivery
// summaries for the given term. Best-effort: logs failures but does not
// block the caller.
func (e *Enqueuer) EnqueueTeacherDeliveryRefresh(ctx context.Context, tenantID, termID string) {
	payload, _ := json.Marshal(DeliveryRefreshPayload{TenantID: tenantID, TermID: termID})
	task := asynq.NewTask(TaskRefreshTeacherDeliverySummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("attendance: enqueue teacher delivery refresh failed",
			"tenant_id", tenantID, "term_id", termID, "error", err,
		)
	}
}

// EnqueueTeacherWorkloadRefresh enqueues a task to refresh teacher workload
// summaries for the given academic year.
func (e *Enqueuer) EnqueueTeacherWorkloadRefresh(ctx context.Context, tenantID, academicYearID string) {
	payload, _ := json.Marshal(WorkloadRefreshPayload{TenantID: tenantID, AcademicYearID: academicYearID})
	task := asynq.NewTask(TaskRefreshTeacherWorkloadSummaries, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("attendance: enqueue teacher workload refresh failed",
			"tenant_id", tenantID, "academic_year_id", academicYearID, "error", err,
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
		e.logger.Warnw("attendance: enqueue term summary refresh failed",
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
		e.logger.Warnw("attendance: enqueue class daily refresh failed",
			"timetable_slot_id", timetableSlotID, "date", date, "error", err,
		)
	}
}

// EnqueueClassLearningAreaTermRefresh enqueues a task to refresh the
// class_learning_area_term_summaries rollup for the given (tenant, school,
// term). If classID is empty, the rollup recomputes every class in the
// school/term scope. Best-effort: logs failures but does not block.
func (e *Enqueuer) EnqueueClassLearningAreaTermRefresh(ctx context.Context, tenantID, schoolID, termID, classID string) {
	payload, _ := json.Marshal(ClassLearningAreaTermRefreshPayload{
		TenantID: tenantID,
		SchoolID: schoolID,
		TermID:   termID,
		ClassID:  classID,
	})
	task := asynq.NewTask(TaskRefreshClassLearningAreaTermSummary, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("attendance: enqueue class learning area term refresh failed",
			"tenant_id", tenantID, "school_id", schoolID, "term_id", termID,
			"class_id", classID, "error", err,
		)
	}
}

// EnqueueClassTermRefresh enqueues a task to refresh the
// class_term_attendance_summaries rollup for the given (tenant, school,
// term). If classID is empty, the rollup recomputes every class in the
// school/term scope. Best-effort: logs failures but does not block.
func (e *Enqueuer) EnqueueClassTermRefresh(ctx context.Context, tenantID, schoolID, termID, classID string) {
	payload, _ := json.Marshal(ClassTermRefreshPayload{
		TenantID: tenantID,
		SchoolID: schoolID,
		TermID:   termID,
		ClassID:  classID,
	})
	task := asynq.NewTask(TaskRefreshClassTermSummary, payload)
	if _, err := e.client.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("summaries")); err != nil {
		e.logger.Warnw("attendance: enqueue class term refresh failed",
			"tenant_id", tenantID, "school_id", schoolID, "term_id", termID,
			"class_id", classID, "error", err,
		)
	}
}

// ─── Worker ───────────────────────────────────────────────────────────────

// Worker processes background attendance summary refresh tasks.
type Worker struct {
	pools    *database.Pools
	enqueuer *Enqueuer
	server   *asynq.Server
	logger   *zap.SugaredLogger
}

// NewWorker creates a new background attendance summary refresh worker.
func NewWorker(pools *database.Pools, logger *zap.SugaredLogger) *Worker {
	return &Worker{pools: pools, logger: logger}
}

// SetEnqueuer injects the enqueuer used by upstream refresh handlers to
// chain dependent rollup tasks (class_learning_area_term_summaries and
// class_term_attendance_summaries) after the source tables are refreshed.
// Mirrors Service.SetEnqueuer so the Worker can fire-and-forget without
// taking a hard dependency on the Enqueuer at construction time.
func (w *Worker) SetEnqueuer(e *Enqueuer) {
	w.enqueuer = e
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	w.server = asynq.NewServer(
		asynq.RedisClientOpt{Addr: w.pools.Redis.Options().Addr},
		asynq.Config{
			Concurrency: 1,
			Queues:      map[string]int{"summaries": 10},
			Logger:      asynqLogger{logger: w.logger},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRefreshTeacherDeliverySummaries, w.withTenant(w.handleTeacherDeliveryRefresh))
	mux.HandleFunc(TaskRefreshTeacherWorkloadSummaries, w.withTenant(w.handleTeacherWorkloadRefresh))
	mux.HandleFunc(TaskRefreshAttendanceTermSummaries, w.withTenant(w.handleAttendanceTermRefresh))
	mux.HandleFunc(TaskRefreshClassDailySummary, w.withTenant(w.handleClassDailyRefresh))
	mux.HandleFunc(TaskRefreshClassLearningAreaTermSummary, w.withTenant(w.handleClassLearningAreaTermRefresh))
	mux.HandleFunc(TaskRefreshClassTermSummary, w.withTenant(w.handleClassTermRefresh))

	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("attendance.Worker.Start: %w", err)
	}
	w.logger.Infow("attendance.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	if w.server != nil {
		w.server.Shutdown()
	}
	w.logger.Infow("attendance.Worker: asynq server stopped")
	return nil
}

// ─── Task Handlers ────────────────────────────────────────────────────────

// withTenant wraps an asynq handler so its task runs with the payload's
// tenant scoped into RLS context (transaction-scoped GUC). Every refresh
// query reads/writes RLS-protected tables; without tenant context those
// queries would silently return zero rows once the app role is locked down.
func (w *Worker) withTenant(h func(ctx context.Context, t *asynq.Task) error) func(ctx context.Context, t *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var tmp struct {
			TenantID string `json:"tenant_id"`
		}
		if err := json.Unmarshal(t.Payload(), &tmp); err != nil {
			return fmt.Errorf("attendance.Worker.withTenant: unmarshal tenant: %w", err)
		}
		if tmp.TenantID == "" {
			return fmt.Errorf("attendance.Worker.withTenant: payload missing tenant_id")
		}
		tctx := database.WithTenantID(ctx, tmp.TenantID)
		tx, err := database.Begin(tctx, w.pools.PG)
		if err != nil {
			return fmt.Errorf("attendance.Worker.withTenant: begin: %w", err)
		}
		defer func() {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}()
		if err := h(database.WithTenantTx(tctx, tx), t); err != nil {
			return err
		}
		return tx.Commit(context.WithoutCancel(ctx))
	}
}

func (w *Worker) handleTeacherDeliveryRefresh(ctx context.Context, t *asynq.Task) error {
	var p DeliveryRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherDeliveryRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	w.logger.Infow("attendance: refreshing teacher delivery summaries", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `SELECT fn_compute_teacher_delivery_summaries($1)`, p.TermID)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherDeliveryRefresh: %w", err)
	}
	w.logger.Infow("attendance: teacher delivery summaries refreshed",
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
	w.logger.Infow("attendance: refreshing term summaries", "term_id", p.TermID)

	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `
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
	w.logger.Infow("attendance: term summaries refreshed",
		"term_id", p.TermID, "duration", time.Since(start).String(),
	)

	// Chain: enqueue the class-grain rollup so it runs *after* the student-grain
	// rollup is current. We pass ClassID empty here because the rollup is
	// recomputed for every class in the school/term in one pass — passing a
	// single class_id would risk skipping other classes whose per-student
	// summaries were also just updated.
	if w.enqueuer != nil {
		w.enqueuer.EnqueueClassLearningAreaTermRefresh(ctx, p.TenantID, p.SchoolID, p.TermID, "")
	}

	return nil
}

func (w *Worker) handleClassDailyRefresh(ctx context.Context, t *asynq.Task) error {
	var p ClassDailyRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleClassDailyRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	w.logger.Infow("attendance: refreshing class daily summary",
		"timetable_slot_id", p.TimetableSlotID, "date", p.Date,
	)

	// Resolve class_id from timetable slot and recompute daily summary
	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `
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
	w.logger.Infow("attendance: class daily summary refreshed",
		"timetable_slot_id", p.TimetableSlotID, "date", p.Date,
		"duration", time.Since(start).String(),
	)

	// Chain: enqueue the class-term rollup so it runs *after* this single
	// daily summary is current. We resolve the term containing this date
	// (within the given tenant/school) and enqueue the rollup scoped to the
	// single class the slot belongs to — that's enough scope because the
	// upsert is keyed on (class_id, academic_term_id) and the SQL aggregates
	// every daily row for that class/term, so it will pick up any other
	// already-fresh daily rows in the same class/term as well.
	var resolvedTermID string
	if err := database.FromContext(ctx, w.pools.PG).QueryRow(ctx, `
		SELECT id FROM academic_terms
		WHERE tenant_id = $1 AND school_id = $2
		  AND start_date <= $3::DATE AND end_date >= $3::DATE
		LIMIT 1
	`, p.TenantID, p.SchoolID, p.Date).Scan(&resolvedTermID); err == nil && resolvedTermID != "" {
		var resolvedClassID string
		if err := database.FromContext(ctx, w.pools.PG).QueryRow(ctx, `
			SELECT class_id FROM cbc_timetable_slots
			WHERE tenant_id = $1 AND id = $2
			LIMIT 1
		`, p.TenantID, p.TimetableSlotID).Scan(&resolvedClassID); err == nil && resolvedClassID != "" {
			if w.enqueuer != nil {
				w.enqueuer.EnqueueClassTermRefresh(ctx, p.TenantID, p.SchoolID, resolvedTermID, resolvedClassID)
			}
		} else if err != nil {
			w.logger.Warnw("attendance: resolve class_id for chained class-term refresh failed; skipping chain",
				"timetable_slot_id", p.TimetableSlotID, "error", err,
			)
		}
	} else if err != nil {
		w.logger.Warnw("attendance: resolve term for chained class-term refresh failed; skipping chain",
			"tenant_id", p.TenantID, "school_id", p.SchoolID, "date", p.Date, "error", err,
		)
	}

	return nil
}

// handleClassLearningAreaTermRefresh aggregates attendance_term_summaries
// (student-grain) into class_learning_area_term_summaries (class-grain).
//
// Class assignment decision (documented):
//
//	attendance_term_summaries is keyed by (student, term, learning_area)
//	and does NOT carry a class_id — students in CBC can change classes
//	mid-term. We resolve student → class via cbc_student_enrollments
//	for the given academic_term_id, which records the class the student
//	was enrolled in at term start. This is a "term-level snapshot" — it
//	does NOT attempt to track mid-term transfers. A student who changed
//	classes mid-term will have their attendance attributed entirely to
//	their term-start class for the full term. This is consistent with
//	the documented total_enrolled workaround on class_daily_attendance_summaries:
//	we accept a known limitation (enrollment snapshot ≠ day-by-day
//	enrollment) rather than trying to "fix" it here.
//
//	The alternative — using cbc_timetable_slots.class_id per lesson
//	(point-in-time) — would be more accurate for mid-term transfers but
//	requires a proportional split of per-student totals across multiple
//	classes, which is complex and error-prone. The term-start class
//	snapshot is simpler, deterministic, and sufficient for the report's
//	purpose of identifying which subjects have the worst attendance for
//	a class this term.
//
// SKIPPED sessions are excluded to mirror the upstream aggregation in
// handleAttendanceTermRefresh.
//
// Upserts on the unique grain (class_id, learning_area_id, academic_term_id);
// idempotent — a retried job for the same scope produces identical rows.
func (w *Worker) handleClassLearningAreaTermRefresh(ctx context.Context, t *asynq.Task) error {
	var p ClassLearningAreaTermRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleClassLearningAreaTermRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	w.logger.Infow("attendance: refreshing class learning area term summaries",
		"tenant_id", p.TenantID, "school_id", p.SchoolID, "term_id", p.TermID, "class_id", p.ClassID,
	)

	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `
		WITH scope AS (
			SELECT $1::UUID AS tenant_id, $2::UUID AS school_id, $3::UUID AS term_id, $4::UUID AS class_id
		)
		INSERT INTO class_learning_area_term_summaries (
			tenant_id, school_id, class_id, learning_area_id, academic_term_id,
			academic_year_id,
			students_included,
			periods_total, periods_present, periods_absent, periods_late, periods_excused,
			attendance_percentage, last_refreshed_at
		)
		SELECT
			ats.tenant_id,
			ats.school_id,
			e.class_id,
			ats.learning_area_id,
			ats.academic_term_id,
			ats.academic_year_id,
			COUNT(DISTINCT ats.student_id)::INT AS students_included,
			SUM(ats.periods_total)::INT       AS periods_total,
			SUM(ats.periods_present)::INT     AS periods_present,
			SUM(ats.periods_absent)::INT      AS periods_absent,
			SUM(ats.periods_late)::INT        AS periods_late,
			SUM(ats.periods_excused)::INT     AS periods_excused,
			CASE
				WHEN SUM(ats.periods_total) > 0
				THEN ROUND(
					(SUM(ats.periods_present)::NUMERIC / SUM(ats.periods_total)::NUMERIC) * 100,
					2
				)
				ELSE 0.00
			END AS attendance_percentage,
			NOW() AS last_refreshed_at
		FROM attendance_term_summaries ats
		JOIN cbc_student_enrollments e
			ON e.tenant_id = ats.tenant_id
			AND e.student_id = ats.student_id
			AND e.academic_term_id = ats.academic_term_id
			AND e.class_id IS NOT NULL
		CROSS JOIN scope sc
		WHERE ats.tenant_id = sc.tenant_id
		  AND ats.school_id = sc.school_id
		  AND ats.academic_term_id = sc.term_id
		  AND (sc.class_id IS NULL OR e.class_id = sc.class_id)
		GROUP BY
			ats.tenant_id, ats.school_id, e.class_id, ats.learning_area_id,
			ats.academic_term_id, ats.academic_year_id
		ON CONFLICT (class_id, learning_area_id, academic_term_id)
		DO UPDATE SET
			tenant_id             = EXCLUDED.tenant_id,
			school_id             = EXCLUDED.school_id,
			academic_year_id      = EXCLUDED.academic_year_id,
			students_included     = EXCLUDED.students_included,
			periods_total         = EXCLUDED.periods_total,
			periods_present       = EXCLUDED.periods_present,
			periods_absent        = EXCLUDED.periods_absent,
			periods_late          = EXCLUDED.periods_late,
			periods_excused       = EXCLUDED.periods_excused,
			attendance_percentage = EXCLUDED.attendance_percentage,
			last_refreshed_at     = EXCLUDED.last_refreshed_at
	`, p.TenantID, p.SchoolID, p.TermID, nullableUUID(p.ClassID))
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleClassLearningAreaTermRefresh: %w", err)
	}
	w.logger.Infow("attendance: class learning area term summaries refreshed",
		"tenant_id", p.TenantID, "school_id", p.SchoolID, "term_id", p.TermID,
		"class_id", p.ClassID, "duration", time.Since(start).String(),
	)
	return nil
}

// handleClassTermRefresh aggregates class_daily_attendance_summaries
// (daily-grain) into class_term_attendance_summaries (term-grain).
//
// Date range is resolved from academic_terms (start_date..end_date) — NOT
// from the MIN/MAX of class_daily_attendance_summaries.date, because we
// only want to roll up rows that fall inside the official term window.
// SKIPPED sessions are already excluded at the daily-table level (they are
// never inserted there), so no further filter is needed here.
//
// total_enrolled_avg inherits the documented limitation from
// class_daily_attendance_summaries.total_enrolled (per-day enrollment
// snapshot, not per-term enrolled). We do NOT attempt to fix that here.
//
// Upserts on (class_id, academic_term_id); idempotent.
func (w *Worker) handleClassTermRefresh(ctx context.Context, t *asynq.Task) error {
	var p ClassTermRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleClassTermRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	w.logger.Infow("attendance: refreshing class term attendance summaries",
		"tenant_id", p.TenantID, "school_id", p.SchoolID, "term_id", p.TermID, "class_id", p.ClassID,
	)

	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `
		WITH scope AS (
			SELECT $1::UUID AS tenant_id, $2::UUID AS school_id, $3::UUID AS term_id, $4::UUID AS class_id
		),
		term_window AS (
			SELECT tenant_id, school_id, id AS term_id, start_date, end_date, academic_year_id
			FROM academic_terms
			WHERE id = (SELECT term_id FROM scope)
		)
		INSERT INTO class_term_attendance_summaries (
			tenant_id, school_id, class_id, academic_term_id, academic_year_id,
			days_in_term,
			total_enrolled_avg,
			present_count, absent_count, late_count, excused_count,
			term_attendance_rate, last_refreshed_at
		)
		SELECT
			cds.tenant_id,
			cds.school_id,
			cds.class_id,
			cds.academic_term_id,
			tw.academic_year_id,
			COUNT(*)::INT AS days_in_term,
			ROUND(AVG(cds.total_enrolled)::NUMERIC, 2) AS total_enrolled_avg,
			SUM(cds.present_count)::INT  AS present_count,
			SUM(cds.absent_count)::INT   AS absent_count,
			SUM(cds.late_count)::INT     AS late_count,
			SUM(cds.excused_count)::INT  AS excused_count,
			CASE
				WHEN (SUM(cds.present_count) + SUM(cds.absent_count) + SUM(cds.late_count) + SUM(cds.excused_count)) > 0
				THEN ROUND(
					(
						SUM(cds.present_count)::NUMERIC
						/ (SUM(cds.present_count) + SUM(cds.absent_count) + SUM(cds.late_count) + SUM(cds.excused_count))::NUMERIC
					) * 100,
					2
				)
				ELSE 0.00
			END AS term_attendance_rate,
			NOW() AS last_refreshed_at
		FROM class_daily_attendance_summaries cds
		JOIN term_window tw
			ON tw.tenant_id = cds.tenant_id
			AND tw.school_id = cds.school_id
			AND tw.term_id  = cds.academic_term_id
			AND cds.date BETWEEN tw.start_date AND tw.end_date
		CROSS JOIN scope sc
		WHERE cds.tenant_id = sc.tenant_id
		  AND cds.school_id = sc.school_id
		  AND cds.academic_term_id = sc.term_id
		  AND (sc.class_id IS NULL OR cds.class_id = sc.class_id)
		GROUP BY
			cds.tenant_id, cds.school_id, cds.class_id, cds.academic_term_id, tw.academic_year_id
		ON CONFLICT (class_id, academic_term_id)
		DO UPDATE SET
			tenant_id            = EXCLUDED.tenant_id,
			school_id            = EXCLUDED.school_id,
			academic_year_id     = EXCLUDED.academic_year_id,
			days_in_term         = EXCLUDED.days_in_term,
			total_enrolled_avg   = EXCLUDED.total_enrolled_avg,
			present_count        = EXCLUDED.present_count,
			absent_count         = EXCLUDED.absent_count,
			late_count           = EXCLUDED.late_count,
			excused_count        = EXCLUDED.excused_count,
			term_attendance_rate = EXCLUDED.term_attendance_rate,
			last_refreshed_at    = EXCLUDED.last_refreshed_at
	`, p.TenantID, p.SchoolID, p.TermID, nullableUUID(p.ClassID))
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleClassTermRefresh: %w", err)
	}
	w.logger.Infow("attendance: class term attendance summaries refreshed",
		"tenant_id", p.TenantID, "school_id", p.SchoolID, "term_id", p.TermID,
		"class_id", p.ClassID, "duration", time.Since(start).String(),
	)
	return nil
}

// nullableUUID returns nil when s is empty (so the optional class_id scope
// filter becomes a no-op via IS NULL check), and a *string containing s
// otherwise. Used to bind an optional UUID parameter to a Postgres query
// that filters with `(sc.class_id IS NULL OR x.class_id = sc.class_id)`.
func nullableUUID(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (w *Worker) handleTeacherWorkloadRefresh(ctx context.Context, t *asynq.Task) error {
	var p WorkloadRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherWorkloadRefresh: unmarshal: %w", err)
	}
	start := time.Now()
	w.logger.Infow("attendance: refreshing teacher workload summaries", "academic_year_id", p.AcademicYearID)

	_, err := database.FromContext(ctx, w.pools.PG).Exec(ctx, `SELECT fn_compute_teacher_workload_summaries($1)`, p.AcademicYearID)
	if err != nil {
		return fmt.Errorf("attendance.Worker.handleTeacherWorkloadRefresh: %w", err)
	}
	w.logger.Infow("attendance: teacher workload summaries refreshed",
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

// asynqLogger implements asynq.Logger via zap.
type asynqLogger struct {
	logger *zap.SugaredLogger
}

func (l asynqLogger) Debug(args ...interface{}) { l.logger.Debug(fmt.Sprint(args...)) }
func (l asynqLogger) Info(args ...interface{})  { l.logger.Info(fmt.Sprint(args...)) }
func (l asynqLogger) Warn(args ...interface{})  { l.logger.Warn(fmt.Sprint(args...)) }
func (l asynqLogger) Error(args ...interface{}) { l.logger.Error(fmt.Sprint(args...)) }
func (l asynqLogger) Fatal(args ...interface{}) { l.logger.Error(fmt.Sprint(args...)) }
