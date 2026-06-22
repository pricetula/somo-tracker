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
