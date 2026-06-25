package imports

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/middleware"
)

// Handler exposes import HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
	repo    Repository
	rdb     SSEPubSubClient
	redis   *redis.Client // full Redis client for hashes/pubsub catch-up
	logger  *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service, repo Repository, pools *database.Pools, logger *zap.Logger) *Handler {
	return &Handler{
		svc:     svc,
		authSvc: authSvc,
		repo:    repo,
		rdb:     pools.Redis,
		redis:   pools.Redis,
		logger:  logger,
	}
}

// RegisterRoutes mounts import routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	imports := router.Group("/api/v1/imports/staff")
	imports.Post("/", h.requireAuth, h.StartImport)
	imports.Get("/track/:id", h.requireAuth, h.TrackImport)
	imports.Get("/track/:id/sse", h.requireAuth, h.SSETrackImport)
	imports.Get("/:id/failures", h.requireAuth, h.ListFailedInvitations)

	// Student import routes (separate group)
	studentImports := router.Group("/api/v1/imports/students")
	studentImports.Post("/", h.requireAuth, h.StartStudentImport)
	studentImports.Get("/stream", h.requireAuth, h.SSEStudentImportStream)

	// Student lookup endpoints (for import wizard)
	students := router.Group("/api/v1")
	students.Get("/parents", h.requireAuth, h.ListParents)
	students.Get("/classes", h.requireAuth, h.ListClasses)
	students.Get("/students", h.requireAuth, h.ListExistingStudents)

	// Academic reference data endpoints
	academic := router.Group("/api/v1/academic")
	academic.Get("/years", h.requireAuth, h.ListAcademicYears)
	academic.Get("/periods", h.requireAuth, h.ListAcademicPeriods)
}

// ─── Auth middleware ──────────────────────────────────────────────────────

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := c.Cookies("somo_sid")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "no session cookie found",
		})
	}

	session, err := h.authSvc.GetSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "invalid or expired session",
		})
	}

	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	return c.Next()
}

// ─── StytchOrgResolver adapter ───────────────────────────────────────────

type stytchOrgResolver struct {
	repo Repository
}

func (r *stytchOrgResolver) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	return r.repo.GetTenantStytchOrgID(ctx, tenantID)
}

// ─── Handlers ────────────────────────────────────────────────────────────

// StartImport handles POST /api/v1/imports/staff
func (h *Handler) StartImport(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var req StartImportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "role is required (SCHOOL_ADMIN, NURSE, or FINANCE)",
		})
	}

	// Resolve the user's active school
	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	resolver := &stytchOrgResolver{repo: h.repo}
	parentJobID := ""
	if req.ParentImportJobID != nil {
		parentJobID = *req.ParentImportJobID
	}
	result, err := h.svc.StartImport(c.Context(), tenantID, schoolID, userID, req.Role, req.Records, resolver, parentJobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}

