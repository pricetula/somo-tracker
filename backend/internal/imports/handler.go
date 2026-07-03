package imports

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler — SSE Progress Stream
// ============================================================================

// Handler exposes import progress streaming over SSE.
type Handler struct {
	svc  *Service
	pool *database.Pools
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, pools *database.Pools) *Handler {
	return &Handler{svc: svc, pool: pools}
}

// RegisterRoutes mounts the generic import stream endpoint.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/imports/:job_id/stream", middleware.RequireAuth, h.StreamProgress)
}

// StreamProgress handles GET /imports/:job_id/stream — SSE endpoint.
// It first sends the current job state (for resume semantics), then subscribes
// to Redis Pub/Sub for live updates.
func (h *Handler) StreamProgress(c *fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	if jobIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "job_id is required",
		})
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid job_id format",
		})
	}

	// Verify the requesting user's tenant matches the job's tenant
	tenantID := c.Locals("tenant_id").(string)
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "forbidden",
			"message": "access denied",
		})
	}

	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		if isNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    "not_found",
				"message": "import job not found",
			})
		}
		return middleware.HTTPError(c, err)
	}

	// Tenant check
	if job.TenantID != tenantUUID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "forbidden",
			"message": "access denied",
		})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Build initial event payload
	initialEvent := ProgressEvent{
		JobID:            job.ID.String(),
		Status:           job.Status,
		TotalRecords:     job.TotalRecords,
		ProcessedRecords: job.ProcessedRecords,
		SuccessCount:     job.SuccessCount,
		FailedCount:      job.FailedCount,
		TotalChunks:      job.TotalChunks,
		ProcessedChunks:  job.ProcessedChunks,
	}

	// Determine if the job is already terminal
	isTerminal := job.Status == ImportJobStatusCompleted ||
		job.Status == ImportJobStatusCompletedWithErrors ||
		job.Status == ImportJobStatusFailed ||
		job.Status == ImportJobStatusCancelled

	// Use StreamWriter for proper SSE streaming
	fctx := c.Context()
	fctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial state immediately (resume semantics)
		if err := writeSSE(w, "state", initialEvent); err != nil {
			slog.Warn("imports.Handler.StreamProgress: write initial event", "error", err)
			return
		}

		// If already terminal, we're done
		if isTerminal {
			return
		}

		// Subscribe to Redis Pub/Sub for live updates
		pubsub := h.pool.Redis.Subscribe(fctx, ProgressChannel)
		defer func() {
			if err := pubsub.Close(); err != nil {
				slog.ErrorContext(fctx, "imports.Handler.StreamProgress: close pubsub", "error", err)
			}
		}()

		ch := pubsub.Channel(redis.WithChannelSize(100))

		for {
			select {
			case <-fctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var event ProgressEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					slog.Warn("imports.Handler.StreamProgress: unmarshal event",
						"error", err,
						"payload", msg.Payload,
					)
					continue
				}

				// Only forward events for this job
				if event.JobID != job.ID.String() {
					continue
				}

				if err := writeSSE(w, "progress", event); err != nil {
					return // client probably disconnected
				}

				// Stop on terminal events
				if event.Status == ImportJobStatusCompleted ||
					event.Status == ImportJobStatusCompletedWithErrors ||
					event.Status == ImportJobStatusFailed ||
					event.Status == ImportJobStatusCancelled {
					return
				}
			}
		}
	})

	return nil
}

// ============================================================================
// SSE helpers
// ============================================================================

func writeSSE(w *bufio.Writer, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal sse event: %w", err)
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
	_, err = w.WriteString(msg)
	if err != nil {
		return err
	}
	return w.Flush()
}
