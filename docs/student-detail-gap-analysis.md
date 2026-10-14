# Student Detail Page – Gap Analysis & Schema Tightening

**Purpose:** Document current implementation gaps for student details and propose concrete schema / feature additions to make the student detail page production-ready for School Management Systems.

**Status:** Draft
**Last Updated:** 2026-10-15

## Current Implementation

### Backend
**Table: students**
- student_id UUID PK
- school_id UUID FK
- admission_number text unique per school
- full_name text
- date_of_birth date
- gender enum student_gender
- metadata JSONB
- created_at / updated_at

**Related tables**
- `guardian_student_links` – links school_membership to student with relationship_type
- `student_class_enrollments` – historical class mapping
- `timetable_attendance` / `event_attendance`
- `student_gender_counts` – denormalized counts

**API**
- `GET /students` – paginated list with search, class filter
- `DELETE /students`
- `GET /students/summary`
No `GET /students/:id` enriched endpoint. No PATCH for personal details.

### Frontend
- List view functional with filters and actions
- Detail view `frontend/src/features/students/components/student-detail.tsx` = placeholder

## Gaps Identified

### 1. Data Model Gaps
- No structured contact fields. Phone, email, address, nationality are inside `metadata JSONB` only
- No emergency contact separate from guardians
- No notes / internal comments
- No stats/KPIs storage – attendance and academic metrics must be computed on read
- No preferred language / special needs flag

### 2. API Gaps
- Missing `GET /students/:id` enriched response
- Missing `PATCH /students/:id` for editable fields
- No endpoints for guardian links, documents, medical info CRUD
- No attendance summary endpoint per student

### 3. UI/UX Gaps
- Detail page placeholder
- No edit form with validation
- No tabbed profile layout
- No guardian management inline
- No attendance/history visualization
- No Stats & KPIs dashboard

## Proposed Schema Tightening

### Option A: Extend `students` table with structured columns
Recommended minimal core columns:
```sql
ALTER TABLE students ADD COLUMN IF NOT EXISTS address text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS phone text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS email text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS nationality text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS emergency_contact_name text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS emergency_contact_phone text;
ALTER TABLE students ADD COLUMN IF NOT EXISTS special_needs text;
```
Keep `metadata JSONB` for future extensibility. Photo and document uploads deferred.

### Option B: New normalized tables

**student_contacts**
- id UUID PK
- student_id UUID FK → students
- contact_type enum ['HOME','MOBILE','EMAIL','OTHER']
- value text
- is_primary boolean
- created_at / updated_at

**student_notes**
- id UUID PK
- student_id UUID FK
- note_type text
- content text
- created_by UUID FK → users
- created_at

**student_stats_cache** *(optional materialized view)*
- student_id UUID PK FK → students
- academic_term_id UUID FK
- attendance_rate numeric
- tardy_count int
- present_days int
- absent_days int
- last_updated timestamp

**student_medical_info**
- student_id UUID PK FK → students
- allergies text[]
- chronic_conditions text
- immunization_status JSONB
- doctor_name text
- doctor_phone text
- updated_at

**student_notes**
- id UUID PK
- student_id UUID FK
- note_type text
- content text
- created_by UUID FK → users
- created_at

### Proposed API Contracts

**GET /students/:id**
```json
{
  "student_id": "...",
  "admission_number": "...",
  "full_name": "...",
  "date_of_birth": "YYYY-MM-DD",
  "gender": "M",
  "address": "...",
  "phone": "...",
  "email": "...",
  "nationality": "...",
  "emergency_contact": {...},
  "current_class": {...},
  "enrollment_history": [...],
  "guardians": [...],
  "stats_kpis": {
    "attendance_rate": 0.92,
    "present_days": 85,
    "absent_days": 7,
    "tardy_count": 3,
    "term": "Term 1 2026"
  },
  "notes": [...]
}
```

**PATCH /students/:id**
Accept partial update of editable fields with validation.

## Feature Artifacts

- Feature spec: `docs/features/FEATURE-enhanced-student-profile.md`
- Gap: `docs/gaps/GAP-student-detail-placeholder.md`
- Related gap: `docs/gaps/GAP-student-detail-comprehensive.md` [to be created]

## Prioritization Recommendation

**Phase 1 – Core profile + Stats**
- Add structured columns to `students` for address, phone, email, emergency contact
- Implement `GET /students/:id` enriched + `PATCH`
- Build detail page Overview tab with Stats & KPIs computed from attendance

**Phase 2 – Relationships**
- Guardian links UI
- Student contacts normalized table
- Notes audit trail

**Phase 3 – Advanced KPIs**
- Materialized view for student_stats_cache
- Academic results integration when tables exist

## Open Questions
- Should contact info be normalized now or stay in metadata?
- Which KPIs to surface first: attendance rate, tardiness, class engagement?
- Compute stats on read vs materialized view for performance?

## References
- `docs/backend_schema.md` → students section
- `backend/internal/services/students_service.go`
- `frontend/src/features/students/components/student-detail.tsx`
