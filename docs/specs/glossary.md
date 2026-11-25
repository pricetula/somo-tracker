# Model Glossary

Canonical domain model and terminology for the Somotracker auth revamp.

Source of truth for entity names, attribute names, and relationships. All spec
docs must use these names; do not invent aliases.

## Status

proposed

## Entities

### organization
Top-level customer account and data-isolation boundary. Replaces the old
"tenant" concept.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| name | TEXT | NOT NULL | Organization display name |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |
| deleted_at | TIMESTAMPTZ | NULL | Soft delete |

### user
One global identity per normalized email. A user belongs to exactly one
organization.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| email | TEXT | NOT NULL | Raw email as provided |
| normalized_email | TEXT | NOT NULL | Lowercased, trimmed. UNIQUE |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| email_verified_at | TIMESTAMPTZ | NULL | Set when OTP verified |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |

### school
A school owned by exactly one organization.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| name | TEXT | NOT NULL | School display name |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |

### user_school
Links a user to a school within their organization. A user may have
different roles in different schools.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| user_id | UUID | NOT NULL | FK → users.id |
| school_id | UUID | NOT NULL | FK → schools.id |
| role | TEXT | NOT NULL | School-level role (e.g. `OWNER`, `ADMIN`, `TEACHER`) |
| status | TEXT | NOT NULL | `active`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| revoked_at | TIMESTAMPTZ | NULL | Set when assignment revoked |

### session
Opaque server-side session. Raw token never stored; only SHA-256 hash.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| token_hash | BYTEA | NOT NULL | SHA-256 of raw token, UNIQUE |
| user_id | UUID | NOT NULL | FK → users.id |
| organization_id | UUID | NOT NULL | Denormalized for efficient scoping |
| active_school_id | UUID | NULL | FK → schools.id. NULL for org-level access |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| last_seen_at | TIMESTAMPTZ | NULL | Sliding idle window |
| expires_at | TIMESTAMPTZ | NOT NULL | Absolute expiry |
| revoked_at | TIMESTAMPTZ | NULL | Set on logout/revocation |
| revocation_reason | TEXT | NULL | Why revoked |

### login_challenge
Short-lived email OTP challenge. Code never stored in plaintext.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| email_normalized | TEXT | NOT NULL | Normalized email |
| purpose | TEXT | NOT NULL | `login` |
| code_mac | BYTEA | NOT NULL | HMAC-SHA256(server_secret, challenge_id \|\| code) |
| expires_at | TIMESTAMPTZ | NOT NULL | 10 minutes from creation |
| attempt_count | INTEGER | NOT NULL | Default 0 |
| max_attempts | INTEGER | NOT NULL | Default 5 |
| used_at | TIMESTAMPTZ | NULL | Set on successful verification |
| invalidated_at | TIMESTAMPTZ | NULL | Set on resend or expiry |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |

### email_outbox
Transactional outbox for email delivery. Challenge and outbox committed
in the same DB transaction.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| challenge_id | UUID | NOT NULL | FK → login_challenges.id |
| message_type | TEXT | NOT NULL | `otp`, `invitation` |
| status | TEXT | NOT NULL | `pending`, `sent`, `failed` |
| attempts | INTEGER | NOT NULL | Default 0 |
| next_attempt_at | TIMESTAMPTZ | NULL | For retry scheduling |
| provider_message_id | TEXT | NULL | Brevo message ID |
| encrypted_payload | BYTEA | NULL | Encrypted OTP + email, key outside DB |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| sent_at | TIMESTAMPTZ | NULL | Set on successful delivery |

## Relationships

- `users.organization_id → organizations.id` (CASCADE)
- `schools.organization_id → organizations.id` (CASCADE)
- `user_schools.organization_id, user_schools.user_id → users(organization_id, id)` (composite FK)
- `user_schools.organization_id, user_schools.school_id → schools(organization_id, id)` (composite FK)
- `sessions.user_id → users.id` (CASCADE)
- `sessions.organization_id → organizations.id` (CASCADE)
- `sessions.active_school_id → schools.id` (NULL allowed)
- `login_challenges.email_normalized` matches `users.normalized_email` (no FK; challenge may pre-exist user)
- `email_outbox.challenge_id → login_challenges.id` (CASCADE)

## Terminology

| Term | Meaning | Notes |
|------|---------|-------|
| `organization_id` | Top-level account ID | Never `tenant_id` |
| opaque token | Session random string | Set in `__Host-session` cookie by Go backend |
| `token_hash` | SHA-256 of opaque token | Stored in DB, never raw token |
| OTP | Six-digit code | HMAC stored, never plaintext |
| code_mac | HMAC-SHA256(server_secret, challenge_id \|\| code) | Key from Doppler |
| Turnstile | Cloudflare bot protection | Disable via env var for local dev |
| Brevo | Transactional email provider | API key from Doppler |
| BFF | Next.js App Router | Proxies to Go, forwards cookies |
| proxy | Next.js → Go | Browser never calls Go directly |

## Constraints

- `organizations.name` unique
- `users.normalized_email` unique (global identity)
- `schools.(organization_id, name)` unique
- `user_schools.(user_id, school_id)` unique
- Composite FKs prevent cross-organization user↔school assignment
- `login_challenges.expires_at` 10 minutes after `created_at`
- `sessions.expires_at` 30 days after `created_at` (absolute)
- Sessions use absolute TTL only; no sliding idle timeout is enforced in MVP

## Links

- Authentication architecture: `docs/adr/0001-authentication-architecture.md`
- Session management: `docs/adr/0003-session-management.md`
- Specification lifecycle: proposed → approved → implemented → done
