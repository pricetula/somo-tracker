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
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/assessment"
	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/billing"
	"somotracker/backend/internal/cbcclasses"
	"somotracker/backend/internal/cbcschools"
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
	"somotracker/backend/internal/utils"
)

// Global Fiber error handler registered in fiber.Config.
// This is the last-resort catcher for any error that escapes handler functions
// (including panics caught by Fiber's recover middleware).
// It logs with slog.ErrorContext and returns the standard error response body.
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

	// Default to internal_error
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
		auth.Module,
		cbcschools.Module,
		cbcclasses.Module,
		curriculum.Module,
		billing.Module,
		parents.Module,
		students.Module,
		teachers.Module,
		invitations.Module,
		members.Module,
		assessment.Module,
		imports.Module,

		// Cross-domain interface wiring: school resolver from members,
		// school creator from cbcschools, curriculum seeder for new schools.
		fx.Provide(
			func(repo members.Repository) invitations.SchoolResolver {
				return repo
			},
			func(repo cbcschools.Repository) auth.SchoolCreator {
				return repo
			},
			func(svc *academicyears.Service) auth.AcademicYearCreator {
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
		),

		fx.Provide(newLogger),
		fx.Invoke(runMigrations),
		fx.Invoke(registerApp),
		fx.Invoke(imports.RegisterWorkerHooks),
		fx.Invoke(consumeSafeClient),
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
	importsHandler *imports.Handler,
	invitationsHandler *invitations.Handler,
	membersHandler *members.Handler,
	curriculumHandler *curriculum.Handler,
	studentsHandler *students.Handler,
	parentsHandler *parents.Handler,
	teachersHandler *teachers.Handler,
	billingHandler *billing.Handler,
) {
	app := fiber.New(fiber.Config{
		AppName:      "somotracker",
		ErrorHandler: globalErrorHandler,
	})

	// Register Fiber's built-in recover middleware before all routes
	// so that handler panics are caught and routed to the error handler
	// rather than crashing the process.
	app.Use(fiberrecover.New())

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
			membersHandler.RegisterRoutes(app)
			invitationsHandler.RegisterRoutes(app)
			curriculumHandler.RegisterRoutes(app)
			importsHandler.RegisterRoutes(app)
			studentsHandler.RegisterRoutes(app)
			parentsHandler.RegisterRoutes(app)
			teachersHandler.RegisterRoutes(app)
			billingHandler.RegisterRoutes(app)

			// Start Fiber in a non-blocking goroutine
			go func() {
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
