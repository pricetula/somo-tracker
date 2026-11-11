# Backend requirements (BE-001…BE-045)

Source: `backend/AGENTS.md`. One line per rule; full rule text lives in the source file.

## Structure

| ID | Summary | Source rule |
|----|---------|-------------|
| BE-001 | Layout: `cmd/<binary>/`, `internal/app/`, `internal/platform/<capability>/`, `internal/<domain>/{handler,service,store}/`; no `utils`, `helpers`, `common`, `misc` | BE-001 |
| BE-002 | Dependency direction handler→service, store→service, app wires; service never imports store/handler/sqlc/pgx/Fiber; domains never import each other | BE-002 |
| BE-003 | A domain is a business capability, not a table; exactly one owning domain per table; only that domain's store writes | BE-003 |
| BE-004 | Shared helpers in `platform/<capability>` or a package named for what it does; share only when 3+ domains need it | BE-004 |
| BE-005 | Manual DI via `New<Type>` constructors wired in `internal/app`; no DI frameworks, globals, `init()`, service locators | BE-005 |

## Handlers

| ID | Summary |
|----|---------|
| BE-006 | Handlers are thin: bind/validate input, read session context, call one service method, map to DTO |
| BE-007 | Handlers never return sqlc rows or domain structs; always named DTO structs |
| BE-008 | No `fiber.Ctx` retention after handler return; pass `context.Context` below the handler |

## Data layer

| ID | Summary |
|----|---------|
| BE-009 | SQL only in `store/queries/*.sql` via sqlc; no ORM/query builder/SQL in Go strings |
| BE-010 | Stores map sqlc rows to domain types and Postgres errors to `apperr`; 1:1 with queries; no business logic |
| BE-012 | One transaction per service call via `TxManager.Run`; stores take connection from context; enqueue jobs after commit |
| BE-013 | UUIDv7 PKs; `timestamptz` UTC; school-day dates `date` Africa/Nairobi; money `bigint` minor units KES |
| BE-014 | Keyset pagination: `limit`/`cursor`/`sort`; `{items, nextCursor}`; limit default 50 max 100; no OFFSET/total count |
| BE-015 | Invariants in DB constraints/indexes; bulk operations atomic with rollback |
| BE-037 | Every migration needs `migrator_integration_test.go` verification |
| BE-038 | UUID defaults always `uuidv7()` |
| BE-039 | `updated_at` via DB trigger `trigger_set_timestamp` |
| BE-040 | Every multi-tenant table: RLS + isolation policy; identical 404s cross-tenant |
| BE-041 | Soft delete `deleted_at` + partial index `WHERE deleted_at IS NULL` |
| BE-042 | Reference tables `ON DELETE RESTRICT`, never CASCADE |
| BE-043 | Status/check fields use `ENUM`, not `TEXT CHECK` |
| BE-044 | Composite indexes for frequent lookup pairs |
| BE-045 | No JSON schema validation unless specified; default `'{}'` |

## Tenancy, session, authorization

| ID | Summary |
|----|---------|
| BE-016 | Every query tenant-scoped (+ school-scoped for school data) plus RLS; non-owner DB role; identical 404 cross-tenant |
| BE-017 | Session context loaded once per request; carried in `context.Context`; never accept school/tenant ID from client |
| BE-018 | Role from live membership only; never cached |
| BE-019 | Opaque session token ≥256 bits, SHA-256 hash stored; `__Secure-session_token` cookie; Stytch at login/logout only; POST magic-link callback |
| BE-020 | Every route `authz.Require(perm)` or explicit `Public()`; permissions never role lists; matrix is data in `platform/authz` |
| BE-021 | Services obtain `Scope` from `authz.ScopeFor` and pass to store; out-of-scope single records return 404 |

## API contract

| ID | Summary |
|----|---------|
| BE-022 | Routes under `/api/v1/`; v1 unreleased (breaking allowed); after release additive only; breaking changes ship as `/api/v2/` |
| BE-023 | `docs/api/openapi.yaml` is frontend type source of truth; `swag` generates `backend/docs/swagger.*` separately |
| BE-024 | Validation via `go-playground/validator/v10` on request DTOs only; business rules in services |
| BE-025 | Error body `{code,message,errors?,request_id}`; SCREAMING_SNAKE codes; status mapping; snake_case JSON, camelCase query params; PUT full body; success codes 200/201/202/204 |
| BE-026 | No `/healthz`, `/readyz`, metrics, or spec under `/api`; authenticated responses `Cache-Control: no-store` |

## Naming and conventions

| ID | Summary |
|----|---------|
| — | Lowercase single-word packages; snake_case files; `New<Type>` constructors; no `I` interfaces; verb-first PascalCase sqlc queries; `<timestamp>_<description>.sql` migrations |
| — | JSON snake_case; UUID string IDs; RFC 3339 UTC `Z`; `yyyy-MM-dd` dates; money `Minor` suffix int; SCREAMING_SNAKE enums; kebab-case plural URLs |

## Jobs, Redis, audit

| ID | Summary |
|----|---------|
| BE-027 | Asynq jobs; idempotent handlers; typed payloads with `tenant_id`/`school_id` |
| BE-028 | Redis ephemeral only; Postgres source of truth; separate eviction policies; in-process rate-limit fallback fail open |
| BE-029 | Audit writes in same transaction as change; view events via `audit.Recorder`; deny view on failure; audit rows hold IDs/field names only |

## Logging and security

| ID | Summary |
|----|---------|
| BE-030 | Rate limits keyed by user after auth, IP before auth; no permanent IP bans; per-tenant concurrency caps |
| BE-031 | `slog` JSON stdout; log IDs/route/status/latency/error code only; no PII, tokens, bodies, SQL args |
| BE-032 | Sentry EU; panics and 5xx only; `SendDefaultPII` off; scrub in `BeforeSend`; tags: request/tenant/school/user IDs |
| BE-033 | Secrets from env/secret manager; typed `platform.config` |
| BE-034 | CSV/spreadsheet exports neutralize formula injection; cap upload size and validate file types |

## Quality

| ID | Summary |
|----|---------|
| BE-035 | Reference FR/NFR IDs in PRs/commits/tests; PRD is rough draft; raise Open Questions on ambiguity; no Phase 2/Later in MVP |
| BE-036 | Every endpoint: tenant-isolation tests (404 cross-tenant) + per-role permission matrix tests; store/RLS tests with real Postgres (testcontainers); services with repository fakes; contract tests against generated spec |

## Dependencies and scope

| ID | Summary |
|----|---------|
| — | Add no dependency without approval; change only what the task requires; no unrelated refactors |
