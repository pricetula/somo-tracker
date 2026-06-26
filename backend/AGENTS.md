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

---

## 4. Database Migration Policy

All schema changes go directly into:

**`internal/database/migrations/000001_initial_schema.up.sql`**

- **Do NOT create new migration files.**
- Add columns inline in `CREATE TABLE IF NOT EXISTS` statements.
- Add new tables, indexes, constraints, and views inline in the same file.
- For tables owned by future extensions, use `ALTER TABLE … ADD COLUMN IF NOT EXISTS`.
- `000002_seed.up.sql` is the only separate file — data population only, not schema DDL.

**Changelog:**
| Date | Change |
|------|--------|
| 2026-06-16 | Merged `000003` (`is_final`) and `000004` (`stream`) into `000001_initial_schema.up.sql` as inline column declarations. |
| 2026-06-26 | Squashed `000003_cbc_streams_and_classes` into `000001_initial_schema.up.sql`: added `cbc_streams` table, refactored `cbc_classes` to use `stream_id` FK, replaced `uq_cbc_classes_tenant` with `uq_cbc_classes_tier_stream`. |

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

### Sentinel errors in every domain.go
Every module under `internal/` must declare these package-level sentinel errors:

```go
var (
    ErrNotFound      = errors.New("<module> not found")
    ErrAlreadyExists = errors.New("<module> already exists")
    ErrInvalidInput  = errors.New("invalid <module> input")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrConflict      = errors.New("<module> conflict")
)
```

- `sql.ErrNoRows` must always be mapped to `ErrNotFound` inside the repository. It must never reach the service layer.
- Module-specific sentinels (e.g. `ErrExpiredToken`) may be added alongside these.

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
- Uses `errors.Is()` to unwrap the full error chain.
- Status mapping:
  - `ErrNotFound` → 404, `ErrAlreadyExists` → 409, `ErrInvalidInput` → 400
  - `ErrUnauthorized` → 401, `ErrForbidden` → 403, `ErrConflict` → 409
  - `context.Canceled` → 499, `context.DeadlineExceeded` → 504
  - everything else → 500 (logged, generic message)

### Global Fiber error handler (`cmd/api/main.go`)
- Registered in `fiber.Config.ErrorHandler`.
- Last-resort catcher for any escaped error, including panics via `recover` middleware.
- Logs with `slog.ErrorContext`, returns the standard JSON body.
- Fiber's built-in `recover` middleware is registered before all routes.

### Log-once rule
- `log/slog` must be used throughout. No `log.Println`, `fmt.Println`, or `log.Printf` in non-test code.
- Log once at the layer where the error is first **handled** (handler or worker).
- Intermediate layers (repository, service) only wrap and return — they do **not** log.
- Level usage: `Error` = unexpected failure, `Warn` = handled degradation, `Info` = significant state change, `Debug` = detailed tracing.

### Forbidden patterns
- `return err` without wrapping — always use `fmt.Errorf`.
- `return nil, err` without wrapping — always use `fmt.Errorf`.
- `err.Error() == "some string"` — use `errors.Is(err, ErrSomeSentinel)`.
- Any `_ = someFunc()` in non-test code.
- `log.Println` / `fmt.Println` in production code paths.
- Empty `if err != nil { }` blocks — log and act.
- Inline goroutines without a `defer recover()` that logs with `slog.ErrorContext`.
- Calling `c.Next()` after a failed auth check.

### Additional rules
- **Transactions:** Every `tx.Begin()` must use the deferred rollback pattern with dual-error logging.
- **External API calls:** Wrap external errors into module-local errors before propagating. Never leak external error messages to HTTP clients.
- **fx lifecycle:** Every constructor returns `(T, error)`. Every `OnStart`/`OnStop` returns `error`. `OnStop` errors are logged AND returned.
- **Migration failure:** Must cause startup to abort — error propagates to fx, which refuses to start.
- **Background workers:** Log failures with `slog.ErrorContext`. Distinguish severity (warn vs error). Never silently continue.

### When adding a new module
Every new module must follow this standard from creation. No retrofitting later.
