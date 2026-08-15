package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/singleflight"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Session cookie names. The auth handler writes these; the resolver reads and
// validates them. Exported so the auth package aliases its own constants to
// these instead of duplicating the wire names.
const (
	SessionCookieName  = "somo_sid"
	RoleCookieName     = "somo_role"
	SchoolIDCookieName = "somo_school_id"
)

var (
	sessionGroup      singleflight.Group
	invalidCacheToken = []byte("INVALID")
	publicRoutes      = map[string]bool{
		"/api/auth/discover": true,
		"/api/auth/callback": true,
	}
)

// fingerprintsMatch reports whether the presented fingerprint matches the
// stored fingerprint, honoring both formats:
//
//   - v2 (current): "v2:" + hex(sha256(User-Agent + "|" + Accept-Language)).
//     IP is intentionally omitted so legitimate IP changes don't log users out.
//   - v1 (legacy): unprefixed hex(sha256(IP + "|" + User-Agent + "|" +
//     Accept-Language)). The original IP is unknown, so v1 sessions are treated
//     as matching only when the presented fingerprint is also non-v2 and equal
//     (they cannot be re-verified with the v2 scheme).
func fingerprintsMatch(presented, stored string) bool {
	if strings.HasPrefix(presented, "v2:") {
		if !strings.HasPrefix(stored, "v2:") {
			// Legacy v1 session – cannot re-verify; allow.
			return true
		}
		return stored == presented
	}
	// Fallback for v1 presented fingerprints (pre-deploy code paths).
	return stored == presented
}

// SessionInfo is the resolved session context attached to every /api request.
// UserID, TenantID and Role are the authorization core; SchoolID/Schools are
// the user's school context within their tenant (used to scope the untrusted
// somo_school_id cookie — C2); DeviceFingerprint is recorded at session
// creation and compared on resume in production only (C5).
type SessionInfo struct {
	UserID            string   `json:"user_id"`
	TenantID          string   `json:"tenant_id"`
	Role              string   `json:"role"`
	SchoolID          string   `json:"school_id"`          // authoritative active school (DB)
	Schools           []string `json:"schools"`            // school IDs with active memberships (within tenant)
	DeviceFingerprint string   `json:"device_fingerprint"` // recorded at session creation
}

// sessionTokenHash returns the hex SHA-256 digest of a raw session token. It
// is the canonical digest used for the sessions.token_hash DB column and as
// the suffix of the Redis session cache key (see SessionCacheKey).
func sessionTokenHash(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// SessionCacheKey returns the canonical Redis key used to cache session
// resolution results for a raw session token. All writers and invalidators of
// the session cache (middleware write-through, auth logout) must use this
// function so a logged-out session cannot survive in the cache (B2).
func SessionCacheKey(rawToken string) string {
	return "session:" + sessionTokenHash(rawToken)
}

// NewSessionResolver loads and verifies sessions against Redis and Postgres,
// then enforces two auth invariants per request:
//
//   - C5 (production only): the device fingerprint presented by this request
//     must match the one recorded when the session was created. A mismatch
//     means the cookie is being replayed from another device and is rejected
//     with 401. Skipped outside production so a single dev machine can act as
//     multiple users.
//
//   - C2: the somo_school_id cookie is an unsigned client hint, never an
//     authority. active_school_id/school_id locals are populated from the
//     user's DB memberships; the cookie is honored only when it names a school
//     the user is an active member of within their session tenant, otherwise
//     the DB active school is used. A forged cookie can therefore never scope
//     a handler to a school the user has no access to.
func NewSessionResolver(pools *database.Pools, cfg config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if !strings.HasPrefix(path, "/api/") || publicRoutes[path] {
			return c.Next()
		}

		token := c.Cookies(SessionCookieName)
		if token != "" {
			sess, err := resolveSession(c.UserContext(), pools, token)
			if err == nil && sess != nil {
				// C5: reject device-bound sessions resumed from a different device.
				if ferr := enforceDeviceFingerprint(c, cfg, sess); ferr != nil {
					return ferr
				}
				c.Locals("session", sess)
			}

			if err != nil {
				// Translate a missing/expired session row (ErrNotFound, already
				// negative-cached inside resolveSession) into a 401 instead of
				// letting it surface as a 404/500 through the global handler.
				if errors.Is(err, ErrNotFound) {
					return ErrUnauthorized
				}
				return err
			}
		}

		// C2: scope the school context to the session's memberships.
		c.Locals("active_school_id", "")
		c.Locals("school_id", "")
		if sess, ok := c.Locals("session").(*SessionInfo); ok && sess != nil {
			schoolID := sess.SchoolID
			if cookieSchool := c.Cookies(SchoolIDCookieName); cookieSchool != "" && containsString(sess.Schools, cookieSchool) {
				schoolID = cookieSchool
			}
			if schoolID != "" {
				c.Locals("active_school_id", schoolID)
				c.Locals("school_id", schoolID)
			}
		}

		return c.Next()
	}
}