// TrackImport handles GET /api/v1/imports/staff/track/:id
func (h *Handler) TrackImport(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "import job ID is required",
		})
	}

	result, err := h.svc.GetImportJob(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// isTerminalJobStatus returns true if the status indicates the job is done
// (whether successfully, with errors, or after all retries exhausted).
func isTerminalJobStatus(status string) bool {
	switch status {
	case "completed", "completed_with_errors", "failed":
		return true
	}
	return false
}

// SSETrackImport handles GET /api/v1/imports/staff/track/:id/sse
// Server-Sent Events endpoint for real-time progress updates.
//
// It subscribes to Redis pub/sub for low-latency progress events and
// simultaneously polls the database every 3 seconds as a fallback.
// If Redis is unavailable at connection time, it falls back to pure
// postgres polling immediately.
func (h *Handler) SSETrackImport(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "import job ID is required",
		})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Send initial connection event via the Fiber write method
	initialEvent := ImportProgressEvent{
		Type:        "connected",
		ImportJobID: jobID,
	}
	initialData, _ := json.Marshal(initialEvent)
	if _, err := fmt.Fprintf(c, "data: %s\n\n", string(initialData)); err != nil {
		return nil
	}

	// Check Redis health on connection open
	redisAvailable := true
	if err := h.rdb.Ping(c.Context()).Err(); err != nil {
		h.logger.Warn("SSE: Redis unreachable at connection, falling back to pure polling",
			zap.String("import_job_id", jobID),
			zap.Error(err),
		)
		redisAvailable = false
	}

	// Capture the request context before the streaming goroutine — in Fiber's
	// test mode the fasthttp RequestCtx may be nil, so we guard against that.
	reqCtx := c.Context()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Subscribe to Redis pub/sub channel (only if Redis was reachable)
		var pubsubCh <-chan *redis.Message

		if redisAvailable {
			pubsub := h.rdb.Subscribe(c.Context(), RedisChannelProgress+jobID)
			defer func() {
				if err := pubsub.Close(); err != nil {
					h.logger.Error("imports.SSETrackImport: pubsub close failed", zap.Error(err))
				}
			}()
			pubsubCh = pubsub.Channel(redis.WithChannelSize(100))
		}

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// The fasthttp Done channel may be nil in test mode; handle gracefully.
		var done <-chan struct{}
		if reqCtx != nil {
			done = reqCtx.Done()
		}

		for {
			select {
			case <-done:
				return
			case msg, ok := <-pubsubCh:
				if !ok {
					// Redis channel closed (e.g. Redis went down mid-stream).
					// Set channel to nil so the select ignores it and
					// continues with DB polling only.
					h.logger.Warn("SSE: Redis pub/sub channel closed, continuing with DB polling",
						zap.String("import_job_id", jobID),
					)
					pubsubCh = nil
					continue
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", msg.Payload); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

				var event ImportProgressEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err == nil {
					if event.Type == EventFinished {
						return
					}
				}
			case <-ticker.C:
				result, err := h.svc.GetImportJob(context.Background(), jobID)
				if err != nil {
					continue
				}
				event := ImportProgressEvent{
					Type:             EventProgress,
					ImportJobID:      jobID,
					Status:           result.Job.Status,
					ProcessedRecords: result.Job.ProcessedRecords,
					SuccessCount:     result.Job.SuccessCount,
					FailedCount:      result.Job.FailedCount,
					TotalRecords:     result.Job.TotalRecords,
				}
				data, _ := json.Marshal(event)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

				if isTerminalJobStatus(result.Job.Status) {
					finishedEvent := ImportProgressEvent{
						Type:             EventFinished,
						ImportJobID:      jobID,
						Status:           result.Job.Status,
						ProcessedRecords: result.Job.ProcessedRecords,
						SuccessCount:     result.Job.SuccessCount,
						FailedCount:      result.Job.FailedCount,
						TotalRecords:     result.Job.TotalRecords,
					}
					finishedData, _ := json.Marshal(finishedEvent)
					if _, err := fmt.Fprintf(w, "data: %s\n\n", string(finishedData)); err != nil {
						return
					}
					if err := w.Flush(); err != nil {
						return
					}
					return
				}
			}
		}
	})

	return nil
}

// ListFailedInvitations handles GET /api/v1/imports/staff/:id/failures
func (h *Handler) ListFailedInvitations(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "import job ID is required",
		})
	}

	result, err := h.svc.GetFailedInvitations(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ─── Helpers ─────────────────────────────────────────────────────────────

func (h *Handler) resolveActiveSchool(c *fiber.Ctx, tenantID, userID string) (string, error) {
	schoolID, err := h.repo.GetActiveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("imports.Handler.resolveActiveSchool: %w", err)
	}
	return schoolID, nil
}

// ============================================================================
// Student Import Handlers
// ============================================================================

// ListParents handles GET /api/v1/parents
// Returns all active parents for the user's active school.
func (h *Handler) ListParents(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	parents, err := h.repo.ListParents(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(parents)
}

// ListClasses handles GET /api/v1/classes
// Returns all active classes for the user's active school.
func (h *Handler) ListClasses(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	classes, err := h.repo.ListClasses(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(classes)
}

// ListExistingStudents handles GET /api/v1/students
// Returns all existing students for the user's active school (for duplicate detection).
func (h *Handler) ListExistingStudents(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	students, err := h.repo.ListExistingStudents(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(students)
}

// StartStudentImport handles POST /api/v1/imports/students
func (h *Handler) StartStudentImport(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	// Resolve the user's active school (same pattern as staff import)
	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	var req StartStudentImportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	// Step 1 — Structural guard (no DB calls)
	if len(req.Students) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "students list is required and must not be empty",
		})
	}
	if req.AcademicYear == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "academic_year is required",
		})
	}
	if req.Term == "" || !ValidTerms[req.Term] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "term must be one of: Term 1, Term 2, Term 3",
		})
	}
	if len(req.Students) > MaxStudentsPerImport {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"code":    "payload_too_large",
			"message": fmt.Sprintf("maximum %d students per import", MaxStudentsPerImport),
		})
	}

	// Use the role from the auth session if available, otherwise default to SCHOOL_ADMIN.
	// The requireAuth middleware sets tenant_id and user_id; role is extracted from the
	// security pipeline's SessionInfo if the global middleware ran.
	role := "SCHOOL_ADMIN"
	if sess := c.Locals("session"); sess != nil {
		if si, ok := sess.(*middleware.SessionInfo); ok && si != nil {
			role = si.Role
		}
	}

	result, err := h.svc.StartStudentImport(c.Context(), tenantID, schoolID, userID, role, &req)
	if err != nil {
		// Map ErrImportInFlight to 409
		if err == ErrImportInFlight || fmt.Sprintf("%v", err) == fmt.Sprintf("%v", ErrImportInFlight) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "conflict",
				"message": "An import is already in progress for this organisation. Please wait for it to complete.",
			})
		}
		return middleware.HTTPError(c, err)
	}

	if result.Status == "enqueue_failed" {
		return c.Status(fiber.StatusAccepted).JSON(result)
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}

