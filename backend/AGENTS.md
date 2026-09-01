# Somotracker Backend — Agent Architecture Contract

Structural patterns, dependency rules, and testing mandates for backend.

---

## 1. Directory Layout & Package Isolation

The Go application uses **Functional Domain Package Layering** — code grouped by functional cohesion, not infrastructural layer.

```
cmd/
└── api/
    └── main.go                 # Entry point: wire dependencies, boot Fiber

internal/
├── tenant/
├── billing/
└── analytics/                  # Example domain package
    ├── domain.go               # Core structs, enums, view models (pure Go)
    ├── repository.go           # Database access (SQL, row scanning)
    ├── service.go              # Business logic and calculation formulas
    ├── handler.go              # Fiber route handlers
    ├── service_test.go         # Unit tests (in-memory mocks)
    └── repository_test.go      # Integration tests (live DB)
```

- **Zero circular imports.** If `billing` imports `student`, then `student` must never import `billing`.
- **Locality of Behavior.** All handlers, business logic, and SQL for a functional area must live entirely within that area's package under `./backend/internal/`.

---

## 2. Cross-Domain Data & SQL Joins

**Same-domain joins:** Write a native SQL `JOIN` inside that package's `repository.go`.

**Cross-domain joins:** No hard imports between domain packages. Use one of:

1. **Orchestrator service** — sits above domain packages, calls both repositories independently (use `errgroup` for concurrency), assembles a DTO in memory.
2. **Database View (CQRS read-model)** — define a read-only PostgreSQL `VIEW` spanning the domains; map it to a flat read-only Go struct in the consuming package.

---

## 3. Dependency Injection

### ⚠️ ABSOLUTE RULE: One `fx.Annotate` per constructor. Never register the same constructor twice.

```go
// ⛔ FORBIDDEN — duplicates the constructor, creates two pool connections
fx.Provide(
    fx.Annotate(NewRepository, fx.As(new(Repository))),
    NewRepository,
)

// ⛔ FORBIDDEN — two fx.Annotate calls for the same constructor
fx.Provide(
    fx.Annotate(NewRepository, fx.As(new(Repository))),
    fx.Annotate(NewRepository, fx.As(new(ServiceRepository))),
)

// ✅ REQUIRED — single fx.Annotate with multiple fx.As
fx.Provide(
    fx.Annotate(
        NewRepository,
        fx.As(new(Repository)),
        fx.As(new(ServiceRepository)),
    ),
)

// ✅ Also valid when only one interface is needed
fx.Provide(
    fx.Annotate(NewRepository, fx.As(new(Repository))),
)
```

**If a constructor appears more than once in any `fx.Provide` block, the code review must be rejected.**

- No global state, no package-level DB vars, no `init()` functions.
- All structs receive dependencies via a `New…` constructor.
- Interfaces are declared at the **consumer** side (not the implementation side).

```go
type Repository interface { /* declared inside this package */ }

type Service struct { repo Repository }

func NewService(r Repository) *Service {
    return &Service{repo: r}
}
```

### `fx.As` — interface types only

`fx.As(new(T))` requires `T` to be an **interface**. Using it with a concrete struct type causes a runtime panic:

```go
// BAD — *PgRepository is a concrete struct, not an interface
fx.Annotate(NewRepository, fx.As(new(*PgRepository)))
// → runtime panic: "fx.As: argument must be a pointer to an interface"
```

### Publishing a single constructor as multiple interfaces

When the same repository struct implements multiple service-layer interfaces (e.g. `Repository` for cross-domain wiring and `ServiceRepository` for the local service), use multiple `fx.As` options on a single `fx.Annotate` call — **do not** register the constructor twice:

```go
// GOOD — single constructor, two interface bindings
fx.Annotate(
    NewRepository,
    fx.As(new(Repository)),          // for cross-domain consumers
    fx.As(new(ServiceRepository)),   // for local Service
)

// BAD — duplicate registration creates two pool connections
fx.Provide(
    fx.Annotate(NewRepository, fx.As(new(Repository))),
    NewRepository,  // ← second call creates a second pool!
)
```

