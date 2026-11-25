package store

import (
	"context"
	"database/sql"
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
	row, err := s.q.GetUserByNormalizedEmail(ctx, email)
	if err != nil {
		return storegen.User{}, err
	}
	return storegen.User{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Email:           row.Email,
		NormalizedEmail: row.NormalizedEmail,
		Name:            row.Name,
		Status:          row.Status,
		EmailVerifiedAt: row.EmailVerifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       row.DeletedAt,
	}, nil
}

// CreateOrganization creates a new organization.
func (s *Store) CreateOrganization(ctx context.Context, name string) (storegen.Organization, error) {
	return s.q.CreateOrganization(ctx, name)
}

// CreateUser creates a new user with email_verified_at set (OTP-verified email).
func (s *Store) CreateUser(ctx context.Context, orgID uuid.UUID, email, normalizedEmail, name string) (storegen.User, error) {
	row, err := s.q.CreateUser(ctx, storegen.CreateUserParams{
		OrganizationID:  orgID,
		Email:           email,
		NormalizedEmail: normalizedEmail,
		Name:            sql.NullString{String: name, Valid: true},
	})
	if err != nil {
		return storegen.User{}, err
	}
	return storegen.User{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Email:           row.Email,
		NormalizedEmail: row.NormalizedEmail,
		Name:            row.Name,
		Status:          row.Status,
		EmailVerifiedAt: row.EmailVerifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       row.DeletedAt,
	}, nil
}

// CreateSchool creates a new school within an organization.
func (s *Store) CreateSchool(ctx context.Context, orgID uuid.UUID, name string, countryID, educationSystemID uuid.NullUUID) (storegen.School, error) {
	row, err := s.q.CreateSchool(ctx, storegen.CreateSchoolParams{
		OrganizationID:   orgID,
		Name:             name,
		CountryID:        countryID,
		EducationSystemID: educationSystemID,
	})
	if err != nil {
		return storegen.School{}, err
	}
	return storegen.School{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		Name:              row.Name,
		CountryID:         row.CountryID,
		EducationSystemID: row.EducationSystemID,
		Status:            row.Status,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		DeletedAt:         row.DeletedAt,
	}, nil
}

// CreateUserSchool links a user to a school as an owner within their organization.
func (s *Store) CreateUserSchool(ctx context.Context, orgID, userID, schoolID uuid.UUID) (storegen.UserSchool, error) {
	return s.q.CreateUserSchool(ctx, storegen.CreateUserSchoolParams{
		OrganizationID: orgID,
		UserID:         userID,
		SchoolID:       schoolID,
	})
}

// UpdateOrganization updates an organization's name and marks onboarding completed.
func (s *Store) UpdateOrganization(ctx context.Context, orgID uuid.UUID, name string) error {
	return s.q.UpdateOrganization(ctx, storegen.UpdateOrganizationParams{
		Name: name,
		ID:   orgID,
	})
}

// UpdateUser updates a user's display name.
func (s *Store) UpdateUser(ctx context.Context, userID uuid.UUID, name string) error {
	return s.q.UpdateUser(ctx, storegen.UpdateUserParams{
		Name: sql.NullString{String: name, Valid: true},
		ID:   userID,
	})
}

// UpdateSessionType upgrades all temporary sessions for a user to full.
func (s *Store) UpdateSessionType(ctx context.Context, userID uuid.UUID) error {
	return s.q.UpdateSessionType(ctx, userID)
}

// GetSessionByTokenHash retrieves a session by its token hash.
func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.Session, error) {
	row, err := s.q.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return storegen.Session{}, err
	}
	return storegen.Session{
		ID:               row.ID,
		TokenHash:        row.TokenHash,
		UserID:           row.UserID,
		OrganizationID:   row.OrganizationID,
		ActiveSchoolID:   row.ActiveSchoolID,
		SessionType:      row.SessionType,
		CreatedAt:        row.CreatedAt,
		LastSeenAt:       row.LastSeenAt,
		ExpiresAt:        row.ExpiresAt,
		RevokedAt:        row.RevokedAt,
		RevocationReason: row.RevocationReason,
	}, nil
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
	row, err := s.q.GetActiveSession(ctx, tokenHash)
	if err != nil {
		return storegen.Session{}, err
	}
	return storegen.Session{
		ID:               row.ID,
		TokenHash:        row.TokenHash,
		UserID:           row.UserID,
		OrganizationID:   row.OrganizationID,
		ActiveSchoolID:   row.ActiveSchoolID,
		CreatedAt:        row.CreatedAt,
		LastSeenAt:       row.LastSeenAt,
		ExpiresAt:        row.ExpiresAt,
		RevokedAt:        row.RevokedAt,
		RevocationReason: row.RevocationReason,
	}, nil
}

// CreateSession creates a new session.
func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userID, orgID uuid.UUID, activeSchoolID uuid.NullUUID, sessionType string, expiresAt time.Time) (storegen.Session, error) {
	row, err := s.q.CreateSession(ctx, storegen.CreateSessionParams{
		TokenHash:       tokenHash,
		UserID:          userID,
		OrganizationID:  orgID,
		ActiveSchoolID:  activeSchoolID,
		SessionType:     sessionType,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		return storegen.Session{}, err
	}
	return storegen.Session{
		ID:               row.ID,
		TokenHash:        row.TokenHash,
		UserID:           row.UserID,
		OrganizationID:   row.OrganizationID,
		ActiveSchoolID:   row.ActiveSchoolID,
		SessionType:      row.SessionType,
		CreatedAt:        row.CreatedAt,
		LastSeenAt:       row.LastSeenAt,
		ExpiresAt:        row.ExpiresAt,
		RevokedAt:        row.RevokedAt,
		RevocationReason: row.RevocationReason,
	}, nil
}

// GetOrganizationsByUser lists organizations a user belongs to.
func (s *Store) GetOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.Organization, error) {
	return s.q.GetOrganizationsByUser(ctx, userID)
}

// GetSchoolsByUser lists schools a user has access to.
func (s *Store) GetSchoolsByUser(ctx context.Context, userID uuid.UUID) ([]storegen.School, error) {
	rows, err := s.q.GetSchoolsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	schools := make([]storegen.School, len(rows))
	for i, row := range rows {
		schools[i] = storegen.School{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			Name:           row.Name,
			Status:         row.Status,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
			DeletedAt:      row.DeletedAt,
		}
	}
	return schools, nil
}

// InvalidateChallengesByEmail marks all unused challenges for an email as used.
func (s *Store) InvalidateChallengesByEmail(ctx context.Context, emailNormalized string) error {
	return s.q.InvalidateChallengesByEmail(ctx, emailNormalized)
}