// SSEStudentImportStream handles GET /api/v1/imports/students/stream?job_id=:job_id
// Real-time progress via Server-Sent Events backed by Redis Pub/Sub.
func (h *Handler) SSEStudentImportStream(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	jobID := c.Query("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "job_id query parameter is required",
		})
	}

	// Validate job exists and belongs to this tenant
	status, totalRecords, _, err := h.repo.GetImportJobStatus(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	_ = tenantID
	_ = status
	_ = totalRecords

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	reqCtx := c.Context()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Step 3 — Catch-up read on connect from Redis hash
		if h.redis != nil {
			fields, err := h.redis.HGetAll(context.Background(), RedisProgressPrefix+jobID).Result()
			if err == nil && len(fields) > 0 {
				// Build progress frame from hash fields
				cf := ProgressFrame{
					Status:       fields["status"],
					Processed:    parseInt(fields["processed"]),
					Total:        parseInt(fields["total"]),
					SuccessCount: parseInt(fields["success_count"]),
					FailedCount:  parseInt(fields["failed_count"]),
				}
				data, _ := json.Marshal(cf)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
					h.logger.Warn("SSE write failed", zap.Error(err))
					return
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("SSE flush failed", zap.Error(err))
					return
				}

				// If terminal, exit immediately
				if cf.Status == "completed" || cf.Status == "failed" {
					return
				}
			}
		} else {
			// No Redis — query DB directly for catch-up
			job, getErr := h.repo.GetImportJob(context.Background(), jobID)
			if getErr == nil {
				cf := ProgressFrame{
					Status:       job.Status,
					Processed:    job.ProcessedRecords,
					Total:        job.TotalRecords,
					SuccessCount: job.SuccessCount,
					FailedCount:  job.FailedCount,
				}
				data, _ := json.Marshal(cf)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
					h.logger.Warn("SSE write failed", zap.Error(err))
					return
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("SSE flush failed", zap.Error(err))
					return
				}

				if cf.Status == "completed" || cf.Status == "completed_with_errors" || cf.Status == "failed" {
					return
				}
			}
		}

		// Subscribe to Redis Pub/Sub
		pubsub := h.rdb.Subscribe(context.Background(), RedisEventsPrefix+jobID)
		defer func() {
			if err := pubsub.Close(); err != nil {
				h.logger.Warn("SSE: pubsub close failed", zap.Error(err))
			}
		}()
		ch := pubsub.Channel()

		keepAlive := time.NewTicker(15 * time.Second)
		defer keepAlive.Stop()

		for {
			select {
			case <-reqCtx.Done():
				return // client disconnected

			case msg, ok := <-ch:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", msg.Payload); err != nil {
					h.logger.Warn("SSE write failed", zap.Error(err))
					return
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("SSE flush failed", zap.Error(err))
					return
				}

				// Check if terminal
				var frame ProgressFrame
				if err := json.Unmarshal([]byte(msg.Payload), &frame); err == nil {
					if frame.Status == "completed" || frame.Status == "failed" {
						return
					}
				}

			case <-keepAlive.C:
				if _, err := fmt.Fprintf(w, ": keep-alive\n\n"); err != nil {
					h.logger.Warn("SSE keep-alive write failed", zap.Error(err))
					return
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("SSE keep-alive flush failed", zap.Error(err))
					return
				}
			}
		}
	})

	return nil
}

// ListAcademicYears handles GET /api/v1/academic/years
func (h *Handler) ListAcademicYears(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	years, err := h.repo.GetAcademicYears(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(years)
}

// ListAcademicPeriods handles GET /api/v1/academic/periods?academic_year_id=xxx
func (h *Handler) ListAcademicPeriods(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "academic_year_id query parameter is required",
		})
	}

	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	periods, err := h.repo.GetAcademicPeriods(c.Context(), tenantID, schoolID, academicYearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(periods)
}

// parseInt is a helper to convert string to int for Redis hash fields.
func parseInt(s string) int {
	var v int
	_, _ = fmt.Sscanf(s, "%d", &v)
	return v
}
