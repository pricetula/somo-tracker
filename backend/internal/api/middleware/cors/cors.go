package cors

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"somotracker/backend/internal/config"
)

// Middleware returns a Fiber CORS handler wired to the typed Config.
// ALLOWED_ORIGINS is set on Doppler and is parsed by Config.AllowedOriginsList;
// we do not construct or modify it here.
func Middleware(cfg *config.Config) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOriginsList(),
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodDelete,
			fiber.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"X-Request-ID",
		},
		AllowCredentials: true,
	})
}
