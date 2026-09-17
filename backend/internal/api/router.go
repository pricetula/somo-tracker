package api

import (
	"github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware/captcha"
	"somotracker/backend/internal/api/middleware/csrf"
	"somotracker/backend/internal/api/middleware/ipblacklist"
	"somotracker/backend/internal/api/middleware/ratelimit"
	"somotracker/backend/internal/api/middleware/session"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/services"
	sessionpkg "somotracker/backend/internal/session"
)

// Rate limit tiers for auth endpoints.
// IP-based: first line of defense against distributed attacks.
// Email-based: stricter limit to prevent email bombing/enumeration.
var (
	authRateIP     = redis_rate.PerMinute(10) // 10 req/min per IP
	authRateEmail  = redis_rate.PerHour(3)    // 3 req/hour per email
	bulkInviteRate = redis_rate.PerHour(5)    // 5 bulk invites per hour per tenant
)

// Router wires all delivery-layer routes. It depends only on the service
// interfaces (not concrete implementations), which makes it fully testable
// with mock services.
type Router struct {
	Auth               *authHandler
	Me                 *meHandler
	School             *SchoolHandler
	SchoolCreate       *SchoolCreateHandler
	AcademicPeriod     *AcademicPeriodHandler
	Streams            *StreamsHandler
	Grades             *GradesHandler
	Classes            *ClassesHandler
	AdminInvitation    *AdminInvitationHandler
	Admins             *AdminsHandler
	Teachers           *TeachersHandler
	Finance            *FinanceHandler
	Guardians          *GuardiansHandler
	TeacherInvitation  *TeacherInvitationHandler
	FinanceInvitation  *FinanceInvitationHandler
	GuardianInvitation *GuardianInvitationHandler
	Timetable          *TimetableHandler
	limiter            *redis_rate.Limiter
	cfg                *config.Config
}

// NewRouter creates a Router from the injected services and the Redis
// rate-limiting limiter singleton. It also accepts a redis.Client for the
// session middleware that protects authenticated routes.
func NewRouter(
	authSvc services.AuthService,
	meSvc services.MeService,
	schoolSvc services.SchoolService,
	academicSvc services.AcademicPeriodService,
	streamsSvc services.StreamsService,
	gradesSvc services.GradesService,
	classesSvc services.ClassesService,
	adminsSvc services.AdminsService,
	teachersSvc services.TeachersService,
	financeSvc services.FinanceService,
	guardiansSvc services.GuardiansService,
	timetableSvc services.TimetableService,
	limiter *redis_rate.Limiter,
	cfg *config.Config,
) *Router {
	return &Router{
		Auth:               newAuthHandler(authSvc, cfg),
		Me:                 newMeHandler(meSvc),
		School:             NewSchoolHandler(&schoolSvc),
		SchoolCreate:       NewSchoolCreateHandler(&schoolSvc),
		AcademicPeriod:     NewAcademicPeriodHandler(academicSvc),
		Streams:            NewStreamsHandler(streamsSvc),
		Grades:             NewGradesHandler(gradesSvc),
		Classes:            NewClassesHandler(classesSvc, zap.L()),
		Admins:             NewAdminsHandler(adminsSvc, zap.L()),
		Teachers:           NewTeachersHandler(teachersSvc, zap.L()),
		Finance:            NewFinanceHandler(financeSvc, zap.L()),
		Guardians:          NewGuardiansHandler(guardiansSvc, zap.L()),
		Timetable:          NewTimetableHandler(timetableSvc, zap.L()),
		AdminInvitation:    nil,
		TeacherInvitation:  nil,
		FinanceInvitation:  nil,
		GuardianInvitation: nil,
		limiter:            limiter,
		cfg:                cfg,
	}
}

