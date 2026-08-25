package middleware

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/database"
)

// WithTenantContext scopes the current request to a tenant by opening a
// request-scoped transaction on a pinned pool connection and setting the
// RLS GUC app.current_tenant_id (transaction-scoped) on it. The transaction
// is stored in Fiber locals under database.TenantTxKey; because handlers
// pass c.Context() (the fasthttp ctx) down to repositories, its Value()
// resolves the transaction through ctx.Value(database.TenantTxKey), which
// is exactly what database.FromContext(ctx, pool) reads.
//
// The tenant is read from the resolved session (set by NewSessionResolver),
// not from per-route auth locals, so this middleware can run globally and
// still cover every authenticated request regardless of route ordering.
//
// The transaction is committed on success and rolled back on error — the
// rollback also resets the GUC, so pooled connections never leak tenant
// context to the next request.
//
// Pre-session flows (login, register, invite acceptance) have no session yet
// and therefore no request-scoped transaction here; those flows set their own
// tenant context via database.Begin/WithTenantTx once the tenant is known.
func WithTenantContext(pools *database.Pools) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, ok := c.Locals("session").(*SessionInfo)
		if !ok || sess == nil || sess.TenantID == "" {
			return c.Next()
		}

		ctx := c.Context()
		tx, err := pools.PG.Begin(ctx)
		if err != nil {
			return fmt.Errorf("middleware.WithTenantContext: begin: %w", err)
		}

		committed := false
		defer func() {
			if !committed {
				// context.WithoutCancel so a client disconnect doesn't abort the cleanup.
				if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil {
					zap.L().Warn("tx rollback failed", zap.Error(rbErr))
				}
			}
		}()

		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", sess.TenantID); err != nil {
			return fmt.Errorf("middleware.WithTenantContext: set_config: %w", err)
		}

		c.Locals(database.TenantTxKey, tx)
		c.Locals(database.TenantIDKey, sess.TenantID)

		if err := c.Next(); err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("middleware.WithTenantContext: commit: %w", err)
		}
		committed = true
		return nil
	}
}
