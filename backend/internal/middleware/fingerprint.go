package middleware

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
)

// newDeviceFingerprinter generates a SHA-256 fingerprint from client request attributes.
func newDeviceFingerprinter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := sha256.New()
		h.Write([]byte(c.IP()))
		h.Write([]byte("|"))
		h.Write([]byte(c.Get("User-Agent")))
		h.Write([]byte("|"))
		h.Write([]byte(c.Get("Accept-Language")))

		c.Locals("device_fingerprint", hex.EncodeToString(h.Sum(nil)))
		return c.Next()
	}
}
