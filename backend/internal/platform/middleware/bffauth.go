// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"crypto/subtle"
	"errors"
	"net"
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
)

// ErrMissingBFFSecret is returned when BFF service secret is missing.
var ErrMissingBFFSecret = errors.New("missing BFF service secret")

// ErrInvalidBFFSecret is returned when BFF service secret is invalid.
var ErrInvalidBFFSecret = errors.New("invalid BFF service secret")

// ErrMissingClientIP is returned when X-Client-IP header is missing after BFF verification.
var ErrMissingClientIP = errors.New("missing X-Client-IP header")

// ErrBFFCIDRMismatch is returned when client IP is not in trusted BFF CIDRs.
var ErrBFFCIDRMismatch = errors.New("client IP not in trusted BFF CIDRs")

// BFFAuthConfig holds configuration for BFF authentication middleware.
type BFFAuthConfig struct {
	ServiceSecret    string
	TrustedCIDRs     []*net.IPNet
	ExemptPaths      map[string]bool
	ClientIPHeader   string
	ServiceSecretHeader string
}

// DefaultBFFAuthConfig returns a default configuration.
func DefaultBFFAuthConfig(cfg *config.Config) *BFFAuthConfig {
	exempt := map[string]bool{
		"/healthz": true,
		"/readyz":  true,
		// Add webhook paths here if needed
	}

	var cidrs []*net.IPNet
	if cfg.TrustedBFFCIDRs != "" {
		for _, cidrStr := range strings.Split(cfg.TrustedBFFCIDRs, ",") {
			cidrStr = strings.TrimSpace(cidrStr)
			if cidrStr == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(cidrStr)
			if err == nil {
				cidrs = append(cidrs, ipNet)
			}
		}
	}

	return &BFFAuthConfig{
		ServiceSecret:       cfg.BFFServiceSecret,
		TrustedCIDRs:        cidrs,
		ExemptPaths:         exempt,
		ClientIPHeader:      "X-Client-IP",
		ServiceSecretHeader: "X-BFF-Service-Secret",
	}
}

// VerifyBFFServiceSecret returns middleware that verifies the BFF service secret.
// This MUST be the first middleware after exempt path checks to prevent
// direct public-URL bypass of the API.
func VerifyBFFServiceSecret(cfg *config.Config) fiber.Handler {
	authCfg := DefaultBFFAuthConfig(cfg)

	return func(c fiber.Ctx) error {
		// Skip exempt paths
		if authCfg.ExemptPaths[c.Path()] {
			return c.Next()
		}

		// In development without BFF secret configured, allow but warn
		if authCfg.ServiceSecret == "" && !cfg.IsProduction() {
			return c.Next()
		}

		// In production, secret is mandatory (validated at startup)
		if authCfg.ServiceSecret == "" {
			return apperr.New(apperr.INTERNAL_ERROR, "BFF service secret not configured")
		}

		provided := c.Get(authCfg.ServiceSecretHeader)
		if provided == "" {
			return apperr.New(apperr.FORBIDDEN, "missing BFF service credential")
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(provided), []byte(authCfg.ServiceSecret)) != 1 {
			return apperr.New(apperr.FORBIDDEN, "invalid BFF service credential")
		}

		return c.Next()
	}
}

// VerifyTrustedBFFCIDR returns middleware that verifies the client IP is in trusted BFF CIDRs.
// This is an optional additional layer of defense.
func VerifyTrustedBFFCIDR(cfg *config.Config) fiber.Handler {
	authCfg := DefaultBFFAuthConfig(cfg)

	// If no CIDRs configured, skip this check
	if len(authCfg.TrustedCIDRs) == 0 {
		return func(c fiber.Ctx) error { return c.Next() }
	}

	return func(c fiber.Ctx) error {
		// Skip exempt paths
		if authCfg.ExemptPaths[c.Path()] {
			return c.Next()
		}

		// Get the peer IP (the BFF's IP as seen by the API)
		peerIP := net.ParseIP(c.IP())
		if peerIP == nil {
			return apperr.New(apperr.FORBIDDEN, "unable to determine peer IP")
		}

		// Check if peer IP is in any trusted CIDR
		allowed := false
		for _, cidr := range authCfg.TrustedCIDRs {
			if cidr.Contains(peerIP) {
				allowed = true
				break
			}
		}

		if !allowed {
			return apperr.New(apperr.FORBIDDEN, "request from untrusted BFF IP")
		}

		return c.Next()
	}
}

// ExtractClientIP returns middleware that extracts and validates the client IP
// from the X-Client-IP header (set by the BFF after parsing X-Forwarded-For).
// The IP is stored in context for downstream rate limiters.
func ExtractClientIP(cfg *config.Config) fiber.Handler {
	authCfg := DefaultBFFAuthConfig(cfg)

	return func(c fiber.Ctx) error {
		// Skip exempt paths
		if authCfg.ExemptPaths[c.Path()] {
			return c.Next()
		}

		clientIP := c.Get(authCfg.ClientIPHeader)
		if clientIP == "" {
			return apperr.New(apperr.INVALID_REQUEST, "missing client IP")
		}

		// Parse and validate IP
		ip := net.ParseIP(clientIP)
		if ip == nil {
			return apperr.New(apperr.INVALID_REQUEST, "invalid client IP")
		}

		// Normalize: IPv4 -> /32, IPv6 -> /64
		normalized := normalizeIP(ip)

		// Store in context for rate limiters
		c.Locals("clientIP", normalized)
		c.Locals("clientIPRaw", clientIP)

		return c.Next()
	}
}

// normalizeIP normalizes an IP address for rate limiting keys.
// IPv4: masked to /32 (single host)
// IPv6: masked to /64 (typical residential prefix)
func normalizeIP(ip net.IP) string {
	if ip4 := ip.To4(); ip4 != nil {
		// IPv4: /32
		return ip4.String() + "/32"
	}
	// IPv6: /64 prefix
	ip16 := ip.To16()
	if ip16 == nil {
		return ip.String()
	}
	// Zero out lower 64 bits
	for i := 8; i < 16; i++ {
		ip16[i] = 0
	}
	return net.IP(ip16).String() + "/64"
}

// ClientIPFromContext retrieves the normalized client IP from context.
func ClientIPFromContext(c fiber.Ctx) string {
	if ip := c.Locals("clientIP"); ip != nil {
		return ip.(string)
	}
	return ""
}

// ClientIPRawFromContext retrieves the raw client IP from context.
func ClientIPRawFromContext(c fiber.Ctx) string {
	if ip := c.Locals("clientIPRaw"); ip != nil {
		return ip.(string)
	}
	return ""
}

// AddExemptPath adds a path to the exempt list (for webhooks, etc.)
func (a *BFFAuthConfig) AddExemptPath(path string) {
	a.ExemptPaths[path] = true
}