// enforceDeviceFingerprint enforces device-bound sessions in production only
// (C5). The presented fingerprint is produced by NewDeviceFingerprinter and is
// versioned:
//
//   - v2 (current): "v2:" + hex(sha256(User-Agent + "|" + Accept-Language)).
//     IP is intentionally omitted so legitimate IP changes (mobile networks,
//     DHCP renewals, load-balancer rotation) don't log users out.
//   - v1 (legacy): unprefixed hex(sha256(IP + "|" + User-Agent + "|" +
//     Accept-Language)). We no longer know the original IP, so a v1 session
//     cannot be re-verified. To avoid logging everyone out during the
//     transition, v1 sessions are allowed to continue (same trade-off as a
//     session with no stored fingerprint).
//
// Both formats are 64-char hex hashes with no "|" in them, so we version via
// the prefix rather than by string-splitting the hash.
func enforceDeviceFingerprint(c *fiber.Ctx, cfg config.Config, sess *SessionInfo) error {
	if cfg.AppEnv != "production" {
		return nil
	}
	presented, _ := c.Locals("device_fingerprint").(string)
	if presented == "" || sess.DeviceFingerprint == "" {
		// No signals on this request, or a legacy session with no stored
		// fingerprint – skip the check.
		return nil
	}

	if fingerprintsMatch(presented, sess.DeviceFingerprint) {
		return nil
	}

	// C5 is a risk signal, not a gate, unless explicitly opted in.
	if !cfg.EnforceDeviceFingerprint {
		loggerFrom(c).Warnw("device fingerprint mismatch (C5) — allowed because enforcement is disabled",
			"user_id", sess.UserID,
			"request_id", c.GetRespHeader("X-Request-ID"))
		return nil
	}
	return ErrDeviceFingerprintMismatch
}

// containsString reports whether v is present in the slice (case-sensitive;
// school IDs are canonical lowercase UUIDs).
func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func resolveSession(ctx context.Context, pools *database.Pools, rawToken string) (*SessionInfo, error) {
	// 1. Hash raw token once in Go. The Redis cache key prefixes the digest
	//    with "session:" (SessionCacheKey) — the DB lookup must use the BARE
	//    digest against sessions.token_hash, never the prefixed key, or no
	//    row would ever match and every session would resolve to 401/500.
	tokenHash := sessionTokenHash(rawToken)
	cacheKey := SessionCacheKey(rawToken)

	// 2. Check Redis
	val, err := pools.Redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		if string(val) == string(invalidCacheToken) {
			return nil, ErrNotFound
		}
		var sess SessionInfo
		if json.Unmarshal(val, &sess) == nil {
			return &sess, nil
		}
	}

	// 3. Singleflight wrap to deduplicate simultaneous requests for same token
	v, err, _ := sessionGroup.Do(cacheKey, func() (interface{}, error) {
		return loadSessionFromDB(ctx, pools, tokenHash)
	})

	// Detach cancel context for background cache writes so client disconnects don't cancel cache persistence
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if errors.Is(err, ErrNotFound) {
		// Negative caching: Cache invalid token in Redis for 30 seconds to prevent DB pool exhaustion
		pools.Redis.Set(writeCtx, cacheKey, invalidCacheToken, 30*time.Second)
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	sess := v.(*SessionInfo)

	// 4. Cache valid session in Redis for 15 minutes
	if data, err := json.Marshal(sess); err == nil {
		pools.Redis.Set(writeCtx, cacheKey, data, 15*time.Minute)
	}

	return sess, nil
}

// loadSessionFromDB resolves a session row together with the user's
// authorization context in one round trip:
//
//   - role: the highest-ranked active membership role (scoped to the session's
//     tenant); NULL → the session has zero memberships and is forbidden (B3).
//   - school_id: the user's active school (member_active_school) falling back
//     to the first membership's school — the authoritative default school
//     context (C2).
//   - schools: every school the user has an active membership in, within the
//     session's tenant — the only school IDs a cookie may select (C2).
//   - device_fingerprint: recorded at session creation, compared on resume in
//     production (C5).
func loadSessionFromDB(ctx context.Context, pools *database.Pools, tokenHash string) (*SessionInfo, error) {
	// fn_resolve_session is SECURITY DEFINER: it resolves the session BEFORE
	// the tenant is known, so it must bypass tenant-scoped RLS. It returns the
	// same shape as the old inline query (role / school_id / schools).
	const query = `
		SELECT user_id, tenant_id, device_fingerprint, role, school_id, schools
		FROM fn_resolve_session($1)
	`

	var role *string
	var schoolID *string
	var schools []string
	var s SessionInfo
	err := pools.PG.QueryRow(ctx, query, tokenHash).
		Scan(&s.UserID, &s.TenantID, &s.DeviceFingerprint, &role, &schoolID, &schools)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("middleware.loadSessionFromDB: %w", err)
	}
	s.Schools = schools
	if schoolID != nil {
		s.SchoolID = *schoolID
	}

	// B3: a valid session for a user with ZERO active memberships must NOT be
	// silently granted a default role (previously COALESCE → 'TEACHER').
	// Without membership context there is nothing to authorize against, so the
	// request is rejected as forbidden — the user is authenticated but not
	// entitled to any tenant resource.
	if role == nil || *role == "" {
		return nil, ErrForbidden
	}
	s.Role = *role
	return &s, nil
}
