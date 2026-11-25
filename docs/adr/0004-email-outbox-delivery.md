# ADR-0004: Email Outbox / Delivery

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0004 |
| title | Email Outbox / Delivery |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend |
| relates_to | ADR-0001, ADR-0005 |

## Status

proposed

## Context

SPEC-001 requires sending OTP codes via email using Brevo (Sendinblue). The system must reliably deliver time-sensitive OTPs without blocking the OTP request response, while ensuring no duplicate or lost emails.

Key forces:
- OTP request creates a `login_challenge` row and an `email_outbox` row in one DB transaction.
- The `email_outbox` table stores `challenge_id`, `message_type`, `status` (pending/sent/failed), `attempts`, `next_attempt_at`, `provider_message_id`, and an `encrypted_payload` containing email + code + expiry.
- The payload is encrypted with a key outside the database.
- Brevo API key is injected via Doppler at runtime (`BREVO_API_KEY`).
- OTP codes expire in 10 minutes; delivery must be fast enough to be useful.
- The system must handle Brevo transient failures with retries and not send duplicate emails for the same challenge.

Key decisions needed:
- Outbox processing model (in-process worker vs separate service).
- Retry/backoff strategy for failed deliveries.
- Idempotency guarantees (one email per challenge).
- Encryption of outbox payloads.
- Status tracking and observability.

## Decision

Adopt a transactional outbox pattern with in-process async delivery:

1. **Outbox table**: `email_outbox` with columns `id`, `challenge_id`, `message_type` ('otp'), `status` ('pending'/'sent'/'failed'), `attempts`, `next_attempt_at`, `provider_message_id`, `encrypted_payload`, `created_at`, `sent_at`.
2. **Write path**: OTP request handler inserts `login_challenge` + `email_outbox` (status='pending') in a single transaction; returns success to client before email is sent.
3. **Delivery**: A background goroutine (in-process, same Go process as the API) polls `email_outbox` for `status='pending'` rows where `next_attempt_at <= NOW()`, calls Brevo API, and updates status to 'sent' (with `provider_message_id` and `sent_at`) or 'failed' (incrementing `attempts`, setting `next_attempt_at` with exponential backoff).
4. **Encryption**: `encrypted_payload` contains email + code + expiry, encrypted with `EMAIL_HMAC_KEY`-derived key outside the database. Plaintext code never stored in `email_outbox`.
5. **Idempotency**: One `email_outbox` row per `challenge_id`; status transitions are one-way (pending → sent/failed). Brevo `provider_message_id` stored for dedupe/tracing.
6. **Retry strategy**: Exponential backoff with cap; after max attempts, status='failed' and an alert fires (see monitoring/alerts).
7. **Configuration**: `BREVO_API_KEY`, `BREVO_SENDER_EMAIL`, `BREVO_SENDER_NAME` injected via Doppler. No fallback defaults in config.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Synchronous email in request handler | Simplest flow, immediate feedback | Blocks OTP response on email latency; slow UX | Rejected — OTP request should return immediately |
| Separate worker service | Isolates delivery load, independent scaling | Operational complexity (deploy, monitor, network) | In-process goroutine sufficient for current scale |
| Third-party queuing (SQS/RabbitMQ) | Durable, retry built-in | Additional infrastructure cost and complexity | PostgreSQL outbox table is durable enough |
| Store plaintext code in outbox | Simpler encryption story | Plaintext OTP in DB = security risk if DB leaked | Rejected — payload must be encrypted |

## Consequences

### Positive

- OTP request returns immediately; email delivery is decoupled and non-blocking.
- Outbox table provides durable retry queue; no email lost on transient Brevo failure.
- Encrypted payload protects OTP codes even if database is compromised.
- `provider_message_id` enables Brevo-side tracing and duplicate detection.

### Negative

- In-process worker shares resources with API; delivery load competes with request handling.
- Worker runs in every API instance; multiple instances may poll the same rows (requires `SELECT ... FOR UPDATE SKIP LOCKED` or similar to avoid double-processing — not yet implemented).
- No separate scaling knob for delivery volume.

### Neutral / Risks

- Double-processing risk if multiple API instances poll without row locking — must implement `SKIP LOCKED` or advisory locks before multi-instance deployment.
- No dead-letter queue beyond `status='failed'`; manual inspection needed for persistently failing emails.
- Brevo rate limits are not yet tracked at the outbox layer (global OTP budget is tracked separately in Redis, see ADR-0005).

## Links

- Spec: `docs/specs/001-auth-otp-request.md` (§CreateEmailOutbox, outbox payload)
- Auth ADR: `docs/adr/0001-authentication-architecture.md`
- Rate limiting ADR: `docs/adr/0005-rate-limiting-strategy.md`
- Store: `backend/internal/auth/store/store.go` (`CreateEmailOutbox`)
- Query: `backend/internal/auth/store/queries/auth.sql` (`email_outbox`)
- Alerts: `backend/monitoring/alerts/ratelimit.yaml`
