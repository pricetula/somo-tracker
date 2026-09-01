package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/attendance"
	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/cbcclasses"
	"somotracker/backend/internal/cbcschools"
	"somotracker/backend/internal/cbcstreams"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/curriculum"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/invitations"
	"somotracker/backend/internal/logger"
	"somotracker/backend/internal/members"
	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/parents"
	"somotracker/backend/internal/students"
	"somotracker/backend/internal/teachers"
	"somotracker/backend/internal/telemetry"
	"somotracker/backend/internal/timetable"
	"somotracker/backend/internal/utils"
)

func main() {
	fx.New(
		// Global dependencies
		config.Module,
		logger.Module,
		database.Module,
		utils.Module,
		telemetry.Module,

		// Feature modules
		academicyears.Module,
		attendance.Module,
		auth.Module,
		cbcclasses.Module,
		cbcschools.Module,
		cbcstreams.Module,
		curriculum.Module,
		imports.Module,
		invitations.Module,
		members.Module,
		parents.Module,
		students.Module,
		teachers.Module,
		timetable.Module,

		// Background workers whose lifecycle hooks are not registered inside
		// their own modules — wired here so fx starts/stops them with the app.
		fx.Invoke(imports.RegisterWorkerHooks),
		fx.Invoke(imports.RegisterCleanupSchedulerHooks),

		// Entrypoint – must be wrapped in fx.Invoke
		fx.Invoke(func(
			lc fx.Lifecycle,
			cfg config.Config,
			pools *database.Pools,
			log *zap.Logger,
			academicyearshandler *academicyears.Handler,
			attendancehandler *attendance.Handler,
			authhandler *auth.Handler,
			cbcclasseshandler *cbcclasses.Handler,
			cbcschoolshandler *cbcschools.Handler,
			cbcstreamshandler *cbcstreams.Handler,
			curriculumhandler *curriculum.Handler,
			importshandler *imports.Handler,
			invitationshandler *invitations.Handler,
			membershandler *members.Handler,
			parentshandler *parents.Handler,
			studentshandler *students.Handler,
			teachershandler *teachers.Handler,
			timetablehandler *timetable.Handler,
		) {
			// Build Fiber app with the canonical error handler: ALL errors
			// (including Fiber's built-in 404/405) are mapped to the standard
			// {code, message, errors} JSON body via middleware.HTTPError.
			app := fiber.New(fiber.Config{
				ErrorHandler: middleware.HTTPError,
				AppName:      "somotracker-api",
				// Deliberate limits: 4 MB JSON body cap (import endpoints are
				// JSON job polling — no file uploads), plus server-level timeouts
				// so a hung client or handler can't hold connections open
				// indefinitely.
				BodyLimit:    4 * 1024 * 1024,
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			})

			// Health check
			app.Get("/health", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "ok"})
			})
			app.Get("/", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "ok"})
			})

			// Global security + context middleware (session resolver, CSRF guard,
			// rate limiters, device fingerprint). Must run before routes so that
			// middleware.RequireAuth (used by protected routes) can read the
			// resolved session from Locals (D1).
			middleware.Register(app, pools, cfg, log.Sugar())

			// Register all API routes.
			academicyearshandler.RegisterRoutes(app)
			attendancehandler.RegisterRoutes(app)
			authhandler.RegisterRoutes(app)
			cbcclasseshandler.RegisterRoutes(app)
			cbcschoolshandler.RegisterRoutes(app)
			cbcstreamshandler.RegisterRoutes(app)
			curriculumhandler.RegisterRoutes(app)
			importshandler.RegisterRoutes(app)
			invitationshandler.RegisterRoutes(app)
			membershandler.RegisterRoutes(app)
			parentshandler.RegisterRoutes(app)
			studentshandler.RegisterRoutes(app)
			teachershandler.RegisterRoutes(app)
			timetablehandler.RegisterRoutes(app)

			// Lifecycle hooks for non-blocking Fiber server
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Info("starting server", zap.String("port", cfg.Port))
					go func() {
						if err := app.Listen(":" + cfg.Port); err != nil {
							log.Error("server listen failed", zap.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Info("shutting down fiber server")
					return app.ShutdownWithContext(ctx)
				},
			})
		}),
	).Run()
}
