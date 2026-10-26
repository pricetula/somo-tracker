# Somotracker Backend: Agent Rules

Scope: Go (Fiber v3, pgx, sqlc, PostgreSQL, Redis, asynq). Root AGENTS.md covers commit policy and the document-led workflow. Rationale for these rules: BACKEND-ARCHITECTURE.md.
Rules are enforced by lint, CI and tests where possible. Run `go vet`, `golangci-lint` and `go test ./...` before finishing.

## Structure

- **BE-001** Layout: `cmd/<binary>/`, `internal/app/` (composition root), `internal/platform/<capability>/` (infrastructure, no business logic), `internal/<domain>/{handler,service,store}/`. Create no other top-level packages. Never create `utils`, `helpers`, `common` or `misc`.
- **BE-002** Dependency direction: `handler` imports `service`; `store` imports `service` (domain types and the Repository interface); `internal/app` wires all three. `service` never imports `store`, `handler`, sqlc, pgx or Fiber. Domains never import each other; a domain that needs another declares a small interface in its own `service/`, and `app` wires the other domain's service in.
- **BE-003** A domain is a business capability, not a table. Every table has exactly one owning domain, and only that domain's store writes to it. Read-only joins across tables are fine.
- **BE-004** Shared code: infrastructure goes in `platform/<capability>`. A shared pure helper gets a package named for what it does (`money`, `cursor`). A helper used by one domain stays unexported in that domain. Share only when 3+ domains need it, and never if it touches the database or business rules.
- **BE-005** Dependency injection is manual: plain `New<Type>` constructors with explicit dependencies, wired only in `internal/app`. No DI frameworks, globals, `init()` side effects or service locators. Interfaces are small and defined by the consumer.

## Handlers

- **BE-006** Handlers are thin: bind and validate input, read the session context, call one service method, map the result to a response DTO. No SQL, no sqlc or pgx imports, no business rules, no transactions.
- **BE-007** Handlers never return sqlc rows or domain structs; always named DTO structs.
- **BE-008** Fiber is fasthttp-based: never keep `fiber.Ctx`, `c.Params()` strings or body bytes after the handler returns or pass them to goroutines; copy first. Below the handler pass `context.Context`, never `fiber.Ctx`.

## Data layer

- **BE-009** SQL lives only in `store/queries/*.sql`, compiled by sqlc into `store/gen/` (never hand-edited). No ORM, no query builder, no SQL in Go strings.
- **BE-010** Stores map sqlc rows to domain types and Postgres errors to `apperr` errors. Store methods map 1:1 to queries and hold no business logic.
- **BE-011** Migrations are Goose SQL files only. Never edit an applied migration. RLS policies and DB roles live in migrations. Migrations never run at app startup.
- **BE-012** One transaction per service call via `TxManager.Run`; stores take their connection from the context, and services never touch a raw connection. An orchestrating service owns the transaction for multi-domain atomic operations. Never hold a transaction across a network call (Stytch, email, PDF, Redis). Enqueue jobs after commit.
- **BE-013** Types: UUID primary keys (UUIDv7 for new rows; client-generated is allowed for offline-written records). Timestamps are `timestamptz` (UTC). School-day dates are `date`, interpreted in Africa/Nairobi. Money is `bigint` minor units (KES), never floats.
- **BE-014** All lists use keyset pagination: request `limit`, `cursor`, `sort`; response `{ items, nextCursor }`. `sort` is a whitelisted name that includes direction (e.g. `name_asc`); `limit` defaults to 50, and above 100 returns 400. Fetch `limit + 1`, use a unique `id` tiebreaker and row comparison, and one named sqlc query per whitelisted sort. The cursor is opaque and bound to the sort and filters; a mismatch returns 400. No OFFSET and no total count (use a dedicated count endpoint if needed).
- **BE-015** Invariants live in the database (constraints and indexes), not only in application code. Bulk operations (import, rollover) are atomic with rollback.

## Tenancy, session, authorization

