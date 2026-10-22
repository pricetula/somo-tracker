# Code Review Checklist — Somotracker

> **Purpose:** General code review checklist used alongside security-audit skill.
> **Usage:** Referenced by `.pi/skills/security-audit/SKILL.md` prerequisites; used in PR reviews.

---

## Universal Checks (All Layers)

### Error Handling
- [ ] No empty `catch` / `if err != nil {}` — every error handled or wrapped
- [ ] No log-and-return — errors logged once at handler/worker layer only
- [ ] No silent `_ = func()` discarding errors in non-test code
- [ ] All non-2xx responses return canonical error shape (`code`, `message`, `errors` map)

### Security
- [ ] No secrets in code, logs, or comments
- [ ] User input validated at boundary (DTOs with validation tags)
- [ ] SQL uses parameterized queries (no string interpolation)
- [ ] Path traversal prevented (no user input in file paths)
- [ ] XSS prevented (no raw HTML in responses; CSP headers set)

### Testing
- [ ] New endpoints have integration tests (auth + authorization scenarios)
- [ ] Business logic has unit tests with edge cases
- [ ] Migration has up/down test (if schema change)
- [ ] Security-relevant changes have authorization test (user A ≠ user B)

### Documentation
- [ ] OpenAPI annotations on new/modified handlers
- [ ] README updated if setup/config changes
- [ ] ADR created for architectural decisions
- [ ] Changelog entry for user-facing changes

---

## Backend (Go/Fiber) Specific

### Handler Layer
- [ ] Uses dedicated request DTO (not DB model) with `c.BodyParser(&req)`
- [ ] Returns `errors.BadRequest()`, `errors.NotFound()`, `errors.Forbidden()`, `errors.Internal()` — never raw `c.Status(500).Send(err.Error())`
- [ ] Reads `userID`, `role`, `tenantID` from `c.Locals()` (set by auth middleware)
- [ ] No business logic in handler — delegates to service

### Service Layer
- [ ] Receives `userID`/`tenantID` as explicit parameters (not from context)
- [ ] Returns domain errors (`ErrNotFound`, `ErrForbidden`, `ErrConflict`) — not HTTP errors
- [ ] No direct DB access — uses repository interface
- [ ] Transactions managed at service level for multi-step operations

### Repository Layer
- [ ] Uses parameterized queries / GORM safe methods
- [ ] Scopes all queries by `tenant_id` / `owner_id` where applicable
- [ ] Returns `ErrNotFound` (not `nil, nil`) for missing records
- [ ] No business logic — pure data access

### Middleware
- [ ] Auth middleware sets `c.Locals("userID")`, `c.Locals("role")`, `c.Locals("tenantID")`
- [ ] Rate limiter uses correct key (userID for authenticated, IP for public)
- [ ] Body parser has size limit (`bodyparser.Config{Limit: 1<<20}`)
- [ ] Security headers via `helmet` middleware
- [ ] CORS restricted to configured origins (no `*` with credentials)

### Database / Migrations
- [ ] Migration is reversible (has `DOWN` SQL)
- [ ] Indexes added for new query patterns
- [ ] Foreign keys for referential integrity
- [ ] `NOT NULL` constraints where appropriate
- [ ] No `DROP COLUMN` in production without multi-phase migration

---

## Frontend (Next.js/React) Specific

### Components
- [ ] Uses pure Shadcn components (no custom Tailwind colors, no wrapper divs)
- [ ] No `ArrowLeft` / "Back" buttons inside pages (navigation in shell only)
- [ ] Forms use `react-hook-form` + `zod` validation
- [ ] API calls use generated client (`frontend/src/lib/api/client.ts`)
- [ ] Error handling uses canonical error shape from backend

### Pages / Routes
- [ ] Server components for data fetching (no `useEffect` for initial load)
- [ ] Client components only for interactivity (`'use client'`)
- [ ] Loading/error states with Suspense boundaries
- [ ] No direct `fetch` — uses API client with interceptors

