package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/auth/store/gen"
)

// StoreInterface defines the methods the auth service needs from the store.
// *Store satisfies this interface automatically.
type StoreInterface interface {
	CreateLoginChallenge(ctx context.Context, challengeID uuid.UUID, emailNormalized string, codeMac []byte, expiresAt time.Time) (storegen.LoginChallenge, error)
	CreateEmailOutbox(ctx context.Context, challengeID uuid.UUID, encryptedPayload []byte) (storegen.EmailOutbox, error)
	GetLoginChallengeByID(ctx context.Context, id uuid.UUID) (storegen.LoginChallenge, error)
	MarkChallengeUsed(ctx context.Context, id uuid.UUID) error
	GetUserByNormalizedEmail(ctx context.Context, email string) (storegen.User, error)
	GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error)
	CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, expiresAt time.Time) (storegen.Session, error)
}
