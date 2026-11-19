package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/platform/db"
)

// Store wraps sqlc-generated queries.
type Store struct {
	q    *storegen.Queries
	pool *db.Pool
}

// New creates a store.
func New(q *storegen.Queries, pool *db.Pool) *Store {
	return &Store{q: q, pool: pool}
}

// GetUserByNormalizedEmail retrieves a user by normalized email.
func (s *Store) GetUserByNormalizedEmail(ctx context.Context, email string) (storegen.User, error) {
	return s.q.GetUserByNormalizedEmail(ctx, email)
}

// CreateLoginChallenge creates a new login challenge.
func (s *Store) CreateLoginChallenge(ctx context.Context, challengeID uuid.UUID, emailNormalized string, codeMac []byte, expiresAt time.Time) (storegen.LoginChallenge, error) {
	return s.q.CreateLoginChallenge(ctx, storegen.CreateLoginChallengeParams{
		ID:             challengeID,
		EmailNormalized: emailNormalized,
		CodeMac:         codeMac,
		ExpiresAt:       expiresAt,
	})
}

// GetLoginChallengeByID retrieves a challenge by ID.
func (s *Store) GetLoginChallengeByID(ctx context.Context, id uuid.UUID) (storegen.LoginChallenge, error) {
	return s.q.GetLoginChallengeByID(ctx, id)
}

// MarkChallengeUsed marks a challenge as used.
func (s *Store) MarkChallengeUsed(ctx context.Context, id uuid.UUID) error {
	return s.q.MarkChallengeUsed(ctx, id)
}

// CreateEmailOutbox creates an email outbox record.
func (s *Store) CreateEmailOutbox(ctx context.Context, challengeID uuid.UUID, encryptedPayload []byte) (storegen.EmailOutbox, error) {
	return s.q.CreateEmailOutbox(ctx, storegen.CreateEmailOutboxParams{
		ChallengeID:      challengeID,
		EncryptedPayload: encryptedPayload,
	})
}

// GetActiveSession retrieves an active session by token hash.
func (s *Store) GetActiveSession(ctx context.Context, tokenHash []byte) (storegen.Session, error) {
	return s.q.GetActiveSession(ctx, tokenHash)
}

// CreateSession creates a new session.
func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, expiresAt time.Time) (storegen.Session, error) {
	return s.q.CreateSession(ctx, storegen.CreateSessionParams{
		TokenHash:       tokenHash,
		UserID:          userID,
		OrganizationID:  orgID,
		ActiveSchoolID:  activeSchoolID,
		ExpiresAt:       expiresAt,
	})
}

// GetOrganizationsByUser lists organizations a user belongs to.
func (s *Store) GetOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.Organization, error) {
	return s.q.GetOrganizationsByUser(ctx, userID)
}

// GetSchoolsByUser lists schools a user has access to.
func (s *Store) GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error) {
	return s.q.GetSchoolsByUser(ctx, userID)
}
