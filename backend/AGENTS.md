# Somotracker Backend — Agent Rules

Version: 0.1.0
Owner: Platform team

## Rule list

### BE-001 Handler / Service separation
Handlers are thin transport adapters only. No SQL, pgx, or sqlc calls in handlers. All DB access via `internal/services`.
Enforced by: TODO(tooling) – suggested: depguard

### BE-002 Academic period resolution
Academic period resolution is backend-only. Never accept `term_id` or `academic_year_id` from clients.
Statuses: `active|break|expired|none`. Use `AcademicPeriodResolver`.
Enforced by: TODO(tooling) – suggested: contract test + lint for forbidden params

### BE-003 Error contract
Non-2xx responses must return:
`{ "code": "snake_case", "message": "...", "errors": { "field": ["msg"] } }`
Enforced by: TODO(tooling) – suggested: error-contract test

### BE-004 API spec alignment
Router prefix + route must match `docs/api/openapi.yaml` and frontend client calls.
Enforced by: TODO(tooling) – suggested: contract test, `make generate` idempotency

### BE-005 Migration testing
Every migration SQL file must have `TestMigrator_<Feature>` integration test asserting schema via `information_schema`.
Enforced by: TODO(tooling) – suggested: CI check for test presence per migration

### BE-006 Forbidden patterns
Forbidden: empty error handling, log-and-return duplication, silent `_ =`.
Enforced by: TODO(tooling) – suggested: golangci-lint errcheck + custom lint

### BE-007 Tenancy scoping
TODO(human): define per-query tenancy scoping. Reference ADR-0001. Do not implement until ADR accepted.
Enforced by: manual review

### BE-008 Authorization
TODO(human): define authorization model. Reference ADR-0002. Do not implement until ADR accepted.
Enforced by: manual review

### BE-009 Time zones
TODO(human): define canonical time zone handling.
Enforced by: TODO(tooling)

### BE-010 Asynq idempotency / retries
TODO(human): define idempotency keys and retry policy.
Enforced by: TODO(tooling)

### BE-011 Migration rollback policy
TODO(human): define rollback policy.
Enforced by: manual review

## General
See root AGENTS.md for commit policy and document-led workflow.
Library versions: see go.mod.