**Rule:** One constructor call per lifecycle. Use multiple `fx.As(new(Interface))` to bind the same result to several interfaces. Never register the same constructor twice — fx calls it each time, creating duplicate resources (pools, clients, etc.).

---

## 5. Testing

Every feature must ship both suites.

**Unit tests** (`go test -short`) — `*_service_test.go`

- Zero network, zero disk, zero live DB.
- Inject in-memory map mocks via the constructor.
- Must complete in milliseconds.

**Integration tests** (`go test`) — `*_repository_test.go`

- Run against an active Postgres instance.
- Verify SQL constraints, data types, composite unique indexes, and RLS rules.

---

## 6. Error Handling

### Canonical error response shape

Every non-2xx HTTP response MUST return `{ "code": string, "message": string, "errors": object }`.
Implementing code: `internal/middleware/errors.go` — `HTTPError()` helper.
Frontend counterpart: `src/lib/api/client.ts`.

### Canonical error types (`internal/xerrors`)

`internal/xerrors` is the single source of structured error types. `DomainError` carries
the machine-readable `Code`, HTTP `Status`, client-safe `Message`, and optional `Fields`
(validation metadata). It implements `error`, `Unwrap`, and the `HasDetails` interface
(`ErrorDetails() any`) for extra response metadata. Packages must never import
`middleware` or HTTP packages to create errors — build them from `xerrors` only.

