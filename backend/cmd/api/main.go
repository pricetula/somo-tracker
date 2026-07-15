// @title           Somotracker API
// @version         0.1.0
// @description     REST API for the Somotracker educational analytics platform.
//
// @contact.name   Somotracker Team
//
// @license.name  Proprietary
//
// @host           localhost:3030
// @BasePath       /
//
// @tag.name       Auth
// @tag.description Authentication and session management endpoints
// @tag.name       Tenants
// @tag.description Tenant (school) management endpoints
// @tag.name       Education Systems
// @tag.description Education system (curriculum framework) endpoints
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/assessment"
	"somotracker/backend/internal/attendance"
	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/behavior"
	"somotracker/backend/internal/billing"
	"somotracker/backend/internal/cbcclasses"
	"somotracker/backend/internal/cbcschools"
	"somotracker/backend/internal/cbcstreams"
	"somotracker/backend/internal/cbctimetableslots"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/curriculum"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/invitations"
	"somotracker/backend/internal/members"
	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/parents"
	"somotracker/backend/internal/students"
	"somotracker/backend/internal/teachers"
	"somotracker/backend/internal/timetablestructure"
	"somotracker/backend/internal/utils"
)

// Global Fiber error handler registered in fiber.Config.
// This is the last-resort catcher for any error that escapes handler functions
// (including panics caught by Fiber's recover middleware).
// It logs with slog.ErrorContext and returns the canonical error response body.
// It handles both *fiber.Error (from Fiber routing) and middleware-returned
// sentinel errors (from RequireAuth, RequireRole) by matching against the
// middleware.Err* sentinels via errors.Is.
func globalErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var message string
	var errorCode string

	// Try to get status code from Fiber's built-in error type
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	// If not a *fiber.Error, match against middleware sentinels so that
	// middleware-returned errors (e.g. RequireAuth returning ErrUnauthorized)
	// are mapped to the correct HTTP status and canonical JSON body.
	if message == "" {
		switch {
		case errors.Is(err, middleware.ErrNotFound):
			code = fiber.StatusNotFound
			message = "the requested resource was not found"
			errorCode = "not_found"
		case errors.Is(err, middleware.ErrAlreadyExists):
			code = fiber.StatusConflict
			message = "the resource already exists"
			errorCode = "already_exists"
		case errors.Is(err, middleware.ErrInvalidInput):
			code = fiber.StatusBadRequest
			message = err.Error()
			errorCode = "invalid_input"
		case errors.Is(err, middleware.ErrUnauthorized):
			code = fiber.StatusUnauthorized
			message = "authentication required"
			errorCode = "unauthorized"
		case errors.Is(err, middleware.ErrForbidden):
			code = fiber.StatusForbidden
			message = "insufficient permissions"
			errorCode = "forbidden"
		case errors.Is(err, middleware.ErrConflict):
			code = fiber.StatusConflict
			message = "the resource was modified by another request"
			errorCode = "conflict"
		case errors.Is(err, context.Canceled):
			code = 499
			message = "the request was canceled"
			errorCode = "request_canceled"
		case errors.Is(err, context.DeadlineExceeded):
			code = fiber.StatusGatewayTimeout
			message = "the request timed out"
			errorCode = "timeout"
		}
	}

	// Default to internal_error for unrecognized errors
	if message == "" {
		message = "an unexpected error occurred"
	}
	if errorCode == "" {
		errorCode = "internal_error"
	}

	// Log the error
	slog.LogAttrs(c.Context(), slog.LevelError,
		"global error handler",
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", code),
		slog.String("error", err.Error()),
	)

	return c.Status(code).JSON(fiber.Map{
		"code":    errorCode,
		"message": message,
	})
}

// ── Cross-Domain Adapters ────────────────────────────────────────────────

// curriculumSchoolSeeder adapts *curriculum.SeedingService to the
// cbcschools.CurriculumSeeder interface used by school creation.
type curriculumSchoolSeeder struct {
	svc *curriculum.SeedingService
}

