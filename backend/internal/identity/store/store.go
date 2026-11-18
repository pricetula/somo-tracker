package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/db"
)

// Store implements Repository for identity domain.
// SQL lives in queries/*.sql, compiled by sqlc into store/gen/ (never hand-edited).
type Store struct {
	q    *storegen.Queries
	pool *db.Pool
}

// New creates store.
func New(q *storegen.Queries, pool *db.Pool) *Store {
	return &Store{q: q, pool: pool}
}

// getQuerier returns a *storegen.Queries bound to the transaction from context
// if one exists (set by TxManager.Run via the request middleware). Otherwise it
// acquires a dedicated connection from the pool, sets the tenant GUC for RLS,
// and returns a release function that resets the GUC and releases the connection.
// This ensures the same connection is used for SET LOCAL and the query, which is
// required for RLS with a connection pool.
func (s *Store) getQuerier(ctx context.Context) (*storegen.Queries, func(), error) {
	if tx := db.GetTx(ctx); tx != nil {
		return storegen.New(tx), func() {}, nil
	}

	conn, err := s.pool.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}

	if err := db.SetTenantGUC(ctx, func(ctx context.Context, sql string, args ...any) (any, error) {
		return conn.ExecContext(ctx, sql, args...)
	}); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	q := storegen.New(conn)
	release := func() {
		_, _ = conn.ExecContext(context.Background(), "RESET app.current_tenant_id")
		_ = conn.Close()
	}
	return q, release, nil
}

