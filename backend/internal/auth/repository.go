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
				return fmt.Errorf("auth.repository.OnStart: postgres ping failed: %w", err)
			}
			logger.Info("auth repository: postgres connection verified")
			return nil
		},
		OnStop: func(ctx context.Context) error {
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

// GetTenantByName retrieves an existing tenant's ID and Stytch org ID by name.
func (r *SqlcRepository) GetTenantByName(ctx context.Context, name string) (string, string, error) {
	const query = `SELECT id, stytch_org_id FROM tenants WHERE name = $1`
	var id, orgID string
	err := r.pool.QueryRow(ctx, query, name).Scan(&id, &orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", fmt.Errorf("%w: tenant not found: %s", ErrNotFound, name)
		}
		return "", "", fmt.Errorf("%w: get tenant by name: %v", ErrInternal, err)
	}
	return id, orgID, nil
}

// GetSessionByToken retrieves a session by its opaque token.
// The user's role is sourced from their highest-privilege active membership.
func (r *SqlcRepository) GetSessionByToken(ctx context.Context, token string) (*UserSession, error) {
	const query = `
		SELECT s.id, s.token, s.user_id, s.tenant_id,
		       COALESCE(m.role::text, 'TEACHER') as role,
		       s.stytch_member_id, s.stytch_org_id,
		       COALESCE(s.stytch_session_token, '') as stytch_session_token,
		       COALESCE(s.device_fingerprint, '') as device_fingerprint,
		       s.expires_at, s.created_at
		FROM sessions s
		LEFT JOIN LATERAL (
			SELECT role FROM memberships
			WHERE user_id = s.user_id AND is_active = true
			ORDER BY
				CASE role
					WHEN 'SYSTEM_ADMIN' THEN 1
					WHEN 'SCHOOL_ADMIN' THEN 2
					WHEN 'TEACHER' THEN 3
					WHEN 'NURSE' THEN 4
					WHEN 'FINANCE' THEN 5
				END
			LIMIT 1
		) m ON true
		WHERE s.token = $1 AND s.expires_at > NOW()
	`
	var s UserSession
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.Token, &s.UserID, &s.TenantID,
		&s.Role,
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
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Error("auth.CreateTenantUserSession: tx rollback failed",
					zap.Error(rbErr),
					zap.String("original_error", err.Error()),
				)
			}
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

// CreateUserSession creates a user and session inside a single transaction,
// without creating a new tenant. Used when a tenant already exists.
func (r *SqlcRepository) CreateUserSession(
	ctx context.Context,
	userParams CreateUserParams,
	sessionParams CreateSessionParams,
) (userID string, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: begin tx: %v", ErrInternal, err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				r.logger.Error("auth.CreateUserSession: tx rollback failed",
					zap.Error(rbErr),
					zap.String("original_error", err.Error()),
				)
			}
		}
	}()

	// 1. Insert user
	userQuery := `
		INSERT INTO users (email, tenant_id, first_name, last_name, external_auth_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = tx.QueryRow(ctx, userQuery,
		userParams.Email, userParams.TenantID, userParams.FirstName, userParams.LastName, userParams.ExternalAuthID,
	).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("%w: create user in tx: %v", ErrInternal, err)
	}

	// 2. Insert session
	sessionQuery := `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, stytch_session_token, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, sessionQuery,
		sessionParams.Token,
		userID,
		sessionParams.TenantID,
		sessionParams.StytchMemberID,
		sessionParams.StytchOrgID,
		sessionParams.StytchSessionToken,
		sessionParams.DeviceFingerprint,
		sessionParams.ExpiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("%w: create session in tx: %v", ErrInternal, err)
	}

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: commit tx: %v", ErrInternal, err)
	}

	r.logger.Info("user and session created in single transaction (existing tenant)",
		zap.String("user_id", userID),
		zap.String("tenant_id", userParams.TenantID),
	)

	return userID, nil
}

// CreateSchool creates a new cbc_school for a tenant and returns its ID.
func (r *SqlcRepository) CreateSchool(ctx context.Context, tenantID string, name string) (string, error) {
	const query = `
		INSERT INTO cbc_schools (tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, '', '', 'Public')
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("%w: create cbc_school: %v", ErrInternal, err)
	}
	return id, nil
}

// CreateMembership creates a membership linking a user to a school with a role.
func (r *SqlcRepository) CreateMembership(ctx context.Context, userID, schoolID, tenantID, role string) error {
	const query = `
		INSERT INTO memberships (role, user_id, school_id, tenant_id)
		VALUES ($1::user_role, $2, $3, $4)
		ON CONFLICT (user_id, school_id) DO UPDATE SET
			role = EXCLUDED.role,
			is_active = true,
			tenant_id = EXCLUDED.tenant_id
	`
	_, err := r.pool.Exec(ctx, query, role, userID, schoolID, tenantID)
	if err != nil {
		return fmt.Errorf("%w: create membership: %v", ErrInternal, err)
	}
	return nil
}

// GetMeInfo returns the full profile info for /me: user details, role,
// and the active school.
func (r *SqlcRepository) GetMeInfo(ctx context.Context, token string) (*MeInfo, error) {
	const query = `
		SELECT
			s.user_id,
			s.tenant_id,
			COALESCE(m.role::text, 'TEACHER') as role,
			u.first_name,
			u.last_name,
			u.email,
			sch.id as school_id,
			sch.name as school_name
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN LATERAL (
			SELECT role, school_id FROM memberships
			WHERE user_id = s.user_id AND is_active = true
			ORDER BY
				CASE role
					WHEN 'SYSTEM_ADMIN' THEN 1
					WHEN 'SCHOOL_ADMIN' THEN 2
					WHEN 'TEACHER' THEN 3
					WHEN 'NURSE' THEN 4
					WHEN 'FINANCE' THEN 5
				END
			LIMIT 1
		) m ON true
		LEFT JOIN cbc_schools sch ON sch.id = m.school_id
		WHERE s.token = $1 AND s.expires_at > NOW()
	`

	var info MeInfo
	var schoolID, schoolName *string
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&info.UserID,
		&info.TenantID,
		&info.Role,
		&info.FirstName,
		&info.LastName,
		&info.Email,
		&schoolID,
		&schoolName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: session not found", ErrNotFound)
		}
		return nil, fmt.Errorf("%w: get me info: %v", ErrInternal, err)
	}

	if schoolID != nil {
		info.SchoolID = *schoolID
	}
	if schoolName != nil {
		info.SchoolName = *schoolName
	}

	return &info, nil
}

// GetUserHighestRole returns the highest (most privileged) role for a user
// across all their active memberships.
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
