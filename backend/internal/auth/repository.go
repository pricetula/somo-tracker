package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// ============================================================================
// SqlcRepository — concrete PostgreSQL implementation of Repository.
// Uses raw pgx queries (sqlc- generated queries would replace this).
// ============================================================================

// SqlcRepository implements Repository backed by PostgreSQL.
type SqlcRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	cfg    config.Config
}

// NewSqlcRepository creates a new SqlcRepository with fx lifecycle hooks.
func NewSqlcRepository(lc fx.Lifecycle, pools *database.Pools, logger *zap.Logger, cfg config.Config) *SqlcRepository {
	// Register lifecycle hooks (requirement 15)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pools.PG.Ping(ctx); err != nil {
				return fmt.Errorf("auth repository: postgres ping failed: %w", err)
			}
			logger.Info("auth repository: postgres connection verified")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			pools.PG.Close()
			logger.Info("auth repository: postgres pool closed")
			return nil
		},
	})

	return &SqlcRepository{
		pool:   pools.PG,
		logger: logger,
		cfg:    cfg,
	}
}

// TenantExists checks if a tenant exists with the given Stytch org ID.
func (r *SqlcRepository) TenantExists(ctx context.Context, orgID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM tenants WHERE stytch_org_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w: check tenant exists: %v", ErrInternal, err)
	}
	return exists, nil
}

// TenantExistsByName checks if a tenant exists with the given school name.
func (r *SqlcRepository) TenantExistsByName(ctx context.Context, name string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM tenants WHERE name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w: check tenant exists by name: %v", ErrInternal, err)
	}
	return exists, nil
}

// UserExistsByExternalID checks if a user exists with the given Stytch user ID.
func (r *SqlcRepository) UserExistsByExternalID(ctx context.Context, externalAuthID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE external_auth_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, externalAuthID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w: check user exists: %v", ErrInternal, err)
	}
	return exists, nil
}

// CreateTenant creates a new tenant and returns its ID.
func (r *SqlcRepository) CreateTenant(ctx context.Context, params CreateTenantParams) (string, error) {
	const query = `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.Name, params.Slug, params.StytchOrgID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("%w: create tenant: %v", ErrInternal, err)
	}
	return id, nil
}

// CreateUser creates a new user and returns its ID.
func (r *SqlcRepository) CreateUser(ctx context.Context, params CreateUserParams) (string, error) {
	const query = `
		INSERT INTO users (email, tenant_id, first_name, last_name, external_auth_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, params.Email, params.TenantID, params.FirstName, params.LastName, params.ExternalAuthID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("%w: create user: %v", ErrInternal, err)
	}
	return id, nil
}

// CreateSession persists a session record.
func (r *SqlcRepository) CreateSession(ctx context.Context, params CreateSessionParams) error {
	const query = `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, stytch_session_token, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		params.Token,
		params.UserID,
		params.TenantID,
		params.StytchMemberID,
		params.StytchOrgID,
		params.StytchSessionToken,
		params.DeviceFingerprint,
		params.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("%w: create session: %v", ErrInternal, err)
	}
	return nil
}

// GetSessionByToken retrieves a session by its opaque token.
func (r *SqlcRepository) GetSessionByToken(ctx context.Context, token string) (*UserSession, error) {
	const query = `
		SELECT id, token, user_id, tenant_id, stytch_member_id, stytch_org_id,
		       COALESCE(stytch_session_token, '') as stytch_session_token,
		       COALESCE(device_fingerprint, '') as device_fingerprint,
		       expires_at, created_at
		FROM sessions
		WHERE token = $1 AND expires_at > NOW()
	`
	var s UserSession
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.Token, &s.UserID, &s.TenantID,
		&s.StytchMemberID, &s.StytchOrgID, &s.StytchSessionToken,
		&s.DeviceFingerprint, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: session not found", ErrNotFound)
		}
		return nil, fmt.Errorf("%w: get session: %v", ErrInternal, err)
	}
	return &s, nil
}

// DeleteSession removes a session record by token.
func (r *SqlcRepository) DeleteSession(ctx context.Context, token string) error {
	const query = `DELETE FROM sessions WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("%w: delete session: %v", ErrInternal, err)
	}
	return nil
}

// CreateTenantUserSession creates tenant, user, and session inside a single
// database transaction (requirement 9). Returns the user ID and tenant ID.
// On failure, if a stytch_org_id is present in the context, it logs a WARN
// for manual reconciliation.
func (r *SqlcRepository) CreateTenantUserSession(
	ctx context.Context,
	tenantParams CreateTenantParams,
	userParams CreateUserParams,
	sessionParams CreateSessionParams,
) (userID string, tenantID string, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", fmt.Errorf("%w: begin tx: %v", ErrInternal, err)
	}
	defer func() {
		if err != nil {
			// Log the stytch_org_id at WARN level if present
			if orgID, ok := ctx.Value(StytchOrgIDKey{}).(string); ok && orgID != "" {
				r.logger.Warn("Postgres write failed after Stytch org creation — manual reconciliation required",
					zap.String("stytch_org_id", orgID),
					zap.Error(err),
				)
			}
			_ = tx.Rollback(ctx)
		}
	}()

	// 1. Insert tenant
	tenantQuery := `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err = tx.QueryRow(ctx, tenantQuery, tenantParams.Name, tenantParams.Slug, tenantParams.StytchOrgID).Scan(&tenantID)
	if err != nil {
		return "", "", fmt.Errorf("%w: create tenant in tx: %v", ErrInternal, err)
	}

	// 2. Insert user
	userQuery := `
		INSERT INTO users (email, tenant_id, first_name, last_name, external_auth_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = tx.QueryRow(ctx, userQuery,
		userParams.Email, tenantID, userParams.FirstName, userParams.LastName, userParams.ExternalAuthID,
	).Scan(&userID)
	if err != nil {
		return "", "", fmt.Errorf("%w: create user in tx: %v", ErrInternal, err)
	}

	// 3. Insert session
	sessionQuery := `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, stytch_session_token, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, sessionQuery,
		sessionParams.Token,
		userID,
		tenantID,
		sessionParams.StytchMemberID,
		sessionParams.StytchOrgID,
		sessionParams.StytchSessionToken,
		sessionParams.DeviceFingerprint,
		sessionParams.ExpiresAt,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: create session in tx: %v", ErrInternal, err)
	}

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return "", "", fmt.Errorf("%w: commit tx: %v", ErrInternal, err)
	}

	r.logger.Info("tenant, user, and session created in single transaction",
		zap.String("tenant_id", tenantID),
		zap.String("user_id", userID),
		zap.String("stytch_org_id", tenantParams.StytchOrgID),
	)

	return userID, tenantID, nil
}

// generateSlug creates a URL-friendly slug from a school name.
func generateSlug(name string) string {
	// Simple slug generation — lowercased, spaces to hyphens, remove non-alphanumeric
	slug := ""
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			slug += string(r + 32) // to lowercase
		} else if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			slug += string(r)
		} else if r == ' ' || r == '-' || r == '_' {
			slug += "-"
		}
	}
	if slug == "" {
		slug = fmt.Sprintf("school-%d", time.Now().UnixNano())
	}
	return slug
}