### State / Data
- [ ] Server state via TanStack Query (not `useState` + `useEffect`)
- [ ] Mutations invalidate related queries
- [ ] Optimistic updates for UX-critical actions

---

## API Contract (Backend ↔ Frontend)

### Request/Response
- [ ] Request DTO matches frontend TypeScript type
- [ ] Response DTO matches frontend TypeScript type
- [ ] Enum values synchronized (backend `const` ↔ frontend `enum`)
- [ ] Pagination format consistent: `{ data: [], meta: { page, limit, total } }`

### Errors
- [ ] All error codes documented in `backend/internal/api/errors.go`
- [ ] Frontend error handler maps codes to user messages
- [ ] Field-level `errors` map populated for validation failures

### Versioning
- [ ] Breaking changes under new version path (`/api/v2/`)
- [ ] Deprecated endpoints marked in OpenAPI + `Sunset` header
- [ ] Migration guide for frontend consumers

---

## Security-Audit Specific (from OWASP API Top 10)

### API1: BOLA
- [ ] Every `GET/PUT/DELETE /:resource/:id` checks ownership via service
- [ ] List endpoints filter by `owner_id`/`tenant_id` from context

### API2: Auth
- [ ] JWT middleware on all protected routes
- [ ] Token expiry ≤15min (access), ≤7d (refresh)
- [ ] Rate limit on `/auth/*` endpoints

### API3: BPLA
- [ ] Response DTOs exclude sensitive fields (`password_hash`, `totp_secret`, etc.)
- [ ] Request DTOs exclude `role`, `owner_id`, `created_at`
- [ ] PATCH uses pointer fields + `omitempty` for partial updates

### API4: Consumption
- [ ] Pagination `limit` capped at 100
- [ ] Body limit middleware configured
- [ ] Expensive operations rate-limited per user

### API5: BFLA
- [ ] Admin routes in separate group with `auth.RequireRole("admin")`
- [ ] Handler-level role checks (defense in depth)
- [ ] Tenant isolation uses context, not request params

### API6: Business Flows
- [ ] CAPTCHA on public registration/invite
- [ ] Idempotency keys on payments/webhooks
- [ ] Business logic limits (max invites, teams, etc.)
- [ ] Audit logging on sensitive mutations

### API7: SSRF
- [ ] No user-supplied URLs fetched directly
- [ ] Allowlist for outbound HTTP domains
- [ ] Private IPs blocked at transport level

### API8: Misconfig
- [ ] Security headers (CSP, HSTS, X-Frame-Options, Referrer-Policy)
- [ ] No debug/pprof endpoints in production
- [ ] Custom error handler (no stack traces in responses)
- [ ] CORS restricted to known origins

### API9: Inventory
- [ ] OpenAPI spec generated from code (`swag init`)
- [ ] All routes documented with annotations
- [ ] Deprecated endpoints marked

### API10: Third-Party
- [ ] Secrets from vault/env only (never in code)
- [ ] TLS verification on all outbound calls
- [ ] Webhook signatures verified before processing
- [ ] Circuit breaker on external API calls

---

## Quick Commands

```bash
# Backend
make build-backend          # Compile check
make test-backend           # Run Go tests
make lint-backend           # golangci-lint

# Frontend
npm run build:frontend      # Next.js build
npm run lint:frontend       # ESLint
npm run test:frontend       # Jest + Playwright (Ignore)

# Security
make security-audit         # Run security-audit skill (CI)
make api-inventory          # List all routes vs OpenAPI spec

# Database
make migrate-up             # Apply migrations
make migrate-down           # Rollback last migration
make migration-test         # Test up+down on test DB
```

---

## PR Review Flow

1. **Author**: Runs `make build-backend`, `make test-backend`, `npm run build:frontend`
2. **Reviewer**: Loads `security-audit` skill → runs checklist → outputs `security-report.md`
3. **CI**: Runs full test suite + `make security-audit` + `make api-inventory`
4. **Merge**: Only after all checks pass + security findings resolved (Critical/High = block)