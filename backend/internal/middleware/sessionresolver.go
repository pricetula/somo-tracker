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

	"somotracker/backend/internal/database"
)

var (
	sessionGroup      singleflight.Group
	invalidCacheToken = []byte("INVALID")
)

type SessionInfo struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

// newSessionResolver loads and verifies sessions against Redis and Postgres.
func NewSessionResolver(pools *database.Pools) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !strings.HasPrefix(c.Path(), "/api/") {
			return c.Next()
		}

		token := c.Cookies("somo_sid")
		if token != "" {
			sess, err := resolveSession(c.UserContext(), pools, token)
			if err == nil && sess != nil {
				c.Locals("session", sess)
			}

			if err != nil {
				return err
			}
		}

		if schoolID := c.Cookies("somo_school_id"); schoolID != "" {
			c.Locals("active_school_id", schoolID)
			c.Locals("school_id", schoolID)
		}

		return c.Next()
	}
}

func resolveSession(ctx context.Context, pools *database.Pools, rawToken string) (*SessionInfo, error) {
	// 1. Hash raw token once in Go
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(sum[:])
	cacheKey := "session:" + tokenHash

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
	v, err, _ := sessionGroup.Do(tokenHash, func() (interface{}, error) {
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

func loadSessionFromDB(ctx context.Context, pools *database.Pools, tokenHash string) (*SessionInfo, error) {
	const query = `
		SELECT s.user_id, s.tenant_id,
		       COALESCE(
			       (SELECT role::text FROM memberships
			         WHERE user_id = s.user_id AND is_active = true
			         ORDER BY
			           CASE role
			             WHEN 'SYSTEM_ADMIN' THEN 1
			             WHEN 'SCHOOL_ADMIN' THEN 2
			             WHEN 'TEACHER' THEN 3
			             WHEN 'NURSE' THEN 4
			             WHEN 'FINANCE' THEN 5
			           END
			         LIMIT 1),
			       'TEACHER'
		       ) as role
		FROM sessions s
		WHERE s.token_hash = $1 AND s.expires_at > NOW()
	`

	var s SessionInfo
	err := pools.PG.QueryRow(ctx, query, tokenHash).Scan(&s.UserID, &s.TenantID, &s.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("middleware.loadSessionFromDB: %w", err)
	}
	return &s, nil
}
