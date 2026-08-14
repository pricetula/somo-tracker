package middleware

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
)

func NewDeviceFingerprinter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// v2 fingerprint: hash of User-Agent + Accept-Language only (IP omitted
		// to allow for legitimate IP changes such as mobile networks, DHCP
		// renewals, or load-balancer IP rotation). The "v2:" prefix lets the
		// session resolver distinguish these from legacy v1 sessions, which
		// stored an unprefixed hash of IP|UA|Accept-Language that we can no
		// longer reproduce.
		h := sha256.New()
		h.Write([]byte(c.Get("User-Agent")))
		h.Write([]byte("|"))
		h.Write([]byte(c.Get("Accept-Language")))

		c.Locals("device_fingerprint", "v2:"+hex.EncodeToString(h.Sum(nil)))
		return c.Next()
	}
}
