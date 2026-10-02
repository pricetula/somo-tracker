# 10 — Settings Completion

## Overview
Complete missing settings pages: Grade Levels management and Academic Years management.

---

## Backend Changes

### 1. Grade Levels CRUD (Partial — only GET exists)

#### SQLC Queries
**File:** `backend/db/queries/grade_levels.sql` (extend)

```sql
-- name: CreateGradeLevel :one
INSERT INTO grade_levels (education_system_id, country_id, name, local_label, sequence_index, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateGradeLevel :one
UPDATE grade_levels
SET name = $2, local_label = $3, sequence_index = $4, description = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteGradeLevel :exec
DELETE FROM grade_levels WHERE id = $1;
```

#### Service & Handler
- Extend `GradesService` + `GradesHandler` with create/update/delete

#### Router Endpoints
```
POST   /api/school/grade-levels
PATCH  /api/school/grade-levels/:id
DELETE /api/school/grade-levels/:id
```

### 2. Academic Years CRUD (Partial — only POST exists)

#### SQLC Queries
**File:** `backend/db/queries/academic_years.sql` (extend)

```sql
-- name: ListAcademicYears :many
SELECT * FROM academic_years
WHERE school_id = $1
ORDER BY start_date DESC
LIMIT $2 OFFSET $3;

-- name: GetAcademicYear :one
SELECT * FROM academic_years WHERE id = $1 AND school_id = $2;

-- name: UpdateAcademicYear :one
UPDATE academic_years
SET name = $2, start_date = $3, end_date = $4, is_active = $5, updated_at = NOW()
WHERE id = $1 AND school_id = $6
RETURNING *;

-- name: DeleteAcademicYear :exec
DELETE FROM academic_years WHERE id = $1 AND school_id = $2;

-- name: SetActiveAcademicYear :one
UPDATE academic_years
SET is_active = CASE WHEN id = $2 THEN TRUE ELSE FALSE END
WHERE school_id = $1 AND id IN ($2, (SELECT id FROM academic_years WHERE school_id = $1 AND is_active = TRUE))
RETURNING *;
```

#### Service & Handler
- New `AcademicYearsService` + `AcademicYearsHandler`

#### Router Endpoints
```
GET    /api/school/academic-years
POST   /api/school/academic-years
GET    /api/school/academic-years/:id
PATCH  /api/school/academic-years/:id
DELETE /api/school/academic-years/:id
POST   /api/school/academic-years/:id/set-active
```

---

## Frontend Changes

### 1. Feature Modules

#### Grade Levels
```
src/features/grades/
├── components/
│   └── grade-levels-table.tsx  (extend existing)
│   └── grade-level-form.tsx
├── hooks/
│   └── use-grade-levels.ts
└── services/
    └── grade-levels-api.ts
```

#### Academic Years
```
src/features/academic-years/
├── components/
│   ├── academic-years-table.tsx
│   └── academic-year-form.tsx
├── hooks/
│   └── use-academic-years.ts
├── services/
│   └── api.ts
├── types/
│   └── academic-year.ts
└── index.ts
```

### 2. Pages
```
src/app/(dashboard)/settings/
├── grade-levels/page.tsx
└── academic-years/page.tsx
```

### 3. Navigation
Already in nav under Settings:
```tsx
{
    title: "Settings",
    url: "#",
    icon: <Settings2Icon />,
    items: [
        { title: "General", url: "/settings" },
        { title: "Streams", url: "/settings/streams" },
        { title: "Grade Levels", url: "/settings/grade-levels" },  // exists
        { title: "Academic Years", url: "/academic-years" },       // needs page
    ],
}
```
Note: Fix `/academic-years` to `/settings/academic-years` for consistency.

---

## Acceptance Criteria
- [ ] Grade Levels: create, edit, delete, reorder (sequence_index)
- [ ] Academic Years: create, edit, delete, set active
- [ ] Validation: academic year dates don't overlap
- [ ] Active year used as default in timetable, attendance, enrollment
- [ ] RLS: tenant isolation

---

## Dependencies
- Requires: Education Systems, Countries (for grade levels)

---

## Estimated Effort
- Backend: ~6 hours
- Frontend: ~5 hours
- **Total: ~11 hours**