- **BE-016** Every query is tenant-scoped, and school-scoped for school data, in addition to RLS. The app connects as a non-owner DB role, and RLS variables are set per transaction from the session context. Cross-tenant and nonexistent records return the identical 404.
- **BE-017** The session context (user, tenant, school, role) is loaded once per request by middleware from one query and carried in `context.Context` as a typed struct. Nothing else reads or writes the active school. Never accept a school or tenant ID from the client as scope.
- **BE-018** The role always comes from the live membership. Never cache it in the session or Redis.
- **BE-019** Sessions use an opaque random token (at least 256 bits); only its SHA-256 hash is stored. The cookie is `HttpOnly; Secure; SameSite=Lax` with the `__Host-` prefix. Reject unsafe methods whose `Origin` doesn't match. No state changes on GET. Sessions have idle and absolute expiry owned by our DB, and the absolute expiry never exceeds the Stytch session's `expires_at`. Stytch is called at login and logout only, never per request. The magic-link callback is a POST, never a GET. The session stores the Stytch member-session ID for revocation; any Stytch token kept (only if revocation by ID proves impossible) is encrypted at rest. Stytch tokens are never logged. Stytch is reached only through an interface declared in `identity/service` and implemented by the adapter in `platform/stytch`; login, invite and session logic live in the identity service.
- **BE-020** Every route registers with `authz.Require(perm)` or an explicit `Public()`. Use permissions, never role lists. The permission matrix is data in `platform/authz`.
- **BE-021** Services obtain a `Scope` from `authz.ScopeFor(ctx, perm)` and pass it to the store, which applies it in the query's WHERE clause. Out-of-scope single records return 404. Field-level restrictions use separate response types backed by restricted query projections; never fetch everything and strip fields.

## API contract

- **BE-022** Versioning: every route lives under `/api/v1/` (webhooks included), created as one route group in `platform/httpserver`; handlers register relative paths. API v1 status: **unreleased** (breaking changes allowed; change this word to `released` at the first production deploy). Once released, v1 changes are additive only (new endpoints, new optional response fields). Breaking changes (removing or renaming fields or endpoints, changing types or semantics, adding required request fields, tightening validation, removing enum values) ship as `/api/v2/` alongside v1.
- **BE-023** The OpenAPI spec is generated by swag from annotations (`@BasePath /api/v1`). Use one annotation template per handler and the standard `@Failure` set. Request and response types are named structs, never inline. Cursor pages use one generic wrapper.
- **BE-024** Validation uses `go-playground/validator/v10`, plugged into Fiber via `fiber.Config.StructValidator` and used through `c.Bind()`. Validation tags go only on request DTOs in `handler/` (shape, format, length). Business rules are validated in services and returned as `apperr`. Field names come from json tags; each failed tag maps to a stable `errors[].code`. Custom validators are registered once in `platform/validation`.
- **BE-025** Services return typed errors from `apperr`. One central Fiber error handler renders `{ code, message, errors?, requestId }`; handlers never write error JSON. `code` is a stable SCREAMING_SNAKE enum defined in `apperr` and exported in the spec. Status mapping: 400 `BAD_REQUEST`, 401 `UNAUTHENTICATED`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 409 `CONFLICT`, 422 `VALIDATION_FAILED`, 429 `RATE_LIMITED` (with `Retry-After`), 500 `INTERNAL` (generic message only). `errors[].field` is a dotted path (`guardians.0.phone`).
- **BE-026** Never serve `/healthz`, `/readyz`, metrics or the spec under `/api`. Trust the forwarded client IP header only from the Next.js proxy hop. Authenticated responses set `Cache-Control: no-store`.

## Jobs, Redis, audit

- **BE-027** Background jobs use asynq. Handlers are idempotent (at-least-once delivery), and payloads are typed structs carrying `tenant_id` and `school_id`.
- **BE-028** Redis holds ephemeral data only (rate limits, short-lived tokens, cache, queue). Postgres is the source of truth. The queue and the cache never share an eviction policy. If Redis is down, rate limiting falls back to an in-process limiter (fail open); nothing durable is lost.
- **BE-029** Audit: create, update and delete events on health, behavior and fee records are written in the same transaction as the change. View events go through `audit.Recorder` (queued, event UUID with `ON CONFLICT DO NOTHING`, direct DB write if enqueue fails; if both fail, deny the view). Audit rows hold IDs and changed field names, never sensitive values (fee waivers also record old and new value, proposer and approver). The app role has INSERT and SELECT only on audit tables.
- **BE-030** Rate limits on authenticated routes are keyed by user, not IP; IP keys apply only before authentication. No permanent in-app IP bans, only TTL blocks. Imports, PDFs and exports have per-tenant concurrency caps.

