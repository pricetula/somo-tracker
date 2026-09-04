// Package session manages server-issued opaque session tokens backed by
// Redis. The raw Stytch session token is stored securely in Redis (mapped
// by our opaque cookie token) and is never returned to clients directly.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionData holds the session metadata cached in Redis for fast validation.
// The raw Stytch session token is included here so that subsequent auth
// checks can use it without hitting the database.
type SessionData struct {
	UserID          string    `json:"user_id"`
	TenantID        string    `json:"tenant_id"`
	StytchSessionID string    `json:"stytch_session_id"`
	ExpiresAt       time.Time `json:"expires_at"`
}

const (
	// sessionTTL in Redis is slightly shorter than DB expiry so that DB
	// always remains the source of truth; Redis is a fast cache.
	sessionTTL = 6 * time.Hour
)

// Store manages session data in Redis.
type Store struct {
	client *redis.Client
}

// NewStore creates a session store backed by the provided Redis client.
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// Cache writes the session metadata to Redis, using the opaque cookie
// token as the key. The raw Stytch session ID is included so that auth
// middleware can retrieve it quickly without a DB round-trip.
func (s *Store) Cache(ctx context.Context, token string, data SessionData) error {
	if s.client == nil {
		return fmt.Errorf("session.Cache: redis client is nil")
	}
	if token == "" {
		return fmt.Errorf("session.Cache: token is required")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session.Cache: marshal session data: %w", err)
	}

	if err := s.client.Set(ctx, sessionKey(token), payload, sessionTTL).Err(); err != nil {
		return fmt.Errorf("session.Cache: redis set: %w", err)
	}
	return nil
}

// Retrieve reads the session metadata from Redis by the opaque cookie token.
// It returns a wrapped error so callers can distinguish cache miss (not found)
// from other failures.
func (s *Store) Retrieve(ctx context.Context, token string) (SessionData, error) {
	if s.client == nil {
		return SessionData{}, fmt.Errorf("session.Retrieve: redis client is nil")
	}
	if token == "" {
		return SessionData{}, fmt.Errorf("session.Retrieve: token is required")
	}

	val, err := s.client.Get(ctx, sessionKey(token)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return SessionData{}, fmt.Errorf("session.Retrieve: session not found: %w", redis.Nil)
		}
		return SessionData{}, fmt.Errorf("session.Retrieve: redis get: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal(val, &data); err != nil {
		return SessionData{}, fmt.Errorf("session.Retrieve: unmarshal session data: %w", err)
	}
	return data, nil
}

// Delete removes the session from Redis (used on logout / revocation).
func (s *Store) Delete(ctx context.Context, token string) error {
	if s.client == nil {
		return fmt.Errorf("session.Delete: redis client is nil")
	}
	if err := s.client.Del(ctx, sessionKey(token)).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("session.Delete: redis del: %w", err)
	}
	return nil
}

func sessionKey(token string) string {
	return fmt.Sprintf("session:%s", token)
}
