// Package compression provides response compression middleware for the Somotracker API.
// It uses Fiber's native compression handler with Gzip and Brotli support.
package compression

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
)

// Middleware returns a Fiber handler that compresses responses using Gzip and Brotli.
// Default compression levels are used to reduce payload sizes for web and mobile clients
// without impacting JSON API contracts. The middleware is lightweight and safe for all routes.
func Middleware() fiber.Handler {
	return compress.New(compress.Config{
		// Use default compression level for balanced CPU vs size.
		Level: compress.LevelDefault,
		// Enable both Gzip and Brotli where supported by the client.
		// Fiber's middleware handles negotiation automatically.
	})
}
