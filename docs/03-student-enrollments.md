# 03 — Student Enrollments

## Overview
Implement student class enrollment management. Currently enrollments are created implicitly via student import; need explicit CRUD for admins to move students between classes, handle mid-year transfers, and manage enrollment status.

---

## Backend Changes

### 1. SQLC Queries
**File:** `backend/db/queries/student_enrollments.sql` (new)

```sql
-- name: CreateEnrollment :one
INSERT INTO student_class_enrollments (
    student_id, class_room_id, academic_year_id, academic_term_id,
    enrollment_date, status, metadata
) VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6)
RETURNING *;

-- name: ListEnrollments :many
SELECT sce.*, 
    cr.name as class_name, cr.grade_level_id, gl.local_label as grade,
    cr.stream, s.admission_number, s.full_name as student_name
FROM student_class_enrollments sce
JOIN class_rooms cr ON cr.id = sce.class_room_id
JOIN grade_levels gl ON gl.id = cr.grade_level_id
JOIN students s ON s.id = sce.student_id
WHERE sce.school_id = $1
  AND ($2::uuid IS NULL OR sce.class_room_id = $2)
  AND ($3::uuid IS NULL OR sce.student_id = $3)
  AND ($4::text IS NULL OR sce.status = $4)
ORDER BY sce.enrollment_date DESC
LIMIT $5 OFFSET $6;

-- name: CountEnrollments :one
SELECT COUNT(*) FROM student_class_enrollments
WHERE school_id = $1
  AND ($2::uuid IS NULL OR class_room_id = $2)
  AND ($3::uuid IS NULL OR student_id = $3)
  AND ($4::text IS NULL OR status = $4);

-- name: UpdateEnrollment :one
UPDATE student_class_enrollments
SET class_room_id = COALESCE($2, class_room_id),
    status = COALESCE($3, status),
    metadata = COALESCE($4, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEnrollment :exec
DELETE FROM student_class_enrollments WHERE id = $1;

-- name: GetActiveEnrollmentByStudent :one
SELECT * FROM student_class_enrollments
WHERE student_id = $1 AND academic_year_id = $2 AND status = 'ACTIVE';
```

### 2. Service
**File:** `backend/internal/services/enrollments_service.go` (new)

```go
type Enrollment struct {
    ID                 uuid.UUID
    StudentID          uuid.UUID
    ClassRoomID        uuid.UUID
    AcademicYearID     uuid.UUID
    AcademicTermID     uuid.UUID
    EnrollmentDate     time.Time
    Status             string // ACTIVE, PROMOTED, REPEATING, GRADUATED
    ClassName          string
    Grade              string
    Stream             string
    StudentName        string
    AdmissionNumber    string
}

type CreateEnrollmentRequest struct {
    StudentID      uuid.UUID `json:"student_id"`
    AcademicYearID uuid.UUID `json:"academic_year_id"`
    AcademicTermID uuid.UUID `json:"academic_term_id"`
    EnrollmentDate string    `json:"enrollment_date"` // YYYY-MM-DD
    Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type UpdateEnrollmentRequest struct {
    ClassRoomID *uuid.UUID `json:"class_room_id,omitempty"`
    Status      *string    `json:"status,omitempty"`
    Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type ListEnrollmentsParams struct {
    SchoolID     uuid.UUID
    Page         int
    Limit        int
    ClassRoomID  *uuid.UUID
    StudentID    *uuid.UUID
    StatusFilter string
}

type EnrollmentsService interface {
    CreateEnrollments(ctx context.Context, schoolID uuid.UUID, reqs []CreateEnrollmentRequest) ([]*Enrollment, error)
    ListEnrollments(ctx context.Context, params ListEnrollmentsParams) ([]*Enrollment, int, error)
    UpdateEnrollment(ctx context.Context, id uuid.UUID, req UpdateEnrollmentRequest) (*Enrollment, error)
    DeleteEnrollment(ctx context.Context, id uuid.UUID) error
    GetActiveEnrollmentByStudent(ctx context.Context, studentID, academicYearID uuid.UUID) (*Enrollment, error)
}
```

### 3. Handler
**File:** `backend/internal/api/enrollments_handler.go` (new)

Endpoints:
- `POST /api/classes/:id/enrollments` — enroll multiple students in class (batch)
- `GET /api/classes/:id/enrollments` — list enrollments for class
- `PATCH /api/enrollments/:id` — update (move class, change status)
- `DELETE /api/enrollments/:id` — unenroll

### 4. Router Registration
```go
protected.Post("/classes/:id/enrollments", r.Enrollments.CreateEnrollments)
protected.Get("/classes/:id/enrollments", r.Enrollments.ListEnrollmentsByClass)
protected.Patch("/enrollments/:id", r.Enrollments.UpdateEnrollment)
protected.Delete("/enrollments/:id", r.Enrollments.DeleteEnrollment)
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/enrollments/
├── components/
│   ├── enrollments-table.tsx
│   ├── enrollment-form.tsx
│   └── bulk-enroll-dialog.tsx
├── hooks/
│   └── use-enrollments.ts
├── services/
│   └── api.ts
├── types/
│   └── enrollment.ts
└── index.ts
```

### 2. Integration Points
- **Classes page**: Add "Enroll Students" button → opens bulk enroll dialog
- **Student detail page**: Show current enrollment, allow transfer
- **Settings**: Enrollment management page

### 3. Page
**File:** `src/app/(dashboard)/classes/[id]/enrollments/page.tsx` (new)

---

## Acceptance Criteria
- [ ] Enroll multiple students in class via batch POST (with date, status)
- [ ] Bulk enroll multiple students (CSV upload or multi-select UI)
- [ ] Transfer student between classes (updates enrollment, preserves history)
- [ ] Change enrollment status: ACTIVE → PROMOTED/REPEATING/GRADUATED
- [ ] Prevent duplicate active enrollment per student per academic year
- [ ] View enrollment history per student
- [ ] RLS: tenant isolation
- [ ] Frontend: DataTable with class, student, status filters

---

## Dependencies
- Requires: Classes, Students, Academic Years/Terms
- Blocks: Accurate attendance (needs correct enrollment)

---

## Estimated Effort
- Backend: ~6 hours
- Frontend: ~5 hours
- **Total: ~11 hours**