// SeedForSchool seeds the CBC curriculum for a newly created school using the
// embedded curriculum JSON files compiled into the binary.
func (a *curriculumSchoolSeeder) SeedForSchool(ctx context.Context, tenantID, schoolID string) error {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("curriculumSchoolSeeder.SeedForSchool: parse tenant_id: %w", err)
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return fmt.Errorf("curriculumSchoolSeeder.SeedForSchool: parse school_id: %w", err)
	}
	return a.svc.SeedSchoolCurriculumDefault(ctx, tenantUUID, schoolUUID)
}

func main() {
	fx.New(
		config.Module,
		database.Module,
		utils.Module,
		academicyears.Module,
		attendance.Module,
		auth.Module,
		behavior.Module,
		cbcschools.Module,
		cbcstreams.Module,
		cbcclasses.Module,
		curriculum.Module,
		billing.Module,
		parents.Module,
		students.Module,
		teachers.Module,
		timetablestructure.Module,
		invitations.Module,
		members.Module,
		assessment.Module,
		cbctimetableslots.Module,
		imports.Module,

		// Cross-domain interface wiring: school resolver from members,
		// school creator from cbcschools, curriculum seeder + academic year
		// seeder for new schools.
		fx.Provide(
			func(repo members.Repository) invitations.SchoolResolver {
				return repo
			},
			// Wire the full cbcschools.Service (CreateSchool) as the SchoolCreator
			// so that registering a new school seeds the CBC curriculum and
			// academic years, enrolls the creator with the correct role, and sets
			// the active school — instead of a bare repository INSERT.
			func(svc *cbcschools.Service) auth.SchoolCreator {
				return svc
			},
			func(repo curriculum.Repository) assessment.LearningAreaResolver {
				return repo
			},
			// When a school is created, automatically seed its CBC curriculum
			// from the embedded JSON files.
			func(seeder *curriculum.SeedingService) cbcschools.CurriculumSeeder {
				return &curriculumSchoolSeeder{svc: seeder}
			},
			// When a school is created, automatically set up the initial
			// academic year and three CBC terms.
			func(svc *academicyears.Service) cbcschools.AcademicYearSeeder {
				return svc
			},
			// Provide the auth repository as a UserSchoolEnroller so that
			// creating a school also enrolls the creator with the correct role
			// and sets it as their active school.
			func(repo auth.Repository) cbcschools.UserSchoolEnroller {
				return repo.(cbcschools.UserSchoolEnroller)
			},
			// Wire behavior notes provider into the students handler for the
			// student detail page.
			func(behSvc *behavior.Service) students.BehaviorNotesProvider {
				return func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]students.BehaviorNoteItem, error) {
					notes, err := behSvc.GetNotesByStudentTerm(ctx, tenantID, schoolID, studentID, termID)
					if err != nil {
						return nil, err
					}
					items := make([]students.BehaviorNoteItem, len(notes))
					for i, n := range notes {
						items[i] = students.BehaviorNoteItem{
							ID:           n.ID,
							CategoryName: n.CategoryName,
							Description:  n.Description,
							Date:         n.Date.Format("2006-01-02"),
							Status:       string(n.Status),
							IsUrgent:     n.IsUrgent,
						}
					}
					return items, nil
				}
			},
		),

		fx.Provide(newLogger),
		fx.Invoke(runMigrations),
		fx.Invoke(registerApp),
		fx.Invoke(imports.RegisterWorkerHooks),
		fx.Invoke(attendance.RegisterWorkerHooks),
		fx.Invoke(consumeSafeClient),
		// Wire behavior notes provider into students handler
		fx.Invoke(func(h *students.Handler, fn students.BehaviorNotesProvider) {
			h.SetBehaviorNotesProvider(fn)
		}),
	).Run()
}

func newLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func errToStatus(err error) string {
	if err == nil {
		return "healthy"
	}
	return "unhealthy: " + err.Error()
}

func consumeSafeClient(client *http.Client) {
	// intentional no-op: ensures the SSRF-safe client is wired into
	// the fx container so it is available to future consumers without
	// triggering an unused-provision warning.
}

