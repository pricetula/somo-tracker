package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"somotracker/backend/internal/config"
)

// registerCORS mounts the CORS middleware. It must be registered before any
// other middleware so preflight OPTIONS requests are handled first.
func registerCORS(app *fiber.App, cfg config.Config) {
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-CSRF-Token",
		AllowCredentials: true,
		MaxAge:           86400,
	}))
}