Package-level sentinels for middleware use: `xerrors.ErrNotFound`, `ErrAlreadyExists`,
`ErrInvalidInput`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`. Named constructors
return per-message instances: `xerrors.NotFound(msg)`, `xerrors.AlreadyExists(msg)`,
`xerrors.InvalidInput(msg)`, `xerrors.Unauthorized(msg)`, `xerrors.Forbidden(msg)`,
`xerrors.Conflict(msg)`, `xerrors.UnprocessableEntity(msg)`, and `xerrors.New(code, status, msg)`
for custom codes (e.g. auth's `expired_token`).

### Sentinel errors in every domain.go

Every module under `internal/` must declare these package-level sentinel errors, built
from `xerrors` constructors so they carry the canonical `Code`/`Status`:

```go
var (
    ErrNotFound      = xerrors.NotFound("<module> not found")
    ErrAlreadyExists = xerrors.AlreadyExists("<module> already exists")
    ErrInvalidInput  = xerrors.InvalidInput("invalid <module> input")
    ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
    ErrForbidden     = xerrors.Forbidden("forbidden")
    ErrConflict      = xerrors.Conflict("<module> conflict")
)
```

- `sql.ErrNoRows` must always be mapped to `ErrNotFound` inside the repository. It must never reach the service layer. (Use `xerrors.MapPgxError` in shared paths, or `errors.Is(err, pgx.ErrNoRows)` inside database packages.)
- Module-specific sentinels (e.g. `ErrExpiredToken`) may be added alongside these via `xerrors.New(code, status, msg)`.

### Error wrapping at every layer boundary

Naming convention: `<Package>.<Type>.<Method>: %w`

```go
// repository
return nil, fmt.Errorf("members.Repository.FindByID: %w", err)
// service
return nil, fmt.Errorf("members.Service.GetMember: %w", err)
```

### HTTPError helper (`internal/middleware/errors.go`)

- `HTTPError(c *fiber.Ctx, err error) error` is the **only** place HTTP status codes are decided for domain errors.
- Uses `errors.As()` to extract the nearest `*xerrors.DomainError` from the chain; `errors.Is()` for special cases (`context.Canceled`, `context.DeadlineExceeded`).
- Status mapping (from the embedded `DomainError.Status`):
    - `ErrNotFound` → 404, `ErrAlreadyExists` → 409, `ErrInvalidInput` → 400
    - `ErrUnauthorized` → 401, `ErrForbidden` → 403, `ErrConflict` → 409, `UnprocessableEntity` → 422
    - `context.Canceled` → 499, `context.DeadlineExceeded` → 504
    - everything else → 500 (logged, generic message)
- Handlers must always wrap errors before returning them, e.g. `fmt.Errorf("invalid request body: %w", xerrors.UnprocessableEntity("malformed request body"))`, so the HTTPError message includes handler context.

### Global Fiber error handler (`cmd/api/main.go`)

- Registered in `fiber.Config.ErrorHandler`.
- Last-resort catcher for any escaped error, including panics via `recover` middleware.
- Logs with `loggerFrom(c).Errorw(...)` (zap), returns the standard JSON body.
- Fiber's built-in `recover` middleware is registered before all routes.

### Log-once rule

- **zap is the logging library.** `log/slog` must not be used. No `log.Println`, `fmt.Println`, or `log.Printf` in non-test code.
- Dependencies receive `*zap.SugaredLogger` via constructor injection (`logger *zap.SugaredLogger` field). The shared instance is provided by `internal/logger` (fx), so `zap.NewNop().Sugar()` is the standard no-op for tests.
- `middleware` helpers (HTTPError, access log, panic recovery, rate limiter) read the logger from `c.Locals` via `middleware.WithLogger` / `loggerFrom(c)` — they must never take a logger parameter.
- Log once at the layer where the error is first **handled** (handler or worker).
- Intermediate layers (repository, service) only wrap and return — they do **not** log.
- Level usage: `Error` = unexpected failure, `Warn` = handled degradation, `Info` = significant state change, `Debug` = detailed tracing.
- Key-value style: use the sugared API (`logger.Infow(msg, "key", value)`). For typed fields use the structured `*zap.Logger` (`logger.Info(msg, zap.String("key", value))`). Never mix `slog.Attr` into a sugared call.

### Forbidden patterns

- `return err` without wrapping — always use `fmt.Errorf`.
- `return nil, err` without wrapping — always use `fmt.Errorf`.
- `err.Error() == "some string"` — use `errors.Is(err, ErrSomeSentinel)`.
- Any `_ = someFunc()` in non-test code.
- `log.Println` / `fmt.Println` in production code paths.
- Empty `if err != nil { }` blocks — log and act.
- Inline goroutines without a `defer recover()` that logs with `logger.Errorw(...)`.
- Calling `c.Next()` after a failed auth check.

### Additional rules

- **Transactions:** Every `tx.Begin()` must use the deferred rollback pattern with dual-error logging.
- **External API calls:** Wrap external errors into module-local errors before propagating. Never leak external error messages to HTTP clients.
- **fx lifecycle:** Every constructor returns `(T, error)`. Every `OnStart`/`OnStop` returns `error`. `OnStop` errors are logged AND returned.
- **Migration failure:** Must cause startup to abort — error propagates to fx, which refuses to start.
- **Background workers:** Log failures with `logger.Errorw(...)`. Distinguish severity (warn vs error). Never silently continue.

### When adding a new module

Every new module must follow this standard from creation. No retrofitting later.

---

## 7. Security Checks & Automated Auditing

All backend code written or modified by the agent must pass local security and vulnerability checks prior to completing any task.

### Mandated Tooling & Local Rules

1. **Static Analysis (SAST):** `gosec` must run against changed Go packages to catch unsafe pointers, hardcoded keys, SQL injection vulnerabilities, and bad permissions.
2. **Vulnerability Scanning:** `govulncheck` must run to verify that no introduced dependencies contain published vulnerabilities in the Go vulnerability database.
3. **Secret Leak Detection:** No hardcoded API keys, JWT secrets, database passwords, or private tokens in source files or configuration samples. `gitleaks` must be evaluated before commits.

### Verification Protocol for Agents

Before completing any feature, bug fix, or refactor involving `cmd/` or `internal/`, the agent **must** execute the following shell pipeline:

```bash
# 1. Dependency vulnerability audit
govulncheck ./...

# 2. Security-focused static analysis
gosec -quiet ./...

# 3. Detect accidentally staged or hardcoded credentials
gitleaks detect --no-git --verbose
```