## Logging and security

- **BE-031** Log with `slog` JSON to stdout only. Log IDs (request, tenant, school, user), route pattern, status, latency and error code. Never log names, phones, emails, health or fee data, tokens, cookies, bodies or SQL arguments. Sensitive types implement `slog.LogValuer`. No `fmt.Print*` and no std `log`. Log an error once, in the central handler; wrap errors with `%w`.
- **BE-032** Error tracking uses Sentry (`sentry-go`, EU region). Report panics and 5xx only, never 4xx. Keep `SendDefaultPII` off, and scrub in `BeforeSend`: drop request bodies, headers, cookies, query strings and user data. Attach only request, tenant, school and user IDs as tags. No SQL arguments or entity values in messages or breadcrumbs.
- **BE-033** Secrets come from the environment or a secret manager, never the repo. All config goes through one typed `platform/config`.
- **BE-034** CSV and spreadsheet exports neutralize formula injection (cells starting `=`, `+`, `-`, `@`). Cap upload size and validate file types.

## Quality

- **BE-035** Reference FR/NFR IDs in PRs, commits and tests. The PRD is a rough draft: when it is silent or ambiguous, raise an Open Question instead of assuming. Don't build Phase 2 or Later features in MVP work.
- **BE-036** Every endpoint has tests for tenant isolation (cross-tenant returns 404) and for each role in the permission matrix (Guardian has no access in the MVP; its tests start with the Phase 2 portal). Store and RLS tests use a real PostgreSQL (testcontainers). Services are tested with fakes of their Repository interface. Contract tests validate real responses against the generated spec. Prefer table-driven tests.

## Conventions

- **Naming:** packages are lowercase single words; files are snake_case; constructors are `New<Type>`; interfaces have no `I` prefix; sqlc queries are verb-first PascalCase (`GetStudent`, `ListStudentsByName`, `CreateStudent`); migrations are `<timestamp>_<description>.sql`.
- **Context:** `context.Context` is the first parameter and is never stored in a struct.
- **Errors:** wrap with `%w`, compare with `errors.Is`/`errors.As`, never panic for control flow.
- **API:**
  - JSON and query parameter names are camelCase (`schoolId`, `createdAt`); acronyms are cased as words (`id`, not `ID`).
  - IDs are UUID strings. Timestamps are RFC 3339 UTC with `Z`. Date-only fields are `yyyy-MM-dd` strings. Money is an integer in minor units with a `Minor` suffix (`amountMinor`), KES only.
  - Enum values are SCREAMING_SNAKE strings (`PRESENT`, `EE`). Adding an enum value is additive; removing one is breaking.
  - URLs are plural kebab-case nouns (`/api/v1/students/{id}/guardians`). Non-CRUD actions are `POST /<resource>/{id}/<verb>`.
  - A single resource returns the bare object, never an envelope; lists return `{ items, nextCursor }`. Declared fields are always present: no value is `null`, and arrays are `[]`, never null.
  - Updates use `PUT` with the full editable body. No generic `PATCH`.
  - Success codes: 200 reads and updates, 201 creates (returns the created body), 202 queued work, 204 deletes.
  - Create endpoints used by offline clients take a client-generated `id` and are idempotent on it (a repeat returns the existing record). No generic `Idempotency-Key` header in the MVP.
- **Dependencies:** add none without approval.
- **Scope:** change only what the task requires; don't refactor unrelated code.

## Definition of done

`go vet`, `golangci-lint` and `go test ./...` pass. `sqlc generate` and `swag init` produce no diff. `govulncheck` is clean. New endpoints have the tests required by BE-036, and requirement IDs are referenced per BE-035.
