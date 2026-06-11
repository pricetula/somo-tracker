Somotracker Backend Architecture Contract: Domain-Driven Package Layering
This document establishes the structural patterns, dependency routing rules, and testing mandates for the Somotracker backend. All AI engineering agents must conform strictly to these layout boundaries to preserve code locality, performance velocity, and complete testability.

🏗️ 1. Backend Directory Layout & Package Isolation
The Go application implements a Functional Domain Package Layering pattern. Code is grouped by functional cohesion rather than infrastructural layers.

.
├── cmd/
│   └── api/
│       └── main.go             # Entry point: Wire dependencies & boot Fiber
└── internal/
    ├── tenant/                 
    ├── billing/                
    └── analytics/              # Self-Contained Analytics Domain Package
        ├── domain.go           # Core structures, Enums, and View Models (Pure Go)
        ├── repository.go       # Database access layer (SQL Statements, Rows Scanning)
        ├── service.go          # Pure business logic and calculation formulas
        ├── handler.go          # Delivery Layer (Fiber router context endpoints)
        ├── service_test.go     # Fast Unit Tests (In-memory mocks)
        └── repository_test.go  # Live Integration Tests (Database/SQL verification)

Non-Negotiable Compilation Boundaries:
Zero Circular Imports: Package student must never import package billing if package billing imports package student. Circular dependency tracking will forcefully halt compilation.

Locality of Behavior (LoB): All handler routes, business operations, and target SQL queries serving a single functional area must exist completely within that specific area's domain folder under ./backend/internal/.

🔍 2. Handling Data Combinations & SQL Joins
When an operational workflow or user interface requires data spanning multiple database tables, agents must apply these strict routing paradigms:

Scenario A: Same-Domain Joins
If the required target tables fall natively under the responsibilities of a single package (e.g., combining analytics snapshots with academic term boundaries), the agent must write a native SQL JOIN query inside that package's repository.go layer.

Scenario B: Cross-Domain Inter-Package Combinations
If tables belong to completely separated packages (e.g., combining student profile metadata with accounting ledger lines), packages must remain strictly isolated. Agents are blocked from creating hard imports between them. Instead, choose one of these two options:

The Application Usecase Orchestrator: Create a specialized service orchestrator layer above the domain packages. This orchestrator calls both package repositories independently (utilizing concurrent Go routines or errgroup if performance critical) and aggregates the records into a single DTO in application memory.

The Database View Model Pattern (CQRS Read-Model): Define a read-only database VIEW at the PostgreSQL layer that spans the cross-domain rows. Inside the consuming reporting package (e.g., analytics), treat this view as a flat, single table mapped directly to a read-only Go struct definition.

🛡️ 3. Dependency Injection Rules
To guarantee complete control over state during testing loops, no package may use global states, database instances hidden in global variables, or implicit init() package functions.

All package structs must explicitly request their dependencies (database connections, external clients, internal interfaces) through concrete constructor functions named New...

Interfaces must be declared where they are consumed (the client side) rather than where the implementation is declared, following idiomatic Go conventions.

// Example Constructor enforcing Clean Mock Injection
type Service struct {
repo Repository // Interface declared locally within this package
}

func NewService(r Repository) *Service {
return &Service{repo: r}
}

🧪 4. Non-Negotiable Test Assertions
Every feature generated or refactored by coding agents must accompany a complete testing profile split into two explicit execution suites using Go build tags:

1. Unit Tests (go test -short)
Target File: *service_test.go or integrated test blocks.

Execution Boundary: Pure, decoupled, in-memory domain logic valuation.

Mandate: Zero network connections, zero disk access, and zero live database connections. If a service depends on a database repository, a clean in-memory map mock structure must be injected into the constructor. Execution window must resolve in milliseconds.

2. Integration Tests (go test)
Target File: *repository_test.go.

Execution Boundary: Verifying actual infrastructure integration against the database layer.

Mandate: Must run against an active Postgres/Neon instance to execute raw SQL scripts, verifying that constraints, data types, composite unique indexes, and multi-tenant Row-Level Security (RLS) rules execute perfectly without regression.