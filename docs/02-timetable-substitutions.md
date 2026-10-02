# 02 — Timetable Substitutions

## Overview
Implement teacher substitution management for timetable slots. When a teacher is absent, admins assign a substitute teacher for specific dates. Critical for timetable operations.

---

## Backend Changes

### 1. SQLC Queries
**File:** `backend/db/queries/timetable_substitutions.sql` (new)

```sql
-- name: CreateSubstitution :one
INSERT INTO timetable_substitutions (
    class_timetable_slot_id, substitution_date, substitute_teacher_membership_id,
    original_teacher_membership_id, reason, status
) VALUES ($1, $2, $3, $4, $5, 'PENDING')
RETURNING *;

-- name: GetSubstitution :one
SELECT * FROM timetable_substitutions WHERE id = $1;

-- name: ListSubstitutions :many
SELECT ts.*, 
    cts.class_room_id, cr.name as class_name,
    s.name as subject_name,
    tm.user_id as original_teacher_id, u.full_name as original_teacher_name,
    stm.user_id as substitute_teacher_id, su.full_name as substitute_teacher_name,
    ts.name as time_slot_name, ts.start_time, ts.end_time
FROM timetable_substitutions ts
JOIN class_timetable_slots cts ON cts.id = ts.class_timetable_slot_id
JOIN class_rooms cr ON cr.id = cts.class_room_id
JOIN subjects s ON s.id = cts.subject_id
JOIN school_memberships tm ON tm.id = cts.teacher_membership_id
JOIN users u ON u.id = tm.user_id
JOIN school_memberships stm ON stm.id = ts.substitute_teacher_membership_id
JOIN users su ON su.id = stm.user_id
JOIN time_slots ts ON ts.id = cts.time_slot_id
WHERE ts.school_id = $1
  AND ($2::date IS NULL OR ts.substitution_date >= $2)
  AND ($3::date IS NULL OR ts.substitution_date <= $3)
  AND ($4::text IS NULL OR ts.status = $4)
ORDER BY ts.substitution_date, ts.start_time
LIMIT $5 OFFSET $6;

-- name: CountSubstitutions :one
SELECT COUNT(*) FROM timetable_substitutions
WHERE school_id = $1
  AND ($2::date IS NULL OR substitution_date >= $2)
  AND ($3::date IS NULL OR substitution_date <= $3)
  AND ($4::text IS NULL OR status = $4);

-- name: UpdateSubstitution :one
UPDATE timetable_substitutions
SET substitute_teacher_membership_id = $2,
    reason = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSubstitution :exec
DELETE FROM timetable_substitutions WHERE id = $1;
```

### 2. Service
**File:** `backend/internal/services/timetable_substitutions_service.go` (extend existing)

```go
type Substitution struct {
    ID                          uuid.UUID
    ClassTimetableSlotID        uuid.UUID
    SubstitutionDate            time.Time
    SubstituteTeacherMembershipID uuid.UUID
    OriginalTeacherMembershipID uuid.UUID
    Reason                      string
    Status                      string // PENDING, ASSIGNED, COMPLETED, CANCELLED
    ClassName                   string
    SubjectName                 string
    OriginalTeacherName         string
    SubstituteTeacherName       string
    TimeSlotName                string
    StartTime                   string
    EndTime                     string
}

type ListSubstitutionsParams struct {
    SchoolID          uuid.UUID
    Page              int
    Limit             int
    DateFrom          *time.Time
    DateTo            *time.Time
    StatusFilter      string
}

type TimetableSubstitutionsService interface {
    CreateSubstitution(ctx context.Context, schoolID uuid.UUID, req CreateSubstitutionRequest) (*Substitution, error)
    GetSubstitution(ctx context.Context, id uuid.UUID) (*Substitution, error)
    ListSubstitutions(ctx context.Context, params ListSubstitutionsParams) ([]*Substitution, int, error)
    UpdateSubstitution(ctx context.Context, id uuid.UUID, req UpdateSubstitutionRequest) (*Substitution, error)
    DeleteSubstitution(ctx context.Context, id uuid.UUID) error
}
```

### 3. Handler
**File:** `backend/internal/api/timetable_substitutions_handler.go` (extend existing)

Add endpoints:
- `GET /api/timetable/substitutions` — list with filters
- `POST /api/timetable/substitutions` — create
- `GET /api/timetable/substitutions/:id` — get
- `PATCH /api/timetable/substitutions/:id` — update
- `DELETE /api/timetable/substitutions/:id` — delete

### 4. Router Registration
**File:** `backend/internal/api/router.go`

```go
protected.Get("/timetable/substitutions", r.TimetableSubstitutions.ListSubstitutions)
protected.Post("/timetable/substitutions", r.TimetableSubstitutions.CreateSubstitution)
protected.Get("/timetable/substitutions/:id", r.TimetableSubstitutions.GetSubstitution)
protected.Patch("/timetable/substitutions/:id", r.TimetableSubstitutions.UpdateSubstitution)
protected.Delete("/timetable/substitutions/:id", r.TimetableSubstitutions.DeleteSubstitution)
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/timetable/
├── components/
│   ├── substitutions-table.tsx
│   └── substitution-form.tsx
├── hooks/
│   └── use-substitutions.ts
├── services/
│   └── substitutions-api.ts
├── types/
│   └── substitution.ts
└── index.ts
```

### 2. Page
**File:** `src/app/(dashboard)/timetable/substitutions/page.tsx` (new)

```tsx
import { SubstitutionsTable } from "@/features/timetable";

export default function SubstitutionsPage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-semibold">Teacher Substitutions</h1>
            <SubstitutionsTable />
        </div>
    );
}
```

### 3. Navigation
Add to timetable section in nav or Settings:
- `/timetable/substitutions`

---

## Acceptance Criteria
- [x] Create substitution: select slot, date, substitute teacher, reason
- [x] List substitutions with filters: date range, status, class
- [x] Status workflow: PENDING → ASSIGNED → COMPLETED/CANCELLED
- [x] Prevent double-booking substitute teacher on same slot/date
- [ ] Notify substitute teacher (future: email/push)
- [ ] RLS: tenant isolation (school isolation enforced; full RBAC pending)
- [x] Frontend: DataTable with status badges, date filters

---

## Dependencies
- Requires: Rooms CRUD (for complete timetable setup)
- Requires: Teachers list (for substitute selection)

---

## Estimated Effort
- Backend: ~5 hours
- Frontend: ~4 hours
- **Total: ~9 hours**