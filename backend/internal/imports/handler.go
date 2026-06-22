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
)

// Handler exposes import HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
	repo    Repository
	rdb     *redis.Client
	logger  *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service, repo Repository, pools *database.Pools, logger *zap.Logger) *Handler {
	return &Handler{
		svc:     svc,
		authSvc: authSvc,
		repo:    repo,
		rdb:     pools.Redis,
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
}

// ─── Auth middleware ──────────────────────────────────────────────────────

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := c.Cookies("somo_sid")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "no session cookie found",
		})
	}

	session, err := h.authSvc.GetSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "invalid or expired session",
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
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "role is required (SCHOOL_ADMIN, NURSE, or FINANCE)",
		})
	}

	// Resolve the user's active school
	schoolID, err := h.resolveActiveSchool(c, tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to resolve active school: " + err.Error(),
		})
	}

	resolver := &stytchOrgResolver{repo: h.repo}
	parentJobID := ""
	if req.ParentImportJobID != nil {
		parentJobID = *req.ParentImportJobID
	}
	result, err := h.svc.StartImport(c.Context(), tenantID, schoolID, userID, req.Role, req.Records, resolver, parentJobID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}

// TrackImport handles GET /api/v1/imports/staff/track/:id
func (h *Handler) TrackImport(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "import job ID is required",
		})
	}

	result, err := h.svc.GetImportJob(c.Context(), jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
			Error:   "not_found",
			Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// SSETrackImport handles GET /api/v1/imports/staff/track/:id/sse
// Server-Sent Events endpoint for real-time progress updates.
func (h *Handler) SSETrackImport(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "import job ID is required",
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
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Subscribe to Redis pub/sub channel
		pubsub := h.rdb.Subscribe(c.Context(), RedisChannelProgress+jobID)
		defer func() { _ = pubsub.Close() }()

		ch := pubsub.Channel(redis.WithChannelSize(100))

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		done := c.Context().Done()

		for {
			select {
			case <-done:
				return
			case msg, ok := <-ch:
				if !ok {
					return
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
				result, err := h.svc.GetImportJob(c.Context(), jobID)
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

				if result.Job.Status == "completed" || result.Job.Status == "completed_with_errors" {
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
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "import job ID is required",
		})
	}

	result, err := h.svc.GetFailedInvitations(c.Context(), jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
			Error:   "not_found",
			Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// ─── Helpers ─────────────────────────────────────────────────────────────

func (h *Handler) resolveActiveSchool(c *fiber.Ctx, tenantID, userID string) (string, error) {
	schoolID, err := h.repo.GetActiveSchoolID(c.Context(), tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve active school: %w", err)
	}
	return schoolID, nil
}
