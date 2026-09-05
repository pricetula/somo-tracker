# Database Schema Reference

## Overview

Somotracker uses PostgreSQL 16 with a **multi-tenant Row-Level Security (RLS)** architecture. All application data is scoped under a `tenant_id` and isolated at the database level via policies that read `current_setting('app.current_tenant_id')`.

Migrations live in `backend/db/migrations/` and are applied via `golang-migrate` during Fx startup.

---

## Extensions (Migration 000001)

| Extension | Source | Purpose |
|-----------|--------|---------|
| `uuid-ossp` | PostgreSQL contrib | UUID v1/v4 generation (fallback) |
| `pgcrypto` | PostgreSQL contrib | `gen_random_uuid()`, `digest()`, `crypt()`, `hmac()` |
| `pg_uuidv7` | Third-party (tvondra/pg_uuidv7) | Time-ordered UUID v7 (optional, gracefully skipped if absent) |

---

## Core Tables (Migration 000002)

### `tenants`

Maps 1:1 to a Stytch OIDC organization. Every Somotracker row is scoped under exactly one tenant.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, `gen_random_uuid()` | Opaque primary key |
| `name` | VARCHAR(255) | NOT NULL | Display name (e.g. "Acme Corp") |
| `slug` | VARCHAR(255) | NOT NULL, UNIQUE | URL-safe identifier for subdomains/routing |
| `stytch_org_id` | VARCHAR(255) | NOT NULL, UNIQUE | Stytch OIDC org ID — authoritative SSO anchor |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | UTC creation timestamp |

**Indexes:**
- `tenants_slug_idx` — lookup by slug (subdomain routing)
- `tenants_stytch_org_id_idx` — O(1) Stytch org join
- `tenants_created_at_idx` — time-bucketed admin queries

---

### `users`

Per-tenant user accounts. Email is unique **within a tenant** only.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, `gen_random_uuid()` | Opaque primary key |
| `email` | VARCHAR(255) | NOT NULL, LOWER() | Canonical email (enforced lowercase via CHECK) |
| `tenant_id` | UUID | NOT NULL, FK→tenants(id) ON DELETE CASCADE | Tenant scope |
| `full_name` | VARCHAR(255) | NOT NULL, DEFAULT '' | Display name |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Soft-disable flag |
| `external_auth_id` | VARCHAR(255) | UNIQUE | Stytch user ID / auth provider subject |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | UTC creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | UTC last modification |

**Constraints:**
- `users_tenant_email_uniq` — UNIQUE (tenant_id, email)
- `users_email_lowercase` — CHECK (email = LOWER(email))

**Indexes:**
- `users_tenant_id_idx` — pure tenant scans
- `users_external_auth_id_idx` — O(1) auth token → user lookup
- `users_updated_at_idx` — stale-user cleanup / last-seen reporting

**Row-Level Security:**
```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY users_tenant_isolation ON users
    FOR ALL TO PUBLIC
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
    );
```
- Fail-closed: empty/invalid `app.current_tenant_id` → zero rows
- Session middleware sets this GUC via `SET LOCAL` at transaction start

---

## Auth & Session Tables (Migration 000003)

### `sessions`

