package cohortpositions

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// ─── Service ──────────────────────────────────────────────────────────────

// Service handles business logic for cohort position computations.
// It orchestrates the batch refresh and read queries.
type Service struct {
	repo   Repository
	pools  *database.Pools
	logger *zap.SugaredLogger
}

// NewService creates a new cohort positions Service.
func NewService(repo Repository, pools *database.Pools, logger *zap.SugaredLogger) *Service {
	return &Service{repo: repo, pools: pools, logger: logger}
}

// RefreshTerm triggers a batch recomputation of cohort positions for all
// students in the given academic term.
func (s *Service) RefreshTerm(ctx context.Context, termID string) error {
	if termID == "" {
		return fmt.Errorf("cohortpositions.Service.RefreshTerm: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.RefreshTerm(ctx, termID)
}

// RefreshAllActiveTerms finds all active academic terms across the system and
// refreshes cohort positions for each. Used by the periodic background worker.
func (s *Service) RefreshAllActiveTerms(ctx context.Context) error {
	rows, err := s.pools.PG.Query(ctx, `
		SELECT id FROM academic_terms WHERE is_current = true
	`)
	if err != nil {
		return fmt.Errorf("cohortpositions.Service.RefreshAllActiveTerms: query active terms: %w", err)
	}
	defer rows.Close()

	var termIDs []string
	for rows.Next() {
		var termID string
		if err := rows.Scan(&termID); err != nil {
			return fmt.Errorf("cohortpositions.Service.RefreshAllActiveTerms: scan: %w", err)
		}
		termIDs = append(termIDs, termID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("cohortpositions.Service.RefreshAllActiveTerms: rows: %w", err)
	}

	if len(termIDs) == 0 {
		s.logger.Warnw("cohortpositions: no active terms found for batch refresh")
		return nil
	}

	for _, termID := range termIDs {
		start := time.Now()
		s.logger.Infow("cohortpositions: refreshing term",
			"term_id", termID,
		)
		if err := s.repo.RefreshTerm(ctx, termID); err != nil {
			// Log and continue — don't let one failing term block others.
			s.logger.Errorw("cohortpositions: term refresh failed",
				"term_id", termID,
				"error", err,
				"duration", time.Since(start).String(),
			)
			continue
		}
		s.logger.Infow("cohortpositions: term refreshed",
			"term_id", termID,
			"duration", time.Since(start).String(),
		)
	}

	return nil
}

// GetByStudentTerm returns the cohort position for a student in a term.
func (s *Service) GetByStudentTerm(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error) {
	if studentID == "" || termID == "" || tenantID == "" {
		return nil, fmt.Errorf("cohortpositions.Service.GetByStudentTerm: %w", ErrInvalidInput)
	}
	return s.repo.GetByStudentTerm(ctx, studentID, termID, tenantID)
}

// ListByClassTerm returns all cohort positions for a class in a term.
func (s *Service) ListByClassTerm(ctx context.Context, classID, termID, tenantID string) ([]CohortPositionSummary, error) {
	if classID == "" || termID == "" || tenantID == "" {
		return nil, fmt.Errorf("cohortpositions.Service.ListByClassTerm: %w", ErrInvalidInput)
	}
	return s.repo.ListByClassTerm(ctx, classID, termID, tenantID)
}

// ListByGradeTerm returns all cohort positions at a grade level in a term.
func (s *Service) ListByGradeTerm(ctx context.Context, schoolID, gradeLevel, termID, tenantID string) ([]CohortPositionSummary, error) {
	if schoolID == "" || gradeLevel == "" || termID == "" || tenantID == "" {
		return nil, fmt.Errorf("cohortpositions.Service.ListByGradeTerm: %w", ErrInvalidInput)
	}
	return s.repo.ListByGradeTerm(ctx, schoolID, gradeLevel, termID, tenantID)
}

// ─── Asynq Task Names ────────────────────────────────────────────────────

const (
	TaskRefreshCohortPositions = "cohortpositions:refresh"
)

// ─── Worker ───────────────────────────────────────────────────────────────

// Worker wraps an Asynq server for processing cohort position refresh tasks.
// The server is built with database.NewAsynqServer so the Redis connection
// options and zap log adapter are shared with every other worker.
type Worker struct {
	server *asynq.Server
	svc    *Service
	pools  *database.Pools
	logger *zap.SugaredLogger
}

// NewWorker creates a new Worker with the cohortpositions refresh handler.
func NewWorker(svc *Service, pools *database.Pools, logger *zap.SugaredLogger) *Worker {
	return &Worker{
		svc:    svc,
		pools:  pools,
		logger: logger,
	}
}

// Start starts the Asynq worker. Called via fx lifecycle.
func (w *Worker) Start(ctx context.Context) error {
	server := database.NewAsynqServer(w.pools, w.logger, asynq.Config{
		Concurrency: 1,
		Queues: map[string]int{
			"cohortpositions": 5,
		},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskRefreshCohortPositions, func(ctx context.Context, t *asynq.Task) error {
		return handleRefresh(ctx, w.svc, w.logger)
	})

	if err := server.Start(mux); err != nil {
		return fmt.Errorf("cohortpositions.Worker.Start: %w", err)
	}

	w.server = server
	w.logger.Infow("cohortpositions.Worker: asynq server started")
	return nil
}

// Stop gracefully shuts down the Asynq worker.
func (w *Worker) Stop(ctx context.Context) error {
	if w.server != nil {
		w.server.Shutdown()
	}
	w.logger.Infow("cohortpositions.Worker: asynq server stopped")
	return nil
}

// handleRefresh processes a cohort position refresh task. It finds all active
// terms and refreshes each one by calling the batch PL/pgSQL function.
func handleRefresh(ctx context.Context, svc *Service, logger *zap.SugaredLogger) error {
	start := time.Now()
	logger.Infow("cohortpositions: batch refresh starting")

	if err := svc.RefreshAllActiveTerms(ctx); err != nil {
		logger.Errorw("cohortpositions: batch refresh failed",
			"error", err,
			"duration", time.Since(start).String(),
		)
		return fmt.Errorf("cohortpositions.handleRefresh: %w", err)
	}

	logger.Infow("cohortpositions: batch refresh completed",
		"duration", time.Since(start).String(),
	)
	return nil
}

// ─── Refresh Scheduler ────────────────────────────────────────────────────

// RefreshScheduler manages the periodic cohort position refresh task.
// Uses Asynq's Scheduler to register a recurring task that runs the batch
// computation every 30 minutes during active grading windows.
// The scheduler is built with database.NewAsynqScheduler so the Redis
// connection options and zap log adapter are shared with every other worker.
type RefreshScheduler struct {
	pools     *database.Pools
	scheduler *asynq.Scheduler
	logger    *zap.SugaredLogger
}

// NewRefreshScheduler creates a new periodic refresh scheduler.
func NewRefreshScheduler(pools *database.Pools, logger *zap.SugaredLogger) *RefreshScheduler {
	return &RefreshScheduler{
		pools:  pools,
		logger: logger,
	}
}

// Start starts the periodic refresh scheduler. Called via fx lifecycle.
// Registers a recurring task that enqueues cohortpositions:refresh once
// every 30 minutes.
func (rs *RefreshScheduler) Start(ctx context.Context) error {
	scheduler := database.NewAsynqScheduler(rs.pools, rs.logger, nil)

	// Schedule the batch refresh every 30 minutes.
	task := asynq.NewTask(TaskRefreshCohortPositions, nil)
	entryID, err := scheduler.Register("*/30 * * * *", task)
	if err != nil {
		return fmt.Errorf("cohortpositions.RefreshScheduler: register refresh task: %w", err)
	}

	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("cohortpositions.RefreshScheduler: start: %w", err)
	}

	rs.scheduler = scheduler
	rs.logger.Infow("cohortpositions.RefreshScheduler: refresh task registered",
		"entry_id", entryID,
		"schedule", "*/30 * * * *",
	)
	return nil
}

// Stop stops the refresh scheduler.
func (rs *RefreshScheduler) Stop(ctx context.Context) error {
	if rs.scheduler != nil {
		rs.scheduler.Shutdown()
	}
	rs.logger.Infow("cohortpositions.RefreshScheduler: stopped")
	return nil
}

// ─── Lifecycle Hooks ──────────────────────────────────────────────────────

// RegisterWorkerHooks registers lifecycle hooks for the Asynq worker.
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: worker.Start,
		OnStop:  worker.Stop,
	})
}

// RegisterSchedulerHooks registers lifecycle hooks for the refresh scheduler.
func RegisterSchedulerHooks(lc fx.Lifecycle, rs *RefreshScheduler) {
	lc.Append(fx.Hook{
		OnStart: rs.Start,
		OnStop:  rs.Stop,
	})
}