// RegisterRoutes attaches the grouped endpoints to the Fiber app.
// Routes are split into public (auth-related) and protected groups.
func (r *Router) RegisterRoutes(app *fiber.App, redisClient *redis.Client, logger *zap.Logger, pool *pgxpool.Pool) {
	// IP blacklist middleware - checks blacklist before any other processing.
	// Uses fail-open behavior: Redis errors allow request through.
	app.Use(ipblacklist.NewIPBlacklistMiddleware(redisClient, logger, ipblacklist.DefaultConfig()))

	// CAPTCHA middleware for abuse-prone endpoints.
	// Reads config to determine if enabled and which provider to use.
	captchaMW := captcha.NewCaptchaMiddleware(captcha.Config{
		Enabled:        r.cfg.CAPTCHAEnabled,
		ProviderName:   r.cfg.CAPTCHAProvider,
		SiteKey:        r.cfg.CAPTCHASiteKey,
		SecretKey:      r.cfg.CAPTCHASecretKey,
		ScoreThreshold: r.cfg.CAPTCHAScoreThreshold,
	}, logger)

	// ─── Public auth routes (no session, no CSRF) ──────────────────────
	// These are registered directly on the app with full paths to avoid
	// inheriting middleware from the protected group below.

	// Magic-link send: dual rate limits (IP + email) + optional CAPTCHA
	// IP limit: 10/min — catches distributed botnets
	// Email limit: 3/hour — prevents email bombing/enumeration per address
	// CAPTCHA: enabled via config (disabled by default for local dev)
	app.Post("/api/auth/magic-link/send",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:magic-link:ip"),
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateEmail, "api:auth:magic-link:email"),
		captchaMW,
		r.Auth.sendMagicLink,
	)

	app.Get("/api/auth/callback",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:callback"),
		r.Auth.callback,
	)

	app.Get("/api/auth/invite/callback",
		ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:invite:callback"),
		r.Auth.inviteCallback,
	)

	// Logout - requires session + CSRF (mutating request)
	app.Post("/api/auth/logout", ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:logout"), r.Auth.logout)

	// ─── Protected routes (session + CSRF) ─────────────────────────────
	// All routes under /api except the public auth endpoints above.
	protected := app.Group("/api",
		session.NewSessionMiddleware(redisClient, logger, r.cfg),
		csrf.NewCSRFMiddleware(),
	)

	// Protected resources — session middleware validates session cookie,
	// injects user_id and tenant_id into c.Locals, and binds RLS context.
	// CSRF middleware validates double-submit token on mutating requests.
	r.School.session = sessionpkg.NewStore(redisClient)
	r.SchoolCreate.session = sessionpkg.NewStore(redisClient)

	// Initialize the admin invitation handler.
	// Handler is pre-configured in newFiberApp with service/stytch/asynq dependencies.

	protected.Get("/me", r.Me.getMe)
	protected.Post("/school/register", r.School.RegisterSchool)
	protected.Get("/schools", r.School.ListSchools)
	protected.Post("/school", r.SchoolCreate.CreateSchool)
	protected.Post("/school/set-active", r.School.SetActiveSchool)
	protected.Post("/school/academic-period", r.AcademicPeriod.CreateAcademicPeriod)
	protected.Get("/school/streams", r.Streams.ListStreams)
	protected.Post("/school/streams", r.Streams.CreateStreams)
	protected.Get("/school/streams/:id", r.Streams.GetStream)
	protected.Patch("/school/streams/:id", r.Streams.UpdateStream)
	protected.Post("/school/streams/delete", r.Streams.DeleteStreams)
	protected.Get("/school/grades", r.Grades.GetGrades)
	protected.Get("/school/classes", r.Classes.ListClasses)
	protected.Get("/school/classes/:id", r.Classes.GetClass)
	protected.Post("/school/classes", r.Classes.CreateClass)
	protected.Get("/timetable/templates", r.Timetable.ListTemplates)
	protected.Post("/timetable/templates", r.Timetable.CreateTemplate)
	protected.Post("/timetable/setup", r.Timetable.SetupClassTimetableSlot)
	protected.Get("/timetable/templates/:id/slots", r.Timetable.ListTimeSlotsByTemplate)
	protected.Get("/admins", r.Admins.ListAdmins)
	protected.Delete("/admins", r.Admins.DeleteAdmins)
	protected.Get("/teachers", r.Teachers.ListTeachers)
	protected.Delete("/teachers", r.Teachers.DeleteTeachers)
	protected.Get("/finance", r.Finance.ListFinance)
	protected.Delete("/finance", r.Finance.DeleteFinance)
	protected.Get("/guardians", r.Guardians.ListGuardians)
	protected.Delete("/guardians", r.Guardians.DeleteGuardians)
	protected.Post("/admins/invitations", ratelimit.NewRateLimitMiddleware(r.limiter, bulkInviteRate, "api:admin:invite:tenant"), r.AdminInvitation.HandleInvites)
	protected.Get("/admins/invitations/jobs/:job_id", r.AdminInvitation.GetJob)
	protected.Post("/admins/invitations/jobs/:job_id/retry-failed", r.AdminInvitation.RetryFailed)
	protected.Get("/admins/invitations/jobs/:job_id/events", r.AdminInvitation.Events)
	// Teacher invitations
	protected.Post("/teachers/invitations", ratelimit.NewRateLimitMiddleware(r.limiter, bulkInviteRate, "api:teacher:invite:tenant"), r.TeacherInvitation.HandleInvites)
	protected.Get("/teachers/invitations/jobs/:job_id", r.TeacherInvitation.GetJob)
	protected.Post("/teachers/invitations/jobs/:job_id/retry-failed", r.TeacherInvitation.RetryFailed)
	protected.Get("/teachers/invitations/jobs/:job_id/events", r.TeacherInvitation.Events)
	// Finance invitations
	protected.Post("/finance/invitations", ratelimit.NewRateLimitMiddleware(r.limiter, bulkInviteRate, "api:finance:invite:tenant"), r.FinanceInvitation.HandleInvites)
	protected.Get("/finance/invitations/jobs/:job_id", r.FinanceInvitation.GetJob)
	protected.Post("/finance/invitations/jobs/:job_id/retry-failed", r.FinanceInvitation.RetryFailed)
	protected.Get("/finance/invitations/jobs/:job_id/events", r.FinanceInvitation.Events)
	// Guardian invitations
	protected.Post("/guardians/invitations", ratelimit.NewRateLimitMiddleware(r.limiter, bulkInviteRate, "api:guardian:invite:tenant"), r.GuardianInvitation.HandleInvites)
	protected.Get("/guardians/invitations/jobs/:job_id", r.GuardianInvitation.GetJob)
	protected.Post("/guardians/invitations/jobs/:job_id/retry-failed", r.GuardianInvitation.RetryFailed)
	protected.Get("/guardians/invitations/jobs/:job_id/events", r.GuardianInvitation.Events)

	// Subjects list for data table with infinite pagination
	protected.Get("/subjects", subjectsListHandler(pool))
	protected.Get("/subjects/:id", subjectsDetailHandler(pool))
	protected.Get("/topics", topicsListHandler(pool))
	protected.Get("/topics/:id", topicsDetailHandler(pool))
	protected.Get("/sub-topics", subTopicsListHandler(pool))
	protected.Get("/sub-topics/:id", subTopicsListHandler(pool))
}