Server-issued opaque session tokens. Raw Stytch tokens are **never** stored here — only cached in Redis.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, `gen_random_uuid()` | Internal row ID |
| `token` | VARCHAR(64) | NOT NULL, UNIQUE | Opaque 256-bit hex token (HttpOnly cookie) |
| `stytch_session_id` | VARCHAR(255) | NOT NULL | Stytch session reference for revocation |
| `user_id` | UUID | NOT NULL, FK→users(id) ON DELETE CASCADE | User scope |
| `tenant_id` | UUID | NOT NULL, FK→tenants(id) ON DELETE CASCADE | Tenant scope |
| `expires_at` | TIMESTAMPTZ | NOT NULL | Rolling expiry (default 7 days) |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Session creation |
| `last_seen_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Last validated request (trigger-updated) |

**Indexes:**
- `sessions_token_idx` — primary cookie → session lookup
- `sessions_stytch_session_id_idx` — global Stytch revocation
- `sessions_user_id_idx` — per-user invalidation (password change)
- `sessions_expires_at_idx` — background cleanup job

**Trigger:** `sessions_last_seen_trg` → `update_last_seen()` BEFORE UPDATE

---

### `members`

B2B member identity mirroring Stytch. Created/updated atomically with `users` during magic-link provisioning.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, `gen_random_uuid()` | Internal row ID |
| `stytch_member_id` | VARCHAR(255) | NOT NULL, UNIQUE | Stytch B2B Member.member_id |
| `user_id` | UUID | NOT NULL, FK→users(id) ON DELETE CASCADE | Local user link |
| `tenant_id` | UUID | NOT NULL, FK→tenants(id) ON DELETE CASCADE | Organization scope |
| `stytch_member_raw` | JSONB | — | Cached Stytch member object (sensitive fields stripped) |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Last modification |

**Constraints:**
- `members_tenant_member_uniq` — UNIQUE (tenant_id, stytch_member_id)

**Indexes:**
- `members_stytch_member_id_idx` — auth provisioning lookups
- `members_user_id_idx` — user → member reverse lookups

**Trigger:** `members_updated_at_trg` → `update_last_seen()` BEFORE UPDATE

> **Note:** `members` is **not RLS-protected** — it's cross-tenant for admin lookups. API layer enforces org-scoped access.

---

## Entity Relationship Diagram

```
tenants (1) ─────< (N) users
    │                   │
    │                   │
    └─────< (N) sessions
    │                   │
    │                   │
    └─────< (N) members
```

- All FKs use `ON DELETE CASCADE`
- `sessions` + `members` carry `tenant_id` for fast scoped queries
- `users` is the **only** RLS-enforced table

---

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Tables | snake_case, plural | `tenants`, `users`, `sessions` |
| Columns | snake_case | `stytch_org_id`, `external_auth_id` |
| PK | `id` (UUID) | `id UUID DEFAULT gen_random_uuid()` |
| FK | `{table}_id` | `tenant_id`, `user_id` |
| Indexes | `{table}_{column}_idx` | `users_tenant_id_idx` |
| Constraints | `{table}_{purpose}_uniq` / `_chk` | `users_tenant_email_uniq` |
| Policies | `{table}_{purpose}` | `users_tenant_isolation` |
| Triggers | `{table}_{action}_trg` | `sessions_last_seen_trg` |

---

## RLS Session Context Propagation

Application code **must** run tenant-scoped queries inside:

```go
database.WithTenantTx(ctx, pool, logger, tenantID, func(ctx context.Context, tx pgx.Tx) error {
    // All queries here auto-scoped to tenant_id
    return q.GetUserByID(ctx, userID)
})
```

The wrapper executes:
```sql
SET LOCAL app.current_tenant_id = $1;  -- tenant_id string
```

This scopes the GUC for the **entire transaction** (including nested queries). `SET LOCAL` is discarded at COMMIT/ROLLBACK — zero leakage risk across pooled connections.

---

## Data Retention & Cleanup

| Table | Strategy |
|-------|----------|
| `sessions` | Background job deletes rows where `expires_at < NOW()` |
| `users` | Soft-delete via `is_active = false` (audit retention) |
| `members` | Cascade from user delete; no standalone cleanup |
| `tenants` | Cascade deletes users → sessions → members |

---

## Security Notes

- **No passwords stored** — passwordless via Stytch magic links
- **Email enforced lowercase** at DB layer (CHECK constraint)
- **RLS on `users` only** — other tables scoped via API/service layer
- **Session tokens opaque** — 256-bit crypto/rand hex, HttpOnly cookie
- **External auth IDs unique** — prevent account takeover via email collision