// Conversion helpers
func toPendingOnboarding(row storegen.GetPendingOnboardingByEmailRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toPendingOnboardingByID(row storegen.GetPendingOnboardingByIDRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toPendingOnboardingUpsert(row storegen.UpsertPendingOnboardingRow) storegen.PendingOnboarding {
	return storegen.PendingOnboarding{
		ID:                      row.ID,
		VerifiedEmailNormalized: row.VerifiedEmailNormalized,
		Status:                  row.Status,
		CurrentStep:             row.CurrentStep,
		FormData:                row.FormData,
		StytchOrganizationID:    row.StytchOrganizationID,
		StytchMemberID:          row.StytchMemberID,
		TenantID:                row.TenantID,
		SchoolName:              row.SchoolName,
		IdempotencyKey:          row.IdempotencyKey,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		LastActivityAt:          row.LastActivityAt,
		DeletedAt:               sql.NullTime{},
	}
}

func toAppSession(row storegen.CreateAppSessionRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toAppSessionByToken(row storegen.GetAppSessionByTokenHashRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toAppSessionByID(row storegen.GetAppSessionByIDRow) storegen.AppSession {
	return storegen.AppSession{
		ID:                     row.ID,
		UserID:                 row.UserID,
		TenantID:               row.TenantID,
		TokenHash:              row.TokenHash,
		StytchMemberSessionID:  row.StytchMemberSessionID,
		CreatedAt:              row.CreatedAt,
		ExpiresAt:              row.ExpiresAt,
		RevokedAt:              row.RevokedAt,
		DeletedAt:              sql.NullTime{},
		LastSeenAt:             row.LastSeenAt,
	}
}

func toOnboardingTicketCreate(row storegen.CreateOnboardingTicketRow) storegen.OnboardingTicket {
	return storegen.OnboardingTicket{
		ID:                  row.ID,
		PendingOnboardingID: row.PendingOnboardingID,
		TicketHash:          row.TicketHash,
		IstCiphertext:       row.IstCiphertext,
		UsedAt:              row.UsedAt,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           time.Time{},
		DeletedAt:           sql.NullTime{},
		IstNonce:            nil,
		IstKeyVersion:       0,
	}
}

func toOnboardingTicket(row storegen.GetOnboardingTicketByHashRow) storegen.OnboardingTicket {
	return storegen.OnboardingTicket{
		ID:                  row.ID,
		PendingOnboardingID: row.PendingOnboardingID,
		TicketHash:          row.TicketHash,
		IstCiphertext:       row.IstCiphertext,
		UsedAt:              row.UsedAt,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           time.Time{},
		DeletedAt:           sql.NullTime{},
		IstNonce:            nil,
		IstKeyVersion:       0,
	}
}

func toUserByID(row storegen.GetUserByIDRow) storegen.User {
	return storegen.User{
		ID:              row.ID,
		TenantID:        row.TenantID,
		EmailNormalized: row.EmailNormalized,
		StytchMemberID:  row.StytchMemberID,
		Name:            row.Name,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		DeletedAt:       sql.NullTime{},
	}
}

// Pending onboarding
func (s *Store) GetPendingOnboardingByEmail(ctx context.Context, email string) (storegen.PendingOnboarding, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.PendingOnboarding{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetPendingOnboardingByEmail(ctx, email)
	return toPendingOnboarding(row), err
}

func (s *Store) GetPendingOnboardingByID(ctx context.Context, id uuid.UUID) (storegen.PendingOnboarding, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.PendingOnboarding{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetPendingOnboardingByID(ctx, id)
	return toPendingOnboardingByID(row), err
}

func (s *Store) UpsertPendingOnboarding(ctx context.Context, arg storegen.UpsertPendingOnboardingParams) (storegen.PendingOnboarding, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.PendingOnboarding{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.UpsertPendingOnboarding(ctx, arg)
	return toPendingOnboardingUpsert(row), err
}

func (s *Store) UpdatePendingOnboardingStep(ctx context.Context, arg storegen.UpdatePendingOnboardingStepParams) error {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return err
	}
	if release != nil {
		defer release()
	}
	return q.UpdatePendingOnboardingStep(ctx, arg)
}

// App session
func (s *Store) CreateAppSession(ctx context.Context, arg storegen.CreateAppSessionParams) (storegen.AppSession, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.AppSession{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.CreateAppSession(ctx, arg)
	return toAppSession(row), err
}

func (s *Store) GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.AppSession, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.AppSession{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetAppSessionByTokenHash(ctx, tokenHash)
	return toAppSessionByToken(row), err
}

func (s *Store) GetAppSessionByID(ctx context.Context, id uuid.UUID) (storegen.AppSession, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.AppSession{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetAppSessionByID(ctx, id)
	return toAppSessionByID(row), err
}

func (s *Store) RevokeAppSession(ctx context.Context, id uuid.UUID) error {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return err
	}
	if release != nil {
		defer release()
	}
	return q.RevokeAppSession(ctx, id)
}

func (s *Store) UpdateSessionLastSeen(ctx context.Context, sessionID uuid.UUID, idleThreshold time.Duration) error {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return err
	}
	if release != nil {
		defer release()
	}
	return q.UpdateSessionLastSeen(ctx, storegen.UpdateSessionLastSeenParams{
		ID:           sessionID,
		Column2:      idleThreshold,
	})
}

func (s *Store) GetSessionWithMemberships(ctx context.Context, tokenHash []byte) (storegen.GetSessionWithMembershipsRow, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.GetSessionWithMembershipsRow{}, err
	}
	if release != nil {
		defer release()
	}
	return q.GetSessionWithMemberships(ctx, tokenHash)
}

// Onboarding ticket
func (s *Store) CreateOnboardingTicket(ctx context.Context, arg storegen.CreateOnboardingTicketParams) (storegen.OnboardingTicket, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.OnboardingTicket{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.CreateOnboardingTicket(ctx, arg)
	return toOnboardingTicketCreate(row), err
}

func (s *Store) GetOnboardingTicketByHash(ctx context.Context, hash []byte) (storegen.OnboardingTicket, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.OnboardingTicket{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetOnboardingTicketByHash(ctx, hash)
	return toOnboardingTicket(row), err
}

// Session
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return storegen.User{}, err
	}
	if release != nil {
		defer release()
	}
	row, err := q.GetUserByID(ctx, id)
	return toUserByID(row), err
}

func (s *Store) GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	q, release, err := s.getQuerier(ctx)
	if err != nil {
		return "", err
	}
	if release != nil {
		defer release()
	}
	role, err := q.GetTenantMembershipRole(ctx, storegen.GetTenantMembershipRoleParams{
		ID:       userID,
		TenantID: tenantID,
	})
	return string(role), err
}
