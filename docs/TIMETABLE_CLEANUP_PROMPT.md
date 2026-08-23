# Timetable Consolidation — Coding Agent Prompt

## Context
Consolidate `internal/cbctimetableslots/` and `internal/timetablestructure/` into a single `internal/timetable/` package. Do NOT commit anything (`git add` / `git commit` / `git push` forbidden by root AGENTS.md — all changes stay unstaged).

## First: Read Contracts
- `/Users/pricetula/Projects/somo-tracker/backend/AGENTS.md`
- `/Users/pricetula/Projects/somo-tracker/AGENTS.md`

## Audit (do this first)
Run `ls /Users/pricetula/Projects/somo-tracker/backend/internal/cbctimetableslots/` and `ls .../timetablestructure/`. Read `domain.go`, `handler.go`, `module.go` in both packages to confirm current endpoints.

## Plan Summary (execute this)

### 1. Delete Old Packages
Delete directories (do not commit):
- `internal/cbctimetableslots/`
- `internal/timetablestructure/`

### 2. Create New Package: `internal/timetable/`
Files to create:
- `internal/timetable/domain.go` — domain errors + `TimeBlock`, `Slot`, `SlotFilter` + view DTOs (see specs below)
- `internal/timetable/handler.go` — all endpoints (see endpoint spec below)
- `internal/timetable/service.go` — business logic
- `internal/timetable/repository.go` — DB access (use SQL from old repos; can combine or split interfaces)
- `internal/timetable/module.go` — single `fx.Module("timetable")` (one `fx.Annotate` per constructor; never duplicate)
- `internal/timetable/worker.go` — only if needed (skip if old worker not required)

### 3. Domain Model — TimeBlock (with new `Order` field)
```go
type TimeBlock struct {
    ID             string    `json:"id"`
    DayOfWeek      int       `json:"day_of_week"`
    PeriodName     string    `json:"period_name"`
    StartTime      string    `json:"start_time"`
    EndTime        string    `json:"end_time"`
    IsBreak        bool      `json:"is_break"`
    AcademicYearID string    `json:"academic_year_id,omitempty"`
    Order          int       `json:"order"` // NEW — for UI sorting
    CreatedAt      time.Time `json:"created_at,omitempty"`
    UpdatedAt      time.Time `json:"updated_at,omitempty"`
}
```

### 4. Domain Model — Slot (assignment)
Keep from old `cbctimetableslots`: `TimetableSlot` with `StructureID`, `ClassID`, `TeacherID`, `LearningAreaID`, `RoomIdentifier`, `AcademicYearID`. Rename model references in code to `Slot` if clean.

### 5. Endpoint specification (all under `/api/v1/timetable/`)

**Mutations — Structure:**
- `POST /structure` — create block; payload: `CreateTimeBlockPayload`; server resolves `AcademicYearID`
- `GET /structure` — list all; query: `academic_year_id` (from context if not provided)
- `GET /structure/:id` — get block by id
- `PUT /structure/:id` — update; payload: `UpdateTimeBlockPayload`
- `DELETE /structure/:id` — delete by id; return `DeleteResult{
    Deleted bool `json:"deleted"`
}`

**Mutations — Assignments:**
- `POST /slots` — create; accepts either single `SlotPayload` OR `[]SlotPayload` (replaces batch); server resolves `TenantID`, `SchoolID`, `AcademicYearID` from context; `StructureID` required
- `GET /slots` — list; query params: `academic_year_id`, `structure_id`, `class_id`, `teacher_id`, `learning_area_id`
- `PUT /slots/:id` — update; payload: fields to change (do NOT allow changing `StructureID` or IDs unless needed — keep simple)
- `DELETE /slots/:id` — delete by id

**Read-only Views (school resolved from active context — do NOT include school ID in URL):**
- `GET /timetable` — full school grid; query: `day_of_week` (optional), `academic_year_id` (optional — default active), `class_id` (optional filter); response: combined structure + assignments in day order using `TimeBlock.Order`
- `GET /timetable?teacher_id=...` — teacher's schedule
- `GET /timetable?class_id=...` — class schedule
- `GET /timetable?student_id=...` — student schedule (resolve via class enrollment; do NOT create new DB joins beyond what exists)

### 6. School Resolution
In handlers / service, extract `tenant_id`, `school_id`, and `academic_year_id` from request context (existing middleware `extractTenantSchool` or `WithLogger` / `TenantContext`). No school ID goes in URL paths.

### 7. Error Handling Rules (must follow)
- Use `internal/xerrors` for errors; never import middleware/HTTP packages in domain/service/repository
- Every non-2xx response returns `{ "code": "...", "message": "...", "errors": {} }` via `middleware.HTTPError()`
- Every package-level sentinel declared: `ErrNotFound`, `ErrAlreadyExists`, `ErrInvalidInput`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`
- Repository maps `sql.ErrNoRows` / `pgx.ErrNoRows` to `ErrNotFound`
- Service wraps errors: `fmt.Errorf("timetable.Service.MethodName: %w", err)` — no empty catch, no log-and-return, no silent discard
- Handler wraps before returning: use `fmt.Errorf("...: %w", xerrors.UnprocessableEntity(...))` where needed
- Use `zap` logging (not `log/slog`, not `fmt.Println`); inject `*zap.SugaredLogger` via constructor

### 8. Dependency Injection (fx rules)
- One `fx.Annotate` per constructor. Example:
  ```go
  fx.Provide(
      fx.Annotate(NewRepository, fx.As(new(Repository))),
      fx.Annotate(NewService, fx.As(new(Service))),
  )
  ```
- Interfaces declared at consumer side; concrete structs receive via `New...` constructors
- `fx.As` requires interface pointer (`new(Interface)`); never use concrete struct

### 9. Testing (must ship both suites per backend rules)
- `service_test.go` — unit tests, zero network/DB; mock repository via constructor injection
- `repository_test.go` — integration tests against active Postgres; verify SQL constraints
- Both must complete (unit should finish in ms; integration can take longer)

### 10. DB / Migrations
Do NOT change table names unless required. Existing tables `cbc_timetable_slots` and whatever `timetablestructure` uses can remain — just point repository SQL to them. Add `order` column to time-block table if not present (migration file if needed: `0000XX_timetable_order.up.sql` / `.down.sql`).

### 11. Remove Old References from Main
Update `cmd/api/main.go`:
- Remove imports / modules for `cbctimetableslots` and `timetablestructure`
- Add import / module for `timetable`
- Wire `timetable.Handler` and call `handler.RegisterRoutes(app)` (routes under `/api/v1/timetable/`)

### 12. Final Verification
After creating, verify with `go build` or `make build-backend`. Confirm no build errors and that `go test -short ./internal/timetable/` passes (unit suite at minimum). Confirm all endpoints listed in this prompt exist and respond correctly.

---
End of prompt. Execute sequentially, do not skip audit or error-handling rules.
