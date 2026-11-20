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
	CreateOrganization(ctx context.Context, name string) (storegen.Organization, error)
	CreateUser(ctx context.Context, orgID uuid.UUID, email, normalizedEmail, name string) (storegen.User, error)
	CreateSchool(ctx context.Context, orgID uuid.UUID, name string) (storegen.School, error)
	UpdateOrganization(ctx context.Context, orgID uuid.UUID, name string) error
	UpdateUser(ctx context.Context, userID uuid.UUID, name string) error
	UpdateSessionType(ctx context.Context, userID uuid.UUID) error
	GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.Session, error)
	GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error)
	CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, expiresAt time.Time) (storegen.Session, error)
}