// runMigrations applies pending database migrations before the HTTP server
// starts. Invoked via fx so it runs during the container startup phase, before
// any lifecycle OnStart hooks.
func runMigrations(cfg config.Config) error {
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		// Log with slog and return the error so fx refuses to start the app
		slog.Error("migration failed", "error", err)
		return err
	}
	return nil
}

func registerApp(
	lc fx.Lifecycle,
	cfg config.Config,
	pools *database.Pools,
	authHandler *auth.Handler,
	academicYearsHandler *academicyears.Handler,
	assessmentHandler *assessment.Handler,
	attendanceHandler *attendance.Handler,
	behaviorHandler *behavior.Handler,
	cbcschoolsHandler *cbcschools.Handler,
	cbcclassesHandler *cbcclasses.Handler,
	importsHandler *imports.Handler,
	cbcstreamsHandler *cbcstreams.Handler,
	cbctimetableslotsHandler *cbctimetableslots.Handler,
	invitationsHandler *invitations.Handler,
	membersHandler *members.Handler,
	curriculumHandler *curriculum.Handler,
	studentsHandler *students.Handler,
	parentsHandler *parents.Handler,
	teachersHandler *teachers.Handler,
	billingHandler *billing.Handler,
	timetablestructureHandler *timetablestructure.Handler,
) {
	app := fiber.New(fiber.Config{
		AppName:      "somotracker",
		ErrorHandler: globalErrorHandler,
	})

	// Recover middleware is registered inside middleware.Register() as part of
	// the security pipeline (Layer 1). It is NOT duplicated here.

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Mount the full security middleware pipeline
			middleware.Register(app, pools, cfg)

			// Register global health endpoint
			app.Get("/health", func(c *fiber.Ctx) error {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				pgErr := pools.PG.Ping(ctx)
				redErr := pools.Redis.Ping(ctx).Err()
				return c.JSON(fiber.Map{
					"status":   "ok",
					"postgres": errToStatus(pgErr),
					"redis":    errToStatus(redErr),
					"env":      cfg.AppEnv,
				})
			})

			// Mount domain routes
			authHandler.RegisterRoutes(app)
			academicYearsHandler.RegisterRoutes(app)
			assessmentHandler.RegisterRoutes(app)
			cbcschoolsHandler.RegisterRoutes(app)
			cbcstreamsHandler.RegisterRoutes(app)
			cbctimetableslotsHandler.RegisterRoutes(app)
			cbcclassesHandler.RegisterRoutes(app)
			membersHandler.RegisterRoutes(app)
			invitationsHandler.RegisterRoutes(app)
			curriculumHandler.RegisterRoutes(app)
			importsHandler.RegisterRoutes(app)
			studentsHandler.RegisterRoutes(app)
			parentsHandler.RegisterRoutes(app)
			teachersHandler.RegisterRoutes(app)
			billingHandler.RegisterRoutes(app)
			attendanceHandler.RegisterRoutes(app)
			behaviorHandler.RegisterRoutes(app)
			timetablestructureHandler.RegisterRoutes(app)

			// Start Fiber in a non-blocking goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.ErrorContext(ctx, "fiber listen panic", slog.Any("recover", r))
					}
				}()
				if err := app.Listen(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
					// Log fatal since this means the server failed to start
					slog.Error("fiber listen fatal", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Bounded shutdown window: 15 seconds total
			shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			var shutdownErr error

			// 1. Gracefully drain Fiber (in-flight requests)
			if err := app.ShutdownWithContext(shutdownCtx); err != nil {
				shutdownErr = errors.Join(shutdownErr, err)
			}

			// 2. Close Postgres pool
			pools.PG.Close()

			// 3. Close Redis client
			if err := pools.Redis.Close(); err != nil {
				shutdownErr = errors.Join(shutdownErr, err)
			}

			if shutdownErr != nil {
				slog.ErrorContext(ctx, "registerApp.OnStop: shutdown error", "error", shutdownErr)
			}

			return shutdownErr
		},
	